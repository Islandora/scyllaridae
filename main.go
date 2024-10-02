package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	stomp "github.com/go-stomp/stomp/v3"
	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

var (
	config *scyllaridae.ServerConfig
)

func init() {
	var err error

	config, err = scyllaridae.ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
}

func main() {
	if len(config.QueueMiddlewares) > 0 {
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

		var wg sync.WaitGroup

		for _, middleware := range config.QueueMiddlewares {
			wg.Add(1)
			go func(middleware scyllaridae.QueueMiddleware) {
				defer wg.Done()
				messageChan := make(chan *stomp.Message, middleware.Consumers)

				// Start the specified number of worker goroutines
				for i := 0; i < middleware.Consumers; i++ {
					slog.Info("Adding consumer", "consumer", i)
					go worker(messageChan, middleware)
				}

				RecvStompMessages(middleware.QueueName, messageChan)
			}(middleware)
		}

		<-stopChan
		slog.Info("Shutting down message listener")
	} else {
		// or make this an available API ala crayfish
		http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
			slog.Info("/healthcheck", "method", r.Method, "ip", r.RemoteAddr, "proto", r.Proto)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "OK")
		})
		http.HandleFunc("/", MessageHandler)
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		slog.Info("Server listening", "port", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			panic(err)
		}
	}
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info(r.RequestURI, "method", r.Method, "ip", r.RemoteAddr, "proto", r.Proto)

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	if r.Header.Get("Apix-Ldp-Resource") == "" && r.Header.Get("X-Islandora-Event") == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Read the Alpaca message payload
	auth := ""
	if config.ForwardAuth {
		auth = r.Header.Get("Authorization")
	}
	message, err := api.DecodeAlpacaMessage(r, auth)
	if err != nil {
		slog.Error("Error decoding alpaca message", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Stream the file contents from the source URL
	req, err := http.NewRequest("GET", message.Attachment.Content.SourceURI, nil)
	if err != nil {
		slog.Error("Error creating request to source", "source", message.Attachment.Content.SourceURI, "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if config.ForwardAuth {
		req.Header.Set("Authorization", auth)
	}
	sourceResp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Error fetching source file contents", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer sourceResp.Body.Close()
	if sourceResp.StatusCode != http.StatusOK {
		slog.Error("SourceURI sent a bad status code", "code", sourceResp.StatusCode, "uri", message.Attachment.Content.SourceURI)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cmd, err := scyllaridae.BuildExecCommand(message, config)
	if err != nil {
		slog.Error("Error building command", "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	cmd.Stdin = sourceResp.Body

	// Create a buffer to stream the output of the command
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	// send stdout to the ResponseWriter stream
	cmd.Stdout = w

	slog.Info("Running command", "cmd", cmd.String())
	if err := cmd.Run(); err != nil {
		slog.Error("Error running command", "cmd", cmd.String(), "err", stdErr.String())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
}

func worker(messageChan <-chan *stomp.Message, middleware scyllaridae.QueueMiddleware) {
	for msg := range messageChan {
		handleMessage(msg, middleware)
	}
}

func RecvStompMessages(queueName string, messageChan chan<- *stomp.Message) {
	attempt := 0
	maxAttempts := 30
	for attempt = 0; attempt < maxAttempts; attempt += 1 {
		if err := connectAndSubscribe(queueName, messageChan); err != nil {
			slog.Error("resubscribing", "queue", queueName, "error", err)
			if err := retryWithExponentialBackoff(attempt, maxAttempts); err != nil {
				slog.Error("Failed subscribing after too many failed attempts", "queue", queueName, "attempts", attempt)
				return
			}
		} else {
			// Subscription was successful
			break
		}
	}
}

func connectAndSubscribe(queueName string, messageChan chan<- *stomp.Message) error {
	addr := os.Getenv("STOMP_SERVER_ADDR")
	if addr == "" {
		addr = "activemq:61613"
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("cannot connect to port", "err", err.Error())
		return err
	}
	tcpConn := c.(*net.TCPConn)

	err = tcpConn.SetKeepAlive(true)
	if err != nil {
		slog.Error("cannot set keepalive", "err", err.Error())
		return err
	}

	err = tcpConn.SetKeepAlivePeriod(10 * time.Second)
	if err != nil {
		slog.Error("cannot set keepalive period", "err", err.Error())
		return err
	}

	conn, err := stomp.Connect(tcpConn, stomp.ConnOpt.HeartBeat(10*time.Second, 0*time.Second))
	if err != nil {
		slog.Error("cannot connect to stomp server", "err", err.Error())
		return err
	}
	defer func() {
		err := conn.Disconnect()
		if err != nil {
			slog.Error("problem disconnecting from stomp server", "err", err)
		}
	}()

	sub, err := conn.Subscribe(queueName, stomp.AckAuto)
	if err != nil {
		slog.Error("cannot subscribe to queue", "queue", queueName, "err", err.Error())
		return err
	}
	defer func() {
		if !sub.Active() {
			return
		}
		err := sub.Unsubscribe()
		if err != nil {
			slog.Error("problem unsubscribing", "err", err)
		}
	}()
	slog.Info("Server subscribed to", "queue", queueName)

	for msg := range sub.C {
		if msg == nil || len(msg.Body) == 0 {
			if !sub.Active() {
				return fmt.Errorf("no longer subscribed to %s", queueName)
			}
			continue
		}
		messageChan <- msg // Send the message to the channel
	}

	return nil
}

func handleMessage(msg *stomp.Message, middleware scyllaridae.QueueMiddleware) {
	req, err := http.NewRequest("GET", middleware.Url, nil)
	if err != nil {
		slog.Error("Error creating HTTP request", "url", middleware.Url, "err", err)
		return
	}
	req.Header.Set("X-Islandora-Event", base64.StdEncoding.EncodeToString(msg.Body))
	islandoraMessage, err := api.DecodeEventMessage(msg.Body)
	if err != nil {
		slog.Error("Unable to decode event message", "err", err)
		return
	}

	if middleware.ForwardAuth {
		auth := msg.Header.Get("Authorization")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending HTTP GET request", "url", middleware.Url, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 299 {
		slog.Error("Failed to deliver message", "url", middleware.Url, "status", resp.StatusCode)
		return
	}

	// Create a pipe to stream the data from resp.Body to the PUT request body
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	putReq, err := http.NewRequest("PUT", islandoraMessage.Attachment.Content.DestinationURI, pr)
	if err != nil {
		slog.Error("Error creating HTTP PUT request", "url", islandoraMessage.Attachment.Content.DestinationURI, "err", err)
		return
	}

	_, err = io.Copy(pw, resp.Body)
	if err != nil {
		slog.Error("Error copying data from GET response to pipe", "err", err)
		return
	}

	putReq.Header.Set("Authorization", msg.Header.Get("Authorization"))
	putReq.Header.Set("Content-Type", islandoraMessage.Attachment.Content.DestinationMimeType)
	putReq.Header.Set("Content-Location", islandoraMessage.Attachment.Content.FileUploadURI)

	// Send the PUT request
	putResp, err := client.Do(putReq)
	if err != nil {
		slog.Error("Error sending HTTP PUT request", "url", islandoraMessage.Attachment.Content.DestinationURI, "err", err)
		return
	}
	defer putResp.Body.Close()

	if putResp.StatusCode >= 299 {
		slog.Error("Failed to PUT data", "url", islandoraMessage.Attachment.Content.DestinationURI, "status", putResp.StatusCode)
	}
}

func retryWithExponentialBackoff(attempt int, maxAttempts int) error {
	if attempt >= maxAttempts {
		return fmt.Errorf("maximum retry attempts reached")
	}
	wait := time.Duration(rand.Intn(1<<attempt)) * time.Second
	time.Sleep(wait)
	return nil
}

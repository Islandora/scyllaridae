package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
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
	// either subscribe to activemq directly
	if config.QueueName != "" {
		subscribed := make(chan bool)
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

		go RecvStompMessages(config.QueueName, subscribed)

		select {
		case <-subscribed:
			slog.Info("Subscription to queue successful")
		case <-stopChan:
			slog.Info("Received stop signal, exiting")
			return
		}

		<-stopChan
		slog.Info("Shutting down message listener")
	} else {
		// or make this an available API ala crayfish
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
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

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

	// build a command to run that we will pipe the stdin stream into
	cmdArgs := map[string]string{
		"sourceMimeType":      message.Attachment.Content.SourceMimeType,
		"destinationMimeType": message.Attachment.Content.DestinationMimeType,
		"addtlArgs":           message.Attachment.Content.Args,
		"target":              "",
	}
	cmd, err := scyllaridae.BuildExecCommand(cmdArgs, config)
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

func RecvStompMessages(queueName string, subscribed chan bool) {
	defer close(subscribed)
	attempt := 0
	maxAttempts := 30
	for attempt = 0; attempt < maxAttempts; attempt += 1 {
		if err := connectAndSubscribe(queueName, subscribed); err != nil {
			slog.Error("resubscribing", "error", err)
			if err := retryWithExponentialBackoff(attempt, maxAttempts); err != nil {
				slog.Error("Failed subscribing after too many failed attempts", "attempts", attempt)
				return
			}
		}
	}
}

func connectAndSubscribe(queueName string, subscribed chan bool) error {
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
	subscribed <- true

	for msg := range sub.C {
		if msg == nil || len(msg.Body) == 0 {
			// if the subscription isn't active return so we can try reconnecting
			if !sub.Active() {
				return fmt.Errorf("no longer subscribed to %s", queueName)
			}
			// else just try reading again. There's probably just no messages in the queue
			continue
		}
		handleStompMessage(msg)
	}

	return nil
}

func handleStompMessage(msg *stomp.Message) {
	message, err := api.DecodeEventMessage(msg.Body)
	if err != nil {
		slog.Error("could not read the event message", "err", err, "msg", string(msg.Body))
		return
	}

	cmdArgs := map[string]string{
		"sourceMimeType":      message.Attachment.Content.SourceMimeType,
		"destinationMimeType": message.Attachment.Content.DestinationMimeType,
		"addtlArgs":           message.Attachment.Content.Args,
		"target":              message.Target,
	}
	for _, u := range message.Object.URL {
		if u.Rel == "canonical" {
			cmdArgs["canonical"] = u.Href
			break
		}
	}
	cmd, err := scyllaridae.BuildExecCommand(cmdArgs, config)
	if err != nil {
		slog.Error("Error building command", "err", err)
		return
	}

	runCommand(cmd)
}

func runCommand(cmd *exec.Cmd) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("error creating stdout pipe", "err", err)
		return
	}
	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			slog.Info("cmd output", "stdout", scanner.Text())
		}
	}()

	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr
	if err := cmd.Start(); err != nil {
		slog.Error("Error starting command", "cmd", cmd.String(), "err", stdErr.String())
		return
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("command finished with error", "err", stdErr.String())
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

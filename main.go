package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"

	stomp "github.com/go-stomp/stomp/v3"
	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

var (
	config  *scyllaridae.ServerConfig
	options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
		//	stomp.ConnOpt.Login("guest", "guest"),
		stomp.ConnOpt.Host("/"),
	}
	stop = make(chan bool)
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
	subscribed := make(chan bool)
	go RecvMessages(subscribed)
	<-subscribed

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
	cmd, err := scyllaridae.BuildExecCommand(message.Attachment.Content.SourceMimeType, message.Attachment.Content.DestinationMimeType, message.Attachment.Content.Args, config)
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

func RecvMessages(subscribed chan bool) {
	defer func() {
		stop <- true
	}()

	conn, err := stomp.Dial("tcp", "activemq:61613", options...)

	if err != nil {
		slog.Error("cannot connect to server", "err", err.Error())
		return
	}

	queueName := "islandora-cache-warmer"
	sub, err := conn.Subscribe(queueName, stomp.AckAuto)
	if err != nil {
		slog.Error("cannot subscribe to", queueName, err.Error())
		return
	}
	close(subscribed)
	slog.Info("Server subscriber to", "queue", queueName)

	for i := 1; i <= 10; i++ {
		msg := <-sub.C
		message, err := api.DecodeEventMessage(msg.Body)
		if err != nil {
			slog.Info("could not read the event message", "err", err, "msg", string(msg.Body))
		}
		cmd, err := scyllaridae.BuildExecCommand(message.Attachment.Content.SourceMimeType, message.Attachment.Content.DestinationMimeType, message.Attachment.Content.Args, config)
		if err != nil {
			slog.Error("Error building command", "err", err)
			return
		}

		// Create a buffer to stream the output of the command
		var stdErr bytes.Buffer
		cmd.Stderr = &stdErr

		slog.Info("Running command", "cmd", cmd.String())
		if err := cmd.Run(); err != nil {
			slog.Error("Error running command", "cmd", cmd.String(), "err", stdErr.String())
			return
		}

	}
}

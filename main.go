package main

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

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
		go RecvStompMessages(config.QueueName, subscribed)
		<-subscribed

		// wait for messages
		stop := make(chan os.Signal, 1)
		<-stop
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
	var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
		//	stomp.ConnOpt.Login("guest", "guest"),
		stomp.ConnOpt.Host("/"),
	}

	addr := os.Getenv("STOMP_SERVER_ADDR")
	if addr == "" {
		addr = "activemq:61613"
	}
	conn, err := stomp.Dial("tcp", addr, options...)

	if err != nil {
		slog.Error("cannot connect to server", "err", err.Error())
		return
	}

	sub, err := conn.Subscribe(queueName, stomp.AckAuto)
	if err != nil {
		slog.Error("cannot subscribe to", queueName, err.Error())
		return
	}
	slog.Info("Server subscribed to", "queue", queueName)

	for msg := range sub.C {
		if msg == nil {
			break
		}

		message, err := api.DecodeEventMessage(msg.Body)
		if err != nil {
			slog.Error("could not read the event message", "err", err, "msg", string(msg.Body))
			continue
		}
		cmdArgs := map[string]string{
			"sourceMimeType":      message.Attachment.Content.SourceMimeType,
			"destinationMimeType": message.Attachment.Content.DestinationMimeType,
			"addtlArgs":           message.Attachment.Content.Args,
			"target":              message.Target,
		}

		cmd, err := scyllaridae.BuildExecCommand(cmdArgs, config)
		if err != nil {
			slog.Error("Error building command", "err", err)
			continue
		}

		// log stdout for the command as it prints
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			slog.Error("error creating stdout pipe", "err", err)
			continue
		}

		// Create a buffer to stream the error output of the command
		var stdErr bytes.Buffer
		cmd.Stderr = &stdErr
		messageID := msg.Header.Get("message-id")

		slog.Info("Running command", "message-id", messageID, "cmd", cmd.String())
		if err := cmd.Start(); err != nil {
			slog.Error("Error starting command", "cmd", cmd.String(), "err", stdErr.String())
		}

		go func(cmd *exec.Cmd, stdout io.ReadCloser, messageID string) {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				slog.Info("cmd output", "message-id", messageID, "stdout", scanner.Text())
			}

			if err := cmd.Wait(); err != nil {
				slog.Error("command finished with error", "message-id", messageID, "err", stdErr.String())
			}
			slog.Info("Great success!")
		}(cmd, stdout, messageID)
		if err := msg.Conn.Ack(msg); err != nil {
			slog.Error("could not ack msg", "message-id", messageID, "err", stdErr.String())
		}
	}
}

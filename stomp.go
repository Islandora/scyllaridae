package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
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

func runStompSubscribers(config *scyllaridae.ServerConfig) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	for _, middleware := range config.QueueMiddlewares {
		wg.Add(1)
		go func(middleware scyllaridae.QueueMiddleware) {
			defer wg.Done()

			for {
				select {
				case <-stopChan:
					slog.Info("Stopping subscriber for queue", "queue", middleware.QueueName)
					return
				default:
					// Process one message at a time
					err := RecvAndProcessMessage(middleware.QueueName, middleware)
					if err != nil {
						slog.Error("Error processing message", "queue", middleware.QueueName, "error", err)
					}
				}
			}
		}(middleware)
	}

	// Wait for a termination signal
	<-stopChan
	slog.Info("Shutting down message listeners")

	// Wait for all subscribers to gracefully stop
	wg.Wait()
}

func RecvAndProcessMessage(queueName string, middleware scyllaridae.QueueMiddleware) error {
	addr := os.Getenv("STOMP_SERVER_ADDR")
	if addr == "" {
		addr = "activemq:61613"
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Cannot connect to port", "err", err.Error())
		return err
	}
	tcpConn := c.(*net.TCPConn)

	err = tcpConn.SetKeepAlive(true)
	if err != nil {
		slog.Error("Cannot set keepalive", "err", err.Error())
		return err
	}

	err = tcpConn.SetKeepAlivePeriod(10 * time.Second)
	if err != nil {
		slog.Error("Cannot set keepalive period", "err", err.Error())
		return err
	}

	conn, err := stomp.Connect(tcpConn, stomp.ConnOpt.HeartBeat(10*time.Second, 0*time.Second))
	if err != nil {
		slog.Error("Cannot connect to STOMP server", "err", err.Error())
		return err
	}
	defer func() {
		err := conn.Disconnect()
		if err != nil {
			slog.Error("Problem disconnecting from STOMP server", "err", err)
		}
	}()
	sub, err := conn.Subscribe(queueName, stomp.AckClient)
	if err != nil {
		slog.Error("Cannot subscribe to queue", "queue", queueName, "err", err.Error())
		return err
	}
	defer func() {
		if !sub.Active() {
			return
		}
		err := sub.Unsubscribe()
		if err != nil {
			slog.Error("Problem unsubscribing", "err", err)
		}
	}()
	slog.Info("Subscribed to queue", "queue", queueName)

	// Process one message at a time
	for {
		msg := <-sub.C // Blocking read for one message
		if msg == nil || len(msg.Body) == 0 {
			if !sub.Active() {
				return fmt.Errorf("no longer subscribed to %s", queueName)
			}
			continue
		}

		// Process the message
		handleMessage(msg, middleware)

		// Acknowledge the message after successful processing
		err := msg.Conn.Ack(msg)
		if err != nil {
			slog.Error("Failed to acknowledge message", "queue", queueName, "error", err)
		}
	}
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

	if middleware.NoPut {
		return
	}

	putReq, err := http.NewRequest("PUT", islandoraMessage.Attachment.Content.DestinationURI, resp.Body)
	if err != nil {
		slog.Error("Error creating HTTP PUT request", "url", islandoraMessage.Attachment.Content.DestinationURI, "err", err)
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
	} else {
		slog.Info("Successfully PUT data to", "url", islandoraMessage.Attachment.Content.DestinationURI, "status", putResp.StatusCode)
	}
}

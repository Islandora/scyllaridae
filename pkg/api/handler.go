package api

// MessageHandler is an interface for custom message handling.
// Services that need to bypass the command-execution pipeline
// (e.g., fcrepo-indexer, blazegraph-indexer) implement this
// interface and set it on ServerConfig.CustomHandler.
type MessageHandler interface {
	Handle(payload Payload, auth string) (statusCode int, body []byte, contentType string, err error)
}

package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Payload defines the structure of the JSON payload received by the server.
//
// swagger:model Payload
type Payload struct {
	Actor         Actor      `json:"actor" description:"Details of the actor performing the action"`
	Object        Object     `json:"object" description:"Contains details about the object of the action"`
	Attachment    Attachment `json:"attachment" description:"Holds additional data related to the action"`
	Target        string     `json:"target" description:"Target for the payload"`
	Type          string     `json:"type" description:"Type of the payload"`
	Summary       string     `json:"summary" description:"Summary of the payload"`
	Authorization string     `json:"authorization" description:"The Authorization HTTP header"`
}

// Actor represents an entity performing an action.
//
// swagger:model Actor
type Actor struct {
	ID string `json:"id" description:"Unique identifier for the actor"`
}

// Object contains details about the object of the action.
//
// swagger:model Object
type Object struct {
	ID           string `json:"id" description:"Unique identifier for the object"`
	URL          []Link `json:"url" description:"List of hyperlinks related to the object"`
	IsNewVersion bool   `json:"isNewVersion" description:"Indicates if this is a new version of the object"`
}

// Link describes a hyperlink related to the object.
//
// swagger:model Link
type Link struct {
	Name      string `json:"name" description:"Name of the link"`
	Type      string `json:"type" description:"Type of the link"`
	Href      string `json:"href" description:"Hyperlink reference URL"`
	MediaType string `json:"mediaType" description:"Media type of the linked resource"`
	Rel       string `json:"rel" description:"Relationship type of the link"`
}

// Attachment holds additional data related to the action.
//
// swagger:model Attachment
type Attachment struct {
	Type      string  `json:"type" description:"Type of the attachment"`
	Content   Content `json:"content" description:"Content details within the attachment"`
	MediaType string  `json:"mediaType" description:"Media type of the attachment"`
}

// Content describes specific content details in an attachment.
//
// swagger:model Content
type Content struct {
	SourceMimeType      string `json:"source_mimetype,omitempty" description:"MIME type of the source URI"`
	DestinationMimeType string `json:"mimetype" description:"MIME type of the derivative being created"`
	Args                string `json:"args" description:"Arguments used or applicable to the content"`
	SourceURI           string `json:"source_uri" description:"Source URI from which the content is fetched"`
	SourceField         string `json:"source_field" description:"Source field from which the media is fetched"`
	DestinationURI      string `json:"destination_uri" description:"Destination URI to where the content is delivered"`
	FileUploadURI       string `json:"file_upload_uri" description:"File upload URI for uploading the content"`
}

// DecodeEventMessage decodes an event message sent by Islandora directly from ActiveMQ.
// It parses the JSON message into a Payload structure.
func DecodeEventMessage(msg []byte) (Payload, error) {
	var p Payload

	if err := json.Unmarshal(msg, &p); err != nil {
		return Payload{}, err
	}

	return p, nil
}

// DecodeAlpacaMessage decodes an event message transformed by Alpaca from HTTP headers.
// It reads the X-Islandora-Event header (base64-encoded JSON) or constructs a Payload from
// individual HTTP headers (Apix-Ldp-Resource, Accept, Content-Type, X-Islandora-Args).
func DecodeAlpacaMessage(r *http.Request, auth string) (Payload, error) {
	p := Payload{}

	p.Attachment.Content.Args = r.Header.Get("X-Islandora-Args")
	p.Attachment.Content.SourceURI = r.Header.Get("Apix-Ldp-Resource")
	p.Attachment.Content.DestinationMimeType = r.Header.Get("Accept")
	p.Attachment.Content.SourceMimeType = r.Header.Get("Content-Type")
	if p.Attachment.Content.DestinationMimeType == "" {
		p.Attachment.Content.DestinationMimeType = "text/plain"
	}
	p.Authorization = auth
	if r.Method == http.MethodPost {
		return p, nil
	}

	// if the message was sent in the event header, just read it
	message := r.Header.Get("X-Islandora-Event")
	if message != "" {
		j, err := base64.StdEncoding.DecodeString(message)
		if err != nil {
			slog.Error("Error decoding base64", "err", err)
			return p, err
		}
		err = json.Unmarshal(j, &p)
		if err != nil {
			slog.Error("Error unmarshalling event", "err", err)
			return p, err
		}
	}

	slog.Debug("Got message", "msgId", p.Object.ID, "payload.attachment", p.Attachment)
	err := p.getSourceUri(auth)
	if err != nil {
		return p, err
	}

	return p, nil
}

func (p *Payload) getSourceUri(auth string) error {
	if p.Attachment.Content.SourceURI == "" {
		return nil
	}
	slog.Debug("Fetching Content-Type HTTP header for SourceURI mime type", "msgId", p.Object.ID, "SourceURI", p.Attachment.Content.SourceURI)

	client := &http.Client{}

	req, err := http.NewRequest("HEAD", p.Attachment.Content.SourceURI, nil)
	if err != nil {
		slog.Error("Unable to create source URI request", "uri", p.Attachment.Content.SourceURI, "err", err)
		return fmt.Errorf("error creating request for %s", p.Attachment.Content.SourceURI)
	}

	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Unable to get source URI", "uri", p.Attachment.Content.SourceURI, "err", err)
		return fmt.Errorf("error issuing HEAD request on %s", p.Attachment.Content.SourceURI)
	}
	defer resp.Body.Close()

	p.Attachment.Content.SourceMimeType = resp.Header.Get("Content-Type")

	slog.Debug("Got SourceURI mime type", "msgId", p.Object.ID, "SourceMimeType", p.Attachment.Content.SourceMimeType)

	return nil
}

package api

import (
	"net/http"
)

// Payload defines the structure of the JSON payload received by the server.
//
// swagger:model Payload
type Payload struct {
	Actor      Actor      `json:"actor" description:"Details of the actor performing the action"`
	Object     Object     `json:"object" description:"Contains details about the object of the action"`
	Attachment Attachment `json:"attachment" description:"Holds additional data related to the action"`
	Type       string     `json:"type" description:"Type of the payload"`
	Summary    string     `json:"summary" description:"Summary of the payload"`
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
	MimeType       string `json:"mimetype" description:"MIME type of the content"`
	Args           string `json:"args" description:"Arguments used or applicable to the content"`
	SourceURI      string `json:"sourceUri" description:"Source URI from which the content is fetched"`
	DestinationURI string `json:"destinationUri" description:"Destination URI to where the content is delivered"`
	FileUploadURI  string `json:"fileUploadUri" description:"File upload URI for uploading the content"`
}

func DecodeAlpacaMessage(r *http.Request) (Payload, error) {
	p := Payload{}

	// set the payload based on the headers alpaca sends to this service
	p.Attachment.Content.Args = r.Header.Get("X-Islandora-Args")
	p.Attachment.Content.SourceURI = r.Header.Get("Apix-Ldp-Resource")
	mimetype := r.Header.Get("Accept")
	if mimetype == "" {
		mimetype = "text/plain"
	}
	p.Attachment.Content.MimeType = mimetype

	return p, nil
}

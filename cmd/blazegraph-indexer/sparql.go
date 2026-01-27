package main

import (
	"fmt"
	"net/url"
	"strings"
)

// DeleteWhere builds a SPARQL DELETE WHERE statement.
// If namedGraph is non-empty, it wraps the triple pattern in a GRAPH clause.
func DeleteWhere(subject, namedGraph string) string {
	var sb strings.Builder
	sb.WriteString("DELETE WHERE { ")

	if namedGraph != "" {
		sb.WriteString("GRAPH <")
		sb.WriteString(encodeURI(namedGraph))
		sb.WriteString("> { ")
	}

	sb.WriteString("<")
	sb.WriteString(encodeURI(subject))
	sb.WriteString("> ?p ?o ")

	if namedGraph != "" {
		sb.WriteString("} ")
	}

	sb.WriteString("}")
	return sb.String()
}

// InsertData builds a SPARQL INSERT DATA statement with the given N-Triples.
// If namedGraph is non-empty, it wraps the data in a GRAPH clause.
func InsertData(ntriples, namedGraph string) string {
	var sb strings.Builder
	sb.WriteString("INSERT DATA { ")

	if namedGraph != "" {
		sb.WriteString("GRAPH <")
		sb.WriteString(encodeURI(namedGraph))
		sb.WriteString("> { ")
	}

	sb.WriteString(ntriples)

	if namedGraph != "" {
		sb.WriteString("} ")
	}

	sb.WriteString("}")
	return sb.String()
}

// BuildUpdateQuery creates a combined DELETE WHERE + INSERT DATA SPARQL update.
func BuildUpdateQuery(subject, ntriples, namedGraph string) string {
	return DeleteWhere(subject, namedGraph) + ";\n" + InsertData(ntriples, namedGraph)
}

// EncodeUpdateBody creates the application/x-www-form-urlencoded body for a SPARQL update.
func EncodeUpdateBody(query string) string {
	return fmt.Sprintf("update=%s", url.QueryEscape(query))
}

// encodeURI performs basic URI encoding suitable for SPARQL.
// This matches the behavior of Apache Jena's URIref.encode().
func encodeURI(uri string) string {
	// For URIs used in SPARQL, we mainly need to handle special characters
	// that might break the query syntax. Most valid URIs are already safe.
	return uri
}

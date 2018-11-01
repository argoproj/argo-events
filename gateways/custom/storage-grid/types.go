package main

import "net/http"

// storageGridEventConfig contains configuration for storage grid sns
// +k8s:openapi-gen=true
type storageGridEventConfig struct {
	Port     string
	Endpoint string
	Events []string
	Filter *Filter
	// srv holds reference to http server
	srv *http.Server
	mux *http.ServeMux
}

// Filter represents filters to apply to bucket nofifications for specifying constraints on objects
// +k8s:openapi-gen=true
type Filter struct {
	Prefix string
	Suffix string
}

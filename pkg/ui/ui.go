// Package ui serves a single page to submit step counts and show the totals.
package ui

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var page []byte

// Handler serves the page. It calls the JSON API from the browser.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	}
}

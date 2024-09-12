//go:build fastlyinternalsetparseuri

package fsthttp

import (
	"net/url"
)

// SetParseRequestURI takes a function like url.ParseRequestURI to use when parsing incoming requests
// It is an experimental interface for applications that want to relax restrictions on url parsing
// It should generally not be needed, and is likely to change, so please avoid unless absolutely necessary
func SetParseRequestURI(parseRequestURI func(string)(*url.URL, error)) {
	_parseRequestURI = parseRequestURI
}

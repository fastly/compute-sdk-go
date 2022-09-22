package fsthttp

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

// Limits handles HTTP limits
var Limits = limits{}

type limits struct {
}

// MaxHeaderNameLen gets the header name limit
func (limits) MaxHeaderNameLen() int {
	return fastly.MaxHeaderNameLen
}

// SetMaxHeaderNameLen sets the header name limit
func (limits) SetMaxHeaderNameLen(len int) {
	fastly.MaxHeaderNameLen = len
}

// MaxHeaderValueLen gets the header value limit
func (limits) MaxHeaderValueLen() int {
	return fastly.MaxHeaderValueLen
}

// SetMaxHeaderValueLen sets the header value limit
func (limits) SetMaxHeaderValueLen(len int) {
	fastly.MaxHeaderValueLen = len
}

// MaxMethodLen gets the request method limit
func (limits) MaxMethodLen() int {
	return fastly.MaxMethodLen
}

// SetMaxMethodLen sets the request method limit
func (limits) SetMaxMethodLen(len int) {
	fastly.MaxMethodLen = len
}

// MaxURLLen gets the request URL limit
func (limits) MaxURLLen() int {
	return fastly.MaxURLLen
}

// SetMaxURLLen sets the request URL limit
func (limits) SetMaxURLLen(len int) {
	fastly.MaxURLLen = len
}

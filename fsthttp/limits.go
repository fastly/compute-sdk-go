package fsthttp

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

// Limits handles HTTP limits
var Limits = limits{}

type limits struct {
}

// GetMaxHeaderNameLen gets the header name limit
func (limits) GetMaxHeaderNameLen() int {
	return fastly.MaxHeaderNameLen
}

// SetMaxHeaderNameLen sets the header name limit
func (limits) SetMaxHeaderNameLen(len int) {
	fastly.MaxHeaderNameLen = len
}

// GetMaxHeaderValueLen gets the header value limit
func (limits) GetMaxHeaderValueLen() int {
	return fastly.MaxHeaderValueLen
}

// SetMaxHeaderValueLen sets the header value limit
func (limits) SetMaxHeaderValueLen(len int) {
	fastly.MaxHeaderValueLen = len
}

// GetMaxMethodLen gets the request method limit
func (limits) GetMaxMethodLen() int {
	return fastly.MaxMethodLen
}

// SetMaxMethodLen sets the request method limit
func (limits) SetMaxMethodLen(len int) {
	fastly.MaxMethodLen = len
}

// GetMaxURLLen gets the request URL limit
func (limits) GetMaxURLLen() int {
	return fastly.MaxURLLen
}

// SetMaxURLLen sets the request URL limit
func (limits) SetMaxURLLen(len int) {
	fastly.MaxURLLen = len
}

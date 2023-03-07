//go:build fastlyinternaldebug

package fsthttp

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

func (req *Request) ConstructABIRequest() error {
	return req.constructABIRequest()
}

func (req *Request) ABI() (*fastly.HTTPRequest, *fastly.HTTPBody) {
	return req.abi.req, req.abi.body
}

//go:build fastlyinternaldebug

package fsthttp

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

func (req *Request) ConstructABIRequest() error {
	if err := req.constructABIRequest(); err != nil {
		return err
	}

	return req.setABIRequestOptions()
}

func (req *Request) ABI() (*fastly.HTTPRequest, *fastly.HTTPBody) {
	return req.abi.req, req.abi.body
}

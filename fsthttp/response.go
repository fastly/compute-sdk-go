// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// ResponseLimits are the limits for the components of an HTTP response.
var ResponseLimits = Limits{
	maxHeaderNameLen:  fastly.DefaultLargeBufLen,
	maxHeaderValueLen: fastly.DefaultLargeBufLen,
}

// Response to an outgoing HTTP request made by this server.
type Response struct {
	// Request associated with the response.
	Request *Request

	// Backend that served the response.
	Backend string

	// StatusCode of the response.
	StatusCode int

	// Header received with the response.
	Header Header

	// Body of the response.
	Body io.ReadCloser

	// BackendAddrIP is the ip address of the server that sent the response.
	BackendAddrIP net.IP

	// BackendAddrPort is the port of the server that sent the response.
	BackendAddrPort uint16
}

// Cookies parses and returns the cookies set in the Set-Cookie headers.
func (resp *Response) Cookies() []*Cookie {
	return readSetCookies(resp.Header)
}

func newResponse(req *Request, backend string, abiResp *fastly.HTTPResponse, abiBody *fastly.HTTPBody) (*Response, error) {
	code, err := abiResp.GetStatusCode()
	if err != nil {
		return nil, fmt.Errorf("status code: %w", err)
	}

	header := NewHeader()
	keys := abiResp.GetHeaderNames(ResponseLimits.maxHeaderNameLen)
	for keys.Next() {
		k := string(keys.Bytes())
		vals := abiResp.GetHeaderValues(k, ResponseLimits.maxHeaderValueLen)
		for vals.Next() {
			v := string(vals.Bytes())
			header.Add(k, v)
		}
		if err := vals.Err(); err != nil {
			return nil, fmt.Errorf("read header key %q: %w", k, err)
		}
	}
	if err := keys.Err(); err != nil {
		return nil, fmt.Errorf("read header keys: %w", err)
	}

	addr, err := abiResp.GetAddrDestIP()
	if err != nil {
		return nil, fmt.Errorf("get addr dest ip: %w", err)
	}

	port, err := abiResp.GetAddrDestPort()
	if err != nil {
		return nil, fmt.Errorf("get addr dest port: %w", err)
	}

	return &Response{
		Request:         req,
		Backend:         backend,
		StatusCode:      code,
		Header:          header,
		Body:            abiBody,
		BackendAddrIP:   addr,
		BackendAddrPort: port,
	}, nil
}

// ResponseWriter is used to respond to client requests.
type ResponseWriter interface {
	// Header returns the headers that will be sent by WriteHeader.
	// Changing the returned headers after a call to WriteHeader has no effect.
	Header() Header

	// WriteHeader initiates the response to the client by sending an HTTP
	// response preamble with the provided status code, and all of the response
	// headers collected by Header. If WriteHeader is not called explicitly,
	// the first call to Write or Close will trigger an implicit call to
	// WriteHeader with a code of 200. After the first call to WriteHeader,
	// subsequent calls have no effect.
	WriteHeader(code int)

	// Write the data to the connection as part of an HTTP reply.
	//
	// If WriteHeader has not yet been called, Write calls WriteHeader(200)
	// before writing the data. Unlike the ResponseWriter in net/http, Write
	// will not automatically add Content-Type or Content-Length headers.
	Write(p []byte) (int, error)

	// Close the response to the client. Close must be called to ensure the
	// response has been fully written to the client.
	//
	// If WriteHeader has not yet been called, Close calls WriteHeader(200)
	// before closing the response. Once closed, a ResponseWriter is
	// invalidated, and may no longer be accessed or used in any way.
	Close() error

	// SetManualFramingMode controls how the framing headers
	// (Content-Length/Transfer-Encoding) are set for a response.
	//
	// If set to true, the response uses the exact framing headers
	// set in the message.  If set to false, or set to true and
	// the framing is invalid, the framing headers are based on the
	// message body, and any framing headers already set in the message are
	// discarded.
	//
	// To have an effect on the response, this must be called before any call to Write() or WriteHeader().
	SetManualFramingMode(bool)

	// Append a body onto the end of this response. Will fail if passed anything other than a Response's Body field.
	// This operation is performed in amortized constant time, and so should always be preferred to directly copying a body with io.Copy.
	Append(other io.ReadCloser) error
}

type responseWriter struct {
	once              sync.Once
	header            Header
	abiResp           *fastly.HTTPResponse
	abiBody           *fastly.HTTPBody
	ManualFramingMode bool
}

func newResponseWriter() (*responseWriter, error) {
	abiResp, err := fastly.NewHTTPResponse()
	if err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	abiRespBody, err := fastly.NewHTTPBody()
	if err != nil {
		return nil, fmt.Errorf("create response body: %w", err)
	}

	return &responseWriter{
		header:  NewHeader(),
		abiResp: abiResp,
		abiBody: abiRespBody,
	}, nil
}

func (resp *responseWriter) Header() Header {
	return resp.header
}

func (resp *responseWriter) WriteHeader(code int) {
	resp.once.Do(func() {
		resp.abiResp.SetFramingHeadersMode(resp.ManualFramingMode)
		resp.abiResp.SetStatusCode(code)
		for _, key := range resp.header.Keys() {
			resp.abiResp.SetHeaderValues(key, resp.header.Values(key))
		}
		resp.abiResp.SendDownstream(resp.abiBody, true)
	})
}

func (resp *responseWriter) Write(p []byte) (int, error) {
	resp.WriteHeader(200)
	return resp.abiBody.Write(p)
}

func (resp *responseWriter) Close() error {
	resp.WriteHeader(200)
	return resp.abiBody.Close()
}

func (resp *responseWriter) SetManualFramingMode(mode bool) {
	resp.ManualFramingMode = mode
}

func (resp *responseWriter) Append(other io.ReadCloser) error {
	otherAbiBody, ok := other.(*fastly.HTTPBody)
	if !ok {
		return fmt.Errorf("non-Response Body passed to ResponseWriter.Append")
	}
	resp.abiBody.Append(otherAbiBody)
	return nil
}

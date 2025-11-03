// Deprecated: Use fsthttp.Request.Handoff* methods instead.
package handoff

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

// Websocket passes the WebSocket directly to a backend.
//
// This can only be used on services that have the WebSockets feature
// enabled and on requests that are valid WebSocket requests.  The sending
// completes in the background.
//
// Once this method has been called, no other response can be sent to this
// request, and the application can exit without affecting the send.
func Websocket(backend string) error {
	return fastly.HandoffWebsocket(backend)

}

// Fanout passes the request through the Fanout GRIP proxy and on to
// a backend.
//
// This can only be used on services that have the Fanout feature enabled.
//
// The sending completes in the background. Once this method has been
// called, no other response can be sent to this request, and the
// application can exit without affecting the send.
func Fanout(backend string) error {
	return fastly.HandoffFanout(backend)
}

package fsthttp

import (
	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// Limits handles HTTP limits.
//
// Deprecated: limits are enforced at a different level within the platform. The
// values returned by the Max.+Len() methods have been preserved, but are no
// longer meaningful. Further information can be found at
// https://docs.fastly.com/products/network-services-resource-limits#request-and-response-limits
type Limits struct{}

// MaxHeaderNameLen gets the header name limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxHeaderNameLen() int {
	return fastly.DefaultLargeBufLen
}

// SetMaxHeaderNameLen sets the header name limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) SetMaxHeaderNameLen(_ int) {
}

// MaxHeaderValueLen gets the header value limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxHeaderValueLen() int {
	return fastly.DefaultLargeBufLen
}

// SetMaxHeaderValueLen sets the header value limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) SetMaxHeaderValueLen(_ int) {
}

// MaxMethodLen gets the request method limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxMethodLen() int {
	return fastly.DefaultMediumBufLen
}

// SetMaxMethodLen sets the request method limit
// Deprecated: the limit is not reset, buffer sizing is adaptive.
func (limits *Limits) SetMaxMethodLen(_ int) {
}

// MaxURLLen gets the request URL limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxURLLen() int {
	return fastly.DefaultLargeBufLen
}

// SetMaxURLLen sets the request URL limit
// Deprecated: the limit is not reset, buffer sizing is adaptive.
func (limits *Limits) SetMaxURLLen(_ int) {
}

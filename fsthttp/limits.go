package fsthttp

import "math"

// Limits handles HTTP limits
// Deprecated: limits are enforced at a different level within the platform.
type Limits struct{}

// MaxHeaderNameLen gets the header name limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxHeaderNameLen() int {
	return math.MaxInt
}

// SetMaxHeaderNameLen sets the header name limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) SetMaxHeaderNameLen(_ int) {
}

// MaxHeaderValueLen gets the header value limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxHeaderValueLen() int {
	return math.MaxInt
}

// SetMaxHeaderValueLen sets the header value limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) SetMaxHeaderValueLen(_ int) {
}

// MaxMethodLen gets the request method limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxMethodLen() int {
	return math.MaxInt
}

// SetMaxMethodLen sets the request method limit
// Deprecated: the limit is not reset, buffer sizing is adaptive.
func (limits *Limits) SetMaxMethodLen(_ int) {
}

// MaxURLLen gets the request URL limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxURLLen() int {
	return math.MaxInt
}

// SetMaxURLLen sets the request URL limit
// Deprecated: the limit is not reset, buffer sizing is adaptive.
func (limits *Limits) SetMaxURLLen(_ int) {
}

package fsthttp

// Limits handles HTTP limits
type Limits struct {
	maxHeaderNameLen  int
	maxHeaderValueLen int
	maxMethodLen      int
	maxURLLen         int
}

// MaxHeaderNameLen gets the header name limit
func (limits *Limits) MaxHeaderNameLen() int {
	return limits.maxHeaderNameLen
}

// SetMaxHeaderNameLen sets the header name limit
func (limits *Limits) SetMaxHeaderNameLen(len int) {
	limits.maxHeaderNameLen = len
}

// MaxHeaderValueLen gets the header value limit
func (limits *Limits) MaxHeaderValueLen() int {
	return limits.maxHeaderValueLen
}

// SetMaxHeaderValueLen sets the header value limit
func (limits *Limits) SetMaxHeaderValueLen(len int) {
	limits.maxHeaderValueLen = len
}

// MaxMethodLen gets the request method limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxMethodLen() int {
	return limits.maxMethodLen
}

// SetMaxMethodLen sets the request method limit
// Deprecated: the limit is not reset, buffer sizing is adaptive.
func (limits *Limits) SetMaxMethodLen(_ int) {
}

// MaxURLLen gets the request URL limit
// Deprecated: the limit is not enforced, buffer sizing is adaptive.
func (limits *Limits) MaxURLLen() int {
	return limits.maxURLLen
}

// SetMaxURLLen sets the request URL limit
// Deprecated: the limit is not reset, buffer sizing is adaptive.
func (limits *Limits) SetMaxURLLen(_ int) {
}

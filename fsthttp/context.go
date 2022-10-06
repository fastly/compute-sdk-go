package fsthttp

import "context"

type (
	requestContextKey        struct{}
	responseWriterContextKey struct{}
	responseContextKey       struct{}
)

// RequestFromContext returns the fsthttp.Request associated with the
// context, if any.
func RequestFromContext(ctx context.Context) *Request {
	req, _ := ctx.Value(requestContextKey{}).(*Request)
	return req
}

func contextWithRequest(ctx context.Context, req *Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, req)
}

// ResponseWriterFromContext returns the fsthttp.ResponseWriter associated
// with the context, if any.
func ResponseWriterFromContext(ctx context.Context) ResponseWriter {
	w, _ := ctx.Value(responseWriterContextKey{}).(ResponseWriter)
	return w
}

func contextWithResponseWriter(ctx context.Context, w ResponseWriter) context.Context {
	return context.WithValue(ctx, responseWriterContextKey{}, w)
}

// ResponseFromContext returns the fsthttp.Response associated with the
// context, if any.
func ResponseFromContext(ctx context.Context) *Response {
	resp, _ := ctx.Value(responseContextKey{}).(*Response)
	return resp
}

func contextWithResponse(ctx context.Context, resp *Response) context.Context {
	return context.WithValue(ctx, responseContextKey{}, resp)
}

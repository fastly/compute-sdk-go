//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package fastly

import "github.com/fastly/compute-sdk-go/internal/abi/prim"

// (module $fastly_image_optimizer

// witx:
//
//	(@interface func (export "transform_image_optimizer_request")
//	    (param $origin_image_request $request_handle)
//	    (param $origin_image_request_body $body_handle)
//	    (param $origin_image_request_backend string)
//	    (param $io_transform_config_mask $image_optimizer_transform_config_options)
//	    (param $io_transform_configuration (@witx pointer $image_optimizer_transform_config))
//	    (param $io_error_detail (@witx pointer $image_optimizer_error_detail))
//	    (result $err (expected
//	            (tuple $response_handle $body_handle)
//	            (error $fastly_status)))
//	)
//
//go:wasmimport fastly_image_optimizer transform_image_optimizer_request
//go:noescape
func fastlyImageOptimizerTransformImageOptimizerRequest(
	h requestHandle,
	body bodyHandle,
	backendData prim.Pointer[prim.U8], backendLen prim.Usize,
	mask imageOptimizerTransformConfigOptionsMask,
	query prim.Pointer[imageOptimizerTransformConfig],
	errorDetails prim.Pointer[imageOptimizerErrorDetail],
	resp prim.Pointer[responseHandle],
	respBody prim.Pointer[bodyHandle],
) FastlyStatus

func (r *HTTPRequest) SendToImageOpto(requestBody *HTTPBody, backend, query string) (response *HTTPResponse, responseBody *HTTPBody, err error) {
	var (
		respHandle = invalidResponseHandle
		bodyHandle = invalidBodyHandle
	)

	var errDetail imageOptimizerErrorDetail
	backendBuffer := prim.NewReadBufferFromString(backend).Wstring()
	queryBuffer := prim.NewReadBufferFromString(query)

	config := imageOptimizerTransformConfig{
		sdkClaimsOptsPtr: prim.ToPointer(queryBuffer.Char8Pointer()),
		sdkClaimsOptsLen: prim.U32(queryBuffer.Len()),
	}

	if err := fastlyImageOptimizerTransformImageOptimizerRequest(
		r.h,
		requestBody.h,
		backendBuffer.Data, backendBuffer.Len,
		imageOptimizerTransformConfigOptionsSDKClaimsOpts, // mask
		prim.ToPointer(&config),
		prim.ToPointer(&errDetail),
		prim.ToPointer(&respHandle),
		prim.ToPointer(&bodyHandle),
	).toError(); err != nil {
		// errorDetail isn't used yet
		return nil, nil, err
	}

	return &HTTPResponse{h: respHandle}, &HTTPBody{h: bodyHandle}, nil
}

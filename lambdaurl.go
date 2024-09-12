// Copyright 2023 Amazon.com, Inc. or its affiliates. All Rights Reserved.

// lambdaurl converts an http.Handler into a Lambda request handler.
// Supports Lambda Function URLs configured with buffered response mode.
// Based on https://github.com/aws/aws-lambda-go/tree/main/lambdaurl
package lambdaurl

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// RequestConvertionError is returned when the conversion of the Lambda request to an http.Request fails.
type RequestConvertionError struct {
	Cause error
}

func (e RequestConvertionError) Error() string {
	return "failed to convert Lambda request to http.Request"
}

func (e RequestConvertionError) Unwrap() error {
	return e.Cause
}

// WriteResponseError is returned when writing the response fails.
type WriteResponseError struct {
	Cause error
}

func (e WriteResponseError) Error() string {
	return "failed to write response"
}

func (e WriteResponseError) Unwrap() error {
	return e.Cause
}

type httpResponseWriter struct {
	header http.Header
	code   int
	writer io.Writer
}

func newHTTPResponseWriter(w io.Writer) httpResponseWriter {
	return httpResponseWriter{
		header: http.Header{},
		code:   http.StatusOK,
		writer: w,
	}
}

func (w *httpResponseWriter) Header() http.Header {
	return w.header
}

func (w *httpResponseWriter) Write(p []byte) (int, error) {
	b, err := w.writer.Write(p)
	if err != nil {
		return b, WriteResponseError{Cause: err}
	}
	return b, nil
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
}

type requestContextKey struct{}

// RequestFromContext returns the *events.LambdaFunctionURLRequest from a context.
func RequestFromContext(ctx context.Context) (*events.LambdaFunctionURLRequest, bool) {
	req, ok := ctx.Value(requestContextKey{}).(*events.LambdaFunctionURLRequest)
	return req, ok
}

// Wrap converts an http.Handler into a Lambda request handler.
func Wrap(handler http.Handler) func(context.Context, events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return func(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		var body io.Reader = strings.NewReader(request.Body)
		if request.IsBase64Encoded {
			body = base64.NewDecoder(base64.StdEncoding, body)
		}
		url := "https://" + request.RequestContext.DomainName + request.RawPath
		if request.RawQueryString != "" {
			url += "?" + request.RawQueryString
		}
		ctx = context.WithValue(ctx, requestContextKey{}, request)
		httpRequest, err := http.NewRequestWithContext(ctx, request.RequestContext.HTTP.Method, url, body)
		if err != nil {
			return events.LambdaFunctionURLResponse{}, RequestConvertionError{Cause: err}
		}
		httpRequest.RemoteAddr = request.RequestContext.HTTP.SourceIP
		for k, v := range request.Headers {
			httpRequest.Header.Add(k, v)
		}

		w := strings.Builder{}
		responseWriter := newHTTPResponseWriter(&w)
		handler.ServeHTTP(&responseWriter, httpRequest)

		response := events.LambdaFunctionURLResponse{
			StatusCode: responseWriter.code,
			Body:       w.String(),
		}
		response.Headers = make(map[string]string, len(responseWriter.header))
		for k, v := range responseWriter.header {
			if k == "Set-Cookie" {
				response.Cookies = v
			} else {
				response.Headers[k] = strings.Join(v, ",")
			}
		}

		return response, nil
	}
}

func Start(handler http.Handler, options ...lambda.Option) {
	lambda.StartHandlerFunc(Wrap(handler), options...)
}

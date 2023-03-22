// MIT License
//
// Copyright (c) 2023 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	json "github.com/json-iterator/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StatusCode int32

func (c StatusCode) String() string {
	switch c {
	case StatusClientException:
		return "Custom Client Exception"
	case StatusServerException:
		return "Custom Server Exception"
	}
	return http.StatusText(int(c))
}

const (
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusMethodNotAllowed    StatusCode = 405
	StatusCancel              StatusCode = 408
	StatusConflict            StatusCode = 409
	StatusPreconditionFiled   StatusCode = 412
	StatusClientException     StatusCode = 499
	StatusInternalServerError StatusCode = 500
	StatusNotImplemented      StatusCode = 501
	StatusBadGateway          StatusCode = 502
	StatusServiceUnavailable  StatusCode = 503
	StatusGatewayTimeout      StatusCode = 504
	StatusServerException     StatusCode = 599
)

// New generates a custom error.
func New(detail string, code StatusCode) *Error {
	e := &Error{
		Code:   int32(code),
		Detail: detail,
		Status: code.String(),
	}
	return e
}

func (e *Error) WithCode(code StatusCode) *Error {
	e.Code = int32(code)
	e.Status = code.String()
	return e
}

// WithCaller fills Error.Caller
func (e *Error) WithCaller() *Error {
	_, file, line, _ := runtime.Caller(1)
	if index := strings.Index(file, "/src/"); index != -1 {
		file = file[index+5:]
	}
	file = strings.Replace(file, string(filepath.Separator), "/", -1)
	e.Caller = fmt.Sprintf("%s:%d", file, line)
	return e
}

func (e Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e Error) ToGRPC() *status.Status {
	switch StatusCode(e.Code) {
	case StatusBadRequest, StatusClientException:
		return status.New(codes.InvalidArgument, e.Detail)
	case StatusUnauthorized:
		return status.New(codes.Unauthenticated, e.Detail)
	case StatusForbidden:
		return status.New(codes.PermissionDenied, e.Detail)
	case StatusNotFound, StatusMethodNotAllowed:
		return status.New(codes.NotFound, e.Detail)
	case StatusCancel, StatusGatewayTimeout:
		return status.New(codes.Canceled, e.Detail)
	case StatusConflict:
		return status.New(codes.AlreadyExists, e.Detail)
	case StatusPreconditionFiled:
		return status.New(codes.FailedPrecondition, e.Detail)
	case StatusInternalServerError, StatusServerException:
		return status.New(codes.Internal, e.Detail)
	case StatusNotImplemented:
		return status.New(codes.Unimplemented, e.Detail)
	case StatusBadGateway:
		return status.New(codes.OutOfRange, e.Detail)
	case StatusServiceUnavailable:
		return status.New(codes.Unavailable, e.Detail)
	default:
		return status.New(codes.OK, "")
	}
}

// Parse tries to parse a JSON string into an error. If that
// fails, it will set the given string as the error detail.
func Parse(err string) *Error {
	e := new(Error)
	errr := json.Unmarshal([]byte(err), e)
	if errr != nil {
		e.Detail = err
	}
	return e
}

// BadRequest generates a 400 error.
func BadRequest(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusBadRequest)
}

// Unauthorized generates a 401 error.
func Unauthorized(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusUnauthorized)
}

// Forbidden generates a 403 error.
func Forbidden(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusForbidden)
}

// NotFound generates a 404 error.
func NotFound(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusNotFound)
}

// MethodNotAllowed generates a 405 error.
func MethodNotAllowed(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusMethodNotAllowed)
}

// Cancel generates a 408 error.
func Cancel(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusCancel)
}

// Conflict generates a 409 error.
func Conflict(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusConflict)
}

// PreconditionFailed generates a 412 error.
func PreconditionFailed(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusPreconditionFiled)
}

// ClientException generates a custom client exception.
func ClientException(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusClientException)
}

// InternalServerError generates a 500 error.
func InternalServerError(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusInternalServerError)
}

// NotImplemented generates a 501 error
func NotImplemented(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusNotImplemented)
}

// BadGateway generates a 502 error
func BadGateway(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusBadGateway)
}

// ServiceUnavailable generates a 503 error
func ServiceUnavailable(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusServiceUnavailable)
}

// GatewayTimeout generates a 504 error
func GatewayTimeout(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusCancel)
}

// ServerException generates a custom server exception
func ServerException(format string, a ...interface{}) *Error {
	return New(fmt.Sprintf(format, a...), StatusServerException)
}

// Equal tries to compare errors
func Equal(err1 error, err2 error) bool {
	verr1, ok1 := err1.(*Error)
	verr2, ok2 := err2.(*Error)

	if ok1 != ok2 {
		return false
	}

	if !ok1 {
		return err1 == err2
	}

	if verr1.Code != verr2.Code {
		return false
	}

	return true
}

// FromErr try to convert go error go *Error
func FromErr(err error) *Error {
	if verr, ok := err.(*Error); ok && verr != nil {
		return verr
	}

	if err != nil {
		if se, ok := err.(interface {
			GRPCStatus() *status.Status
		}); ok {
			s := se.GRPCStatus()
			switch s.Code() {
			case codes.OK:
				return &Error{Code: 0}
			case codes.Canceled:
				return Cancel("", s.Message())
			case codes.InvalidArgument:
				return BadRequest("", s.Message())
			case codes.DeadlineExceeded:
				return GatewayTimeout("", s.Message())
			case codes.NotFound:
				return NotFound("", s.Message())
			case codes.AlreadyExists:
				return Conflict("", s.Message())
			case codes.PermissionDenied:
				return Forbidden("", s.Message())
			case codes.FailedPrecondition:
				return PreconditionFailed("", s.Message())
			case codes.Aborted:
				return Conflict("", s.Message())
			case codes.OutOfRange:
				return BadGateway("", s.Message())
			case codes.Unimplemented:
				return NotImplemented("", s.Message())
			case codes.Internal:
				return InternalServerError("", s.Message())
			case codes.Unavailable:
				return ServiceUnavailable("", s.Message())
			case codes.DataLoss:
				return InternalServerError("", s.Message())
			case codes.Unauthenticated:
				return Unauthorized("", s.Message())
			}
			return Parse(se.GRPCStatus().Message())
		}
	}

	return Parse(err.Error())
}

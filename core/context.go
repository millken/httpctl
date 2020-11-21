package core

import "io"

type Context struct {
	RequestHeader  *RequestHeader
	ResponseHeader *ResponseHeader
	ResponseBody   io.Reader
}

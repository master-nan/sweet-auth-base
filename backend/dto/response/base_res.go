package response

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminError is the HTTP error response DTO. Application failures live in internal/errors.
type AdminError struct {
	StatusCode   int    `json:"status_code"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Success      bool   `json:"success"`
}

type BufferedResponseWriter struct {
	gin.ResponseWriter
	Body         *bytes.Buffer
	MaxBodyBytes int
	Truncated    bool
}

func (w *BufferedResponseWriter) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *BufferedResponseWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *BufferedResponseWriter) capture(b []byte) {
	if w.Body == nil || len(b) == 0 {
		return
	}
	if w.MaxBodyBytes <= 0 {
		_, _ = w.Body.Write(b)
		return
	}
	remaining := w.MaxBodyBytes - w.Body.Len()
	if remaining <= 0 {
		w.Truncated = true
		return
	}
	if len(b) > remaining {
		_, _ = w.Body.Write(b[:remaining])
		w.Truncated = true
		return
	}
	_, _ = w.Body.Write(b)
}

// Response 成功返回值参数
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Code    int         `json:"code,omitempty"`
	Total   int         `json:"total"`
}

func NewResponse() *Response {
	return &Response{
		Success: true,
		Code:    http.StatusOK,
		Message: "操作成功",
	}
}

func (r *Response) SetSuccess(success bool) *Response {
	r.Success = success
	return r
}

func (r *Response) SetData(data interface{}) *Response {
	r.Data = data
	return r
}

func (r *Response) SetTotal(total int) *Response {
	r.Total = total
	return r
}

func (r *Response) SetMessage(msg string) *Response {
	r.Message = msg
	return r
}

func (r *Response) SetCode(code int) *Response {
	r.Code = code
	return r
}

type ListResult[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
}

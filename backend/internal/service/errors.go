package service

import "fmt"

// BizError 业务错误，携带 HTTP 状态码
type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string { return e.Msg }

// BadRequest 400 业务错误
func BadRequest(format string, args ...interface{}) error {
	return &BizError{Code: 400, Msg: fmt.Sprintf(format, args...)}
}

// Conflict 409 业务错误
func Conflict(format string, args ...interface{}) error {
	return &BizError{Code: 409, Msg: fmt.Sprintf(format, args...)}
}

// NotFound 404 业务错误
func NotFound(format string, args ...interface{}) error {
	return &BizError{Code: 404, Msg: fmt.Sprintf(format, args...)}
}

// ToHTTP 将 error 映射为 (httpCode, msg)
func ToHTTP(err error) (int, string) {
	if be, ok := err.(*BizError); ok {
		return be.Code, be.Msg
	}
	return 500, "服务器内部错误"
}

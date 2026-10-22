package response

import "github.com/gin-gonic/gin"

// Body 统一响应结构，与旧系统保持一致 {code, msg, data}
type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, Body{Code: 200, Msg: "success", Data: data})
}

// Fail 失败响应，httpCode 同时作为业务 code
func Fail(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, Body{Code: httpCode, Msg: msg, Data: nil})
}

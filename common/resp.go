package common

import (
	"fmt"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var debug bool

// 设置调试模式（由 web 服务初始化时调用）：调试模式下底层错误原样返回。
// 入参: d (是否调试模式)
func SetDebug(d bool) {
	debug = d
}

// ResponseType 统一响应结构。
type ResponseType[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  T      `json:"result"`
}

// 成功返回。
// 入参: c (gin 上下文), results (业务数据，取第一个)
func SuccessResponse(c *gin.Context, results ...any) {
	FullSuccessResponse(c, "成功", results...)
}

// 完整成功返回。
// 入参: c (gin 上下文), message (提示信息), results (业务数据，取第一个)
func FullSuccessResponse(c *gin.Context, message string, results ...any) {
	FullResponse(c, 0, message, results...)
}

// 错误返回（错误码 -1）。
// 入参: c (gin 上下文), message (提示信息)
func ErrorResponse(c *gin.Context, message string) {
	FullErrorResponse(c, -1, message)
}

// 使用预定义错误码返回错误。
// 入参: c (gin 上下文), code (错误码)
func ErrorResponseWithCode(c *gin.Context, code ErrorCode) {
	FullErrorResponse(c, code.Code, code.Message)
}

// 完整错误返回。
// 入参: c (gin 上下文), code (错误码), message (提示信息)
func FullErrorResponse(c *gin.Context, code int, message string) {
	FullResponse(c, code, message, nil)
}

// 完整返回。
// 入参: c (gin 上下文), code (错误码), message (提示信息), results (业务数据，取第一个)
func FullResponse(c *gin.Context, code int, message string, results ...any) {
	var result any
	if len(results) > 0 {
		result = results[0]
	}
	c.JSON(200, ResponseType[any]{
		Code:    code,
		Message: message,
		Result:  result,
	})
}

// 将底层技术错误转换为对外返回的错误信息：调试模式返回原始错误，
// 非调试模式模糊为通用业务信息并记录原始错误日志。
// 入参: err (底层真实错误), message (对外返回的通用错误信息)
// 出参: 处理后的 error
func MaskError(err error, message string) error {
	if err == nil {
		return nil
	}
	if debug {
		return err
	}
	log.Errorf("%s: %+v", message, err)
	return fmt.Errorf("%s", message)
}

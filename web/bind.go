package web

import (
	"github.com/gin-gonic/gin"
)

// 绑定请求参数到目标结构体（包装 gin 的 ShouldBind：按请求方法与 Content-Type 自动选择绑定器），
// 仅返回错误，不自动写响应、不自动中断请求（由调用方决定如何处理）。
// 入参: c (gin 上下文), dst (目标结构体指针)
// 出参: 绑定或校验错误
func Bind(c *gin.Context, dst any) error {
	return c.ShouldBind(dst)
}

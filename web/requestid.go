package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader 请求 ID 头名：请求已携带时以请求为准，缺失时生成。
	RequestIDHeader = "X-Request-Id"
	// RequestIDKey 请求 ID 在 gin context 中的键名（c.GetString(web.RequestIDKey) 取用）。
	RequestIDKey = "request_id"
)

// 请求 ID 中间件：请求已携带 X-Request-Id 时以请求为准，缺失则生成；
// 写入 gin context（键 RequestIDKey）与响应头。
// 出参: gin 中间件函数
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// 生成请求 ID：16 字节随机数的十六进制串；随机源不可用时回落到纳秒时间戳。
// 出参: 请求 ID 字符串
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

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
	// maxRequestIDLen 客户端请求 ID 允许的最大长度（超限视为非法，重新生成）。
	maxRequestIDLen = 64
)

// 请求 ID 中间件：请求已携带合法的 X-Request-Id 时以请求为准，缺失或非法则生成；
// 写入 gin context（键 RequestIDKey）与响应头。
// 出参: gin 中间件函数
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// 校验客户端携带的请求 ID：长度 1~64，仅允许字母、数字、短横线、下划线、点号。
// 客户端 ID 会进入响应头、context 与日志，白名单校验防止异常字符注入与滥用。
// 入参: id (客户端请求 ID)
// 出参: 是否合法
func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		b := id[i]
		ok := b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' ||
			b == '-' || b == '_' || b == '.'
		if !ok {
			return false
		}
	}
	return true
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

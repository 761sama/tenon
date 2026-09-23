package common

// ErrorCode 错误码配置。
type ErrorCode struct {
	Code    int
	Message string
}

var (
	CODE_MISSING_AUTH_HEADER  = ErrorCode{Code: -10001, Message: "缺少认证头"}             // 缺少认证头
	CODE_TIMESTAMP_ERROR      = ErrorCode{Code: -10002, Message: "时间戳异常或过期"}       // 时间戳异常
	CODE_INVALID_AUTH         = ErrorCode{Code: -10003, Message: "无效授权"}              // 无效授权 或 授权不存在
	CODE_NO_PERMISSION        = ErrorCode{Code: -10006, Message: "无权限"}                // 无权限
	CODE_BODY_READ_ERROR      = ErrorCode{Code: -10004, Message: "读取错误"}              // 读取异常
	CODE_DECODE_ERROR         = ErrorCode{Code: -10005, Message: "解码或解密错误"}        // 解码 或 解密异常
	CODE_RATE_LIMIT_MINUTE    = ErrorCode{Code: -10101, Message: "请求过于频繁 / 每分钟限制"} // 请求过于频繁 (每分钟限制)
	CODE_RATE_LIMIT_DAILY     = ErrorCode{Code: -10102, Message: "请求过于频繁 / 每日限制"}   // 请求过于频繁 (每日限制)
	CODE_PROGRESS_FAILD       = ErrorCode{Code: -50000, Message: "执行失败"}              // 执行失败 或 异常
	CODE_MISSING_REQUESR_ARGS = ErrorCode{Code: -50001, Message: "缺少关键参数"}          // 缺少关键参数
	CODE_INVALID_REQUEST_ARGS = ErrorCode{Code: -50002, Message: "无效请求参数"}          // 无效请求参数
)

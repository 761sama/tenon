package common

// 分页默认值。
const (
	DefaultPageSize int64 = 10  // 默认每页数量
	MaxPageSize     int64 = 100 // 每页数量上限
)

// PageRequest 分页请求。
type PageRequest struct {
	Page     int64 `form:"page"`
	PageSize int64 `form:"page_size"`
}

// PageResponse 分页响应。
type PageResponse[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"page_size"`
}

// 校验并修正分页参数：page 小于 1 时重置为 1；
// pageSize 小于等于 0 时重置为默认值，大于上限时截断为上限。
// 入参: page (页码), pageSize (每页数量)
// 出参: 修正后的页码与每页数量
func PageSizeCheck(page, pageSize int64) (int64, int64) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

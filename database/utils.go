package database

import "fmt"

// 处理不同数据库的列名引用兼容问题。
// 入参: name (列名)
// 出参: 带引用符的列名
func ColumnName(name string) string {
	mu.RLock()
	typ := dbType
	mu.RUnlock()
	if typ == "postgres" || typ == "kingbase" {
		return fmt.Sprintf(`"%s"`, name)
	}
	return fmt.Sprintf("`%s`", name)
}

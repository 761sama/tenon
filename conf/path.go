package conf

import "path/filepath"

// 将相对路径解析为基于数据目录的路径；绝对路径原样返回。
// 供调用方将日志文件、SQLite 数据库、TLS 证书等相对路径统一挂到数据目录下。
// 入参: dataDir (数据目录，空字符串表示工作目录), name (待解析的路径)
// 出参: 解析后的路径
func ResolveDataPath(dataDir, name string) string {
	if filepath.IsAbs(name) || dataDir == "" {
		return filepath.Clean(name)
	}
	return filepath.Join(dataDir, name)
}

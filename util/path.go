package util

import "path/filepath"

// 将相对路径解析为基于数据目录的路径；绝对路径原样返回。
// 入参: dataDir (数据目录), name (待解析的路径)
// 出参: 解析后的路径
func ResolveDataPath(dataDir, name string) string {
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	return filepath.Join(dataDir, name)
}

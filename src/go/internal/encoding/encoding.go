// Package encoding 提供 Windows 中文编码（GBK ↔ UTF-8）转换工具
// CGo 库（bitmaputil、robotgo）的底层 C 文件 API 需要 GBK 路径，Go 层使用 UTF-8
package encoding

import (
	"bytes"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ToGBK 将 UTF-8 字符串转为 GBK（用于传递给 CGo 库的文件路径）
func ToGBK(s string) string {
	encoder := simplifiedchinese.GBK.NewEncoder()
	encoded, err := io.ReadAll(transform.NewReader(bytes.NewReader([]byte(s)), encoder))
	if err != nil {
		return s
	}
	return string(encoded)
}

// ToUTF8 将 GBK 数据转为 UTF-8，失败或已是合法 UTF-8 时返回原数据
func ToUTF8(data []byte) ([]byte, bool) {
	// 已是合法 UTF-8，无需转换
	if utf8.Valid(data) {
		return data, false
	}
	decoder := simplifiedchinese.GBK.NewDecoder()
	decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
	if err != nil {
		return data, false
	}
	// 含 U+FFFD 说明 GBK 解码失败，返回原数据
	if bytes.ContainsRune(decoded, '\ufffd') {
		return data, false
	}
	return decoded, true
}

// ToUTF8String 将 GBK 字符串转为 UTF-8 字符串
func ToUTF8String(s string) string {
	decoded, ok := ToUTF8([]byte(s))
	if !ok {
		return s
	}
	return string(decoded)
}

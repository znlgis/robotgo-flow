package main

import "encoding/json"

// goErrorJSON 使用 json.Marshal 构造标准错误 JSON 字符串。
// 避免使用 %q，因为 Go 的转义规则与 JSON 不完全兼容。
func goErrorJSON(msg string) string {
	b, _ := json.Marshal(map[string]interface{}{"ok": false, "error": msg})
	return string(b)
}

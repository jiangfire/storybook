package service

import "encoding/json"

// ParseStringArrayJSON 将 JSON 字节序列解码为字符串数组,失败或空输入均返回 []string{}。
func ParseStringArrayJSON(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	return out
}

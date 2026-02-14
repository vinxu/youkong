package agent

import (
	"strings"

	"github.com/mozillazg/go-pinyin"
)

var pinyinArgs = pinyin.NewArgs()

func init() {
	pinyinArgs.Style = pinyin.Normal // 不带声调
}

// FuzzyMatchName 模糊匹配好友名字
// 支持：中文包含、拼音↔汉字、大小写不敏感
// query: 用户输入（可能是中文"壮"或拼音"zhuang"）
// name: 好友昵称（可能是中文"小明"或拼音"zhuang"）
func FuzzyMatchName(query, name string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	n := strings.ToLower(strings.TrimSpace(name))
	if q == "" || n == "" {
		return false
	}

	// 1. 直接包含（大小写不敏感）
	if strings.Contains(n, q) || strings.Contains(q, n) {
		return true
	}

	// 2. 将 query 转拼音，与 name 比较
	// 例: query="壮" → ["zhuang"]，name="zhuang" → match
	queryPinyin := toPinyinStr(q)
	if queryPinyin != "" && strings.Contains(n, queryPinyin) {
		return true
	}

	// 3. 将 name 转拼音，与 query 比较
	// 例: name="小明" → "xiaoming"，query="xiaoming" → match
	namePinyin := toPinyinStr(n)
	if namePinyin != "" && strings.Contains(namePinyin, q) {
		return true
	}

	// 4. 双方都转拼音后比较
	// 例: query="壮" → "zhuang"，name="壮壮" → "zhuangzhuang" → contains
	if queryPinyin != "" && namePinyin != "" && strings.Contains(namePinyin, queryPinyin) {
		return true
	}

	// 5. 拼音首字母匹配
	// 例: query="xm"，name="小明" → initials="xm" → match
	nameInitials := toPinyinInitials(n)
	if nameInitials != "" && len(q) >= 2 && strings.Contains(nameInitials, q) {
		return true
	}

	return false
}

// toPinyinStr 将中文字符串转为拼音（无声调，连续拼接）
// "小明" → "xiaoming"，"abc" → ""（非中文返回空）
func toPinyinStr(s string) string {
	result := pinyin.Pinyin(s, pinyinArgs)
	if len(result) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, py := range result {
		if len(py) > 0 {
			sb.WriteString(py[0])
		}
	}
	return sb.String()
}

// toPinyinInitials 获取拼音首字母
// "小明" → "xm"
func toPinyinInitials(s string) string {
	result := pinyin.Pinyin(s, pinyinArgs)
	if len(result) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, py := range result {
		if len(py) > 0 && len(py[0]) > 0 {
			sb.WriteByte(py[0][0])
		}
	}
	return sb.String()
}

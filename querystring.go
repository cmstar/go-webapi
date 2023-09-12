package webapi

import (
	"net/url"
	"strings"
)

// QueryString 模拟 .net Framework 的 HttpRequest.QueryString 。
type QueryString struct {
	// Nameless 记录没有参数名的参数。如“?a&b=1”中的 “a”。
	Nameless string

	// HasNameless 表示是否有无名称的参数。用于区分 Nameless 是空字符串时，空字符串是否是参数值。
	HasNameless bool

	// Named 记录其余全部有参数名的参数。
	// 相同名称的参数出现多个时，会被以逗号拼接起来，如“?a=1&a=2”结果为“a=1,2”；
	// 所有参数名称都会被转为小写，以便以大小写不敏感的方式匹配参数。
	Named map[string]string
}

// Get 以大小写不敏感方式获取指定名称的参数。返回一个 bool 表示该名称的是参数是否存在。
// 只能获取有名称的参数（名称可以是空字符串），若需要无名称的，直接访问 Nameless 字段。
func (qs QueryString) Get(name string) (string, bool) {
	name = strings.ToLower(name)
	v, ok := qs.Named[name]
	if ok {
		return v, true
	}
	return "", false
}

// appendNamedParam 解析 name=value 结构，将结果追加到 QueryString.Query 。
// name 会被转为小写。若 URL 解码失败，该参数被忽略。
func (qs *QueryString) appendNamedParam(param string) {
	parts := strings.Split(param, "=")

	name, err := url.QueryUnescape(parts[0])
	if err != nil {
		return
	}
	name = strings.ToLower(name)

	value, err := url.QueryUnescape(parts[1])
	if err != nil {
		return
	}

	old, ok := qs.Named[name]
	if ok {
		qs.Named[name] = old + "," + value
	} else {
		qs.Named[name] = value
	}
}

func (qs *QueryString) appendNameless(value string) {
	v, err := url.QueryUnescape(value)

	// 有非法字符时解析不出来相关参数，相当于参数不存在。
	if err != nil {
		return
	}

	// 有多个时，用逗号拼接，e.g. "?a&b" -> "a,b"
	if qs.HasNameless {
		qs.Nameless += "," + v
	} else {
		qs.Nameless = v
		qs.HasNameless = true
	}
}

// ParseQueryString 模拟 .net Framework 的 HttpRequest.QueryString 的解析方式。
// 给定的 queryString 可以以“?”开头，也可以不带。
//
// 在传统ASP.net中，“?a&b”解析为一个名称为 null，值为“a,b”的参数；而 Go 的框架则将其等同于 “?a=&b=” 处理，变成
// 两个名称分别为 a 、 b 而值为空的参数。这与预定义的 API 协议如 SlimAPI 不符。
//
// 此方法用于获取与 .net Framework 相同的解析结果。
// 如果一个参数出现多次，会被以逗号拼接起来，如“?a=1&a=2”结果为“a=1,2”；
// 特别的，单一的“?”会得到一个没有参数名称的参数，值为空字符串。
func ParseQueryString(queryString string) QueryString {
	result := &QueryString{Named: make(map[string]string)}

	if queryString == "" {
		return *result
	}

	if queryString == "?" {
		result.HasNameless = true
		return *result
	}

	left := 0
	if queryString[0] == '?' {
		left = 1
	}

	length := len(queryString)
	right := 0

	for ; left < length; left = right + 1 {
		right = strings.IndexByte(queryString[left:], '&')
		if right == -1 {
			right = length
		} else {
			// right 是切片 [left:] 里的相对位置，绝对位置得加上 left 。
			right += left
		}

		param := queryString[left:right]

		// 含等号的是带名称的参数；否则是无名称的参数。
		if strings.Contains(param, "=") {
			result.appendNamedParam(param)
		} else {
			result.appendNameless(param)
		}
	}

	// 如果末尾是 & ，则认为后面还有个空字符串的值。
	if right == length-1 && queryString[right] == '&' {
		result.appendNameless("")
	}

	return *result
}

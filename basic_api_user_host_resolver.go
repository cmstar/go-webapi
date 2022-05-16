package webapi

import (
	"strings"
)

// basicApiUserHostResolver 提供 ApiUserHostResolver 的标准实现。
type basicApiUserHostResolver struct {
}

// NewBasicApiUserHostResolver 返回一个预定义的 ApiUserHostResolver 的标准实现。
// 当实现一个 ApiHandler 时，可基于此实例实现 ApiUserHostResolver 。
func NewBasicApiUserHostResolver() ApiUserHostResolver {
	return &basicApiUserHostResolver{}
}

func (r *basicApiUserHostResolver) FillUserHost(state *ApiState) {
	// 在标准过程里， IP 并不需要特殊处理，注意解析 X-Forwarded-For 头即可，它已经被 chi 库处理了。
	// 好像文档没有明确格式。一般是“IP:PORT”，而且 IPv6 下本地地址是“[::1]:port”。
	ip := state.RawRequest.RemoteAddr

	// ipv6的本地地址表示是“::1”或“[::1]”，这里出现这样的统一转成"127.0.0.1"，以便于统计分析。
	// 这个转换似乎有些侵入性？
	// X-Forwarded-For 头给的 IP 可能有多段，用逗号分割，下文只取第一段，所以替换一次即可。
	ip = strings.Replace(ip, "::1", "127.0.0.1", 1)

	// X-Forwarded-For 头给的第一个 IP 是客户端原始 IP 。
	parts := strings.Split(ip, ",")
	ip = parts[0]

	// 带端口的，去掉端口。上面已经替换了“::1”，除端口外应该没有冒号了。
	if colonIdx := strings.Index(ip, ":"); colonIdx > 0 {
		ip = ip[:colonIdx]
	}

	// 还可能有“[]”包裹，也去掉。
	if len(ip) > 2 && ip[0] == '[' && ip[len(ip)-1] == ']' {
		ip = ip[1 : len(ip)-1]
	}

	state.UserHost = ip
}

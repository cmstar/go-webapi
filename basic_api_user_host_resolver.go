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
	ip := state.RawRequest.RemoteAddr

	// IP 可能有多个，用逗号分割，第一个是客户端原始 IP 。
	parts := strings.Split(ip, ",")
	ip = parts[0]

	// ipv6的本地地址表示是“::1”，所有这里出现这样的统一转成"127.0.0.1"，以便于统计分析。
	// 这个转换似乎有些侵入性？
	if ip == "::1" {
		ip = "127.0.0.1"
	}

	state.UserHost = ip
}

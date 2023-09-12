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
	// 在标准过程里， IP 并不需要特殊处理，注意解析 X-Forwarded-For 头即可，它已经被 chi 库处理了，
	// 并且 X-Forwarded-For 有多段 IP 的， chi 仅保留第一段。
	// 好像文档没有明确 RemoteAddr 的格式。一般是“IP:PORT”， IPv6 下地址是“[IP]:PORT”。
	ip := state.RawRequest.RemoteAddr

	// 去掉端口部分，仅保留 IP 。
	if strings.Contains(ip, ".") { // Is IPv4?
		if colonIdx := strings.IndexByte(ip, ':'); colonIdx > 0 {
			ip = ip[:colonIdx]
		}
	} else { // IPv6
		start := strings.IndexByte(ip, '[')
		if start < 0 {
			goto END
		}

		end := strings.LastIndexByte(ip, ']')
		if end < start {
			goto END
		}

		ip = ip[start+1 : end]
	}

END:
	state.UserHost = ip
}

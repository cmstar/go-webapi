package webapi

import (
	"net/http"
	"net/netip"
	"strings"
)

// basicApiUserHostResolver 提供 ApiUserHostResolver 的标准实现。
type basicApiUserHostResolver struct {
	source BasicApiUserHostSource
}

// BasicApiUserHostSource 表示 [BasicApiUserHostResolver] 获取客户端 IP 的来源。
type BasicApiUserHostSource uint8

const (
	// FromXFF 使用 X-Forwarded-For 头的第一个 IP 。
	//
	// 此方式存在客户端伪造 X-Forwarded-For 头的风险。要安全使用，通常需要受信任的反向代理/CDN服务强制改写 XFF 头。
	FromXFF BasicApiUserHostSource = iota

	// RemoteAddr 使用请求 TCP 连接对端的 IP 。
	//
	// 会忽略 X-Forwarded-For 头，若请求通过反向代理，将无法拿到客户端原始 IP ，而是拿到代理服务器的 IP 。
	RemoteAddr
)

// BasicApiUserHostResolverOp 用于 [NewBasicApiUserHostResolver]，提供客户端 IP 来源的选项配置。
type BasicApiUserHostResolverOp struct {
	// Source 指定客户端 IP 的来源。零值为 FromXFF，以兼容旧版本的默认行为。
	Source BasicApiUserHostSource
}

// NewBasicApiUserHostResolver 返回一个预定义的 ApiUserHostResolver 的标准实现。
// 当实现一个 ApiHandler 时，可基于此实例实现 ApiUserHostResolver 。
// 未指定选项时，使用 X-Forwarded-For 的第一个 IP，以兼容旧版本的默认行为。
func NewBasicApiUserHostResolver(ops ...BasicApiUserHostResolverOp) ApiUserHostResolver {
	if len(ops) > 1 {
		panic("NewBasicApiUserHostResolver currently accepts at most one option")
	}

	op := BasicApiUserHostResolverOp{}
	if len(ops) == 1 {
		op = ops[0]
	}

	resolver := &basicApiUserHostResolver{
		source: op.Source,
	}

	if resolver.source > RemoteAddr {
		panic("unsupported BasicApiUserHostSource")
	}

	return resolver
}

func (r *basicApiUserHostResolver) FillUserHost(state *ApiState) {
	switch r.source {
	case FromXFF:
		if ip := firstIPFromXFF(state.RawRequest); ip != "" {
			state.UserHost = ip
			return
		}
		// 若取不到 IP ，继续按 RemoteAddr 处理。
	}

	// 未提供有效转发地址或指定 TCP 对端模式时，使用连接对端地址。
	ip := state.RawRequest.RemoteAddr

	// 格式一般是“IP:PORT”， IPv6 下地址是“[IP]:PORT”。
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

func firstIPFromXFF(request *http.Request) string {
	values := request.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return ""
	}

	ip, _, _ := strings.Cut(values[0], ",")
	normalized, _ := normalizeHeaderIP(ip)
	return normalized
}

// 解析请求头中的 IP 地址，并统一 IPv4 映射形式，移除 IPv6 Zone 。
func normalizeHeaderIP(value string) (string, bool) {
	ip, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return "", false
	}
	return ip.Unmap().WithZone("").String(), true
}

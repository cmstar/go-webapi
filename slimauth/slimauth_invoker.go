package slimauth

import (
	"net/http"
	"time"

	"github.com/cmstar/go-webapi/slimapi"
)

// SlimAuthInvoker 用于调用一个 SlimAuth 协议的 API 。
//
// TParam 是输入参数的类型； TData 对应输出的 [webapi.ApiResponse.Data] 。
type SlimAuthInvoker[TParam, TData any] struct {
	*slimapi.SlimApiInvoker[TParam, TData]
}

// SlimAuthInvokerOp 用于初始化 [SlimAuthInvoker] 。
type SlimAuthInvokerOp struct {
	Uri        string // 目标 URL 。
	Key        string // SlimAuth 协议的 key 。
	Secret     string // SlimAuth 协议的 secret 。
	AuthScheme string // Authorization 头的 <scheme> 部分，为空时自动使用默认值。
}

// SlimAuthInvoker 创建一个 [SlimAuthInvoker] 实例。
func NewSlimAuthInvoker[TParam, TData any](op SlimAuthInvokerOp) *SlimAuthInvoker[TParam, TData] {
	inner := slimapi.NewSlimApiInvoker[TParam, TData](op.Uri)
	inner.RequestSetup = func(r *http.Request) error {
		timestamp := time.Now().Unix()
		signResult := AppendSign(r, op.Key, op.Secret, op.AuthScheme, timestamp)
		return signResult.Cause
	}

	return &SlimAuthInvoker[TParam, TData]{
		SlimApiInvoker: inner,
	}
}

package webapi

import (
	"errors"

	"github.com/cmstar/go-errx"
)

// basicApiResponseBuilder 提供 ApiResponseBuilder 的标准实现。
type basicApiResponseBuilder struct {
}

// NewBasicApiResponseBuilder 返回一个预定义的 ApiResponseBuilder 的标准实现。
// 当实现一个 ApiHandler 时，可基于此实例实现 ApiResponseBuilder 。
func NewBasicApiResponseBuilder() ApiResponseBuilder {
	return &basicApiResponseBuilder{}
}

// BuildResponse implements ApiResponseBuilder.BuildResponse
func (r *basicApiResponseBuilder) BuildResponse(state *ApiState) {
	resp := &ApiResponse{
		Data: state.Data,
	}
	state.Response = resp

	if state.Error == nil {
		return
	}

	// ApiResponse 内容是返回给请求者的，不应该暴露内部细节，只有 BizError 是和具体业务高度关联的，
	// 可以给出具体信息；对于其他错误，只给一个笼统的信息。
	var bizErr errx.BizError
	if errors.As(state.Error, &bizErr) {
		resp.Code = bizErr.Code()
		resp.Message = bizErr.Message()
		return
	}

	var badRequestErr BadRequestError
	if errors.As(state.Error, &badRequestErr) {
		resp.Code = ErrorCodeBadRequest
		resp.Message = "bad request"
		return
	}

	resp.Code = ErrorCodeInternalError
	resp.Message = "internal error"
}

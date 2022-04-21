package webapi

import (
	"errors"
	"reflect"

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
	typBizError := reflect.TypeOf((*errx.BizError)(nil)).Elem()
	if err := r.toErrorType(state.Error, typBizError); err != nil {
		e := err.(errx.BizError)
		resp.Code = e.Code()
		resp.Message = e.Message()
		return
	}

	typBadRequestError := reflect.TypeOf(BadRequestError{})
	if err := r.toErrorType(state.Error, typBadRequestError); err != nil {
		resp.Code = ErrorCodeBadRequest
		resp.Message = "bad request"
		return
	}

	resp.Code = ErrorCodeInternalError
	resp.Message = "internal error"
}

func (r *basicApiResponseBuilder) toErrorType(src error, typDest reflect.Type) error {
	// 错误可能被封装，需要一层层解出来。
	// 同时 errx.BizError/StackfulError 是接口，需要以接口方式判断。
	isInterface := typDest.Kind() == reflect.Interface

	for {
		typSrc := reflect.TypeOf(src)

		if isInterface {
			if typSrc.Implements(typDest) {
				return src
			}
		} else {
			if typSrc == typDest {
				return src
			}
		}

		src = errors.Unwrap(src)
		if src == nil {
			return nil
		}
	}
}

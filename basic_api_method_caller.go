package webapi

// basicApiMethodCaller 实现 ApiMethodCaller ，提供一个标准过程。
// 一个 WebAPI 方法可以有 0 到 2 个返回值：
// 若只有一个返回值，可以是正常的结果，也可以返回 error 表示错误。
// 若有两个返回值，则第一个表示正常的结果，第二个必须是 error 。
type basicApiMethodCaller struct {
}

// NewBasicApiMethodCaller 返回一个预定义的 ApiMethodCaller 的标准实现。
// 当实现一个 ApiHandler 时，可基于此实例实现 ApiMethodCaller 。
func NewBasicApiMethodCaller() ApiMethodCaller {
	return &basicApiMethodCaller{}
}

// Call implements ApiMethodCaller.Call
func (c *basicApiMethodCaller) Call(state *ApiState) {
	state.MustHaveMethod()
	res := state.Method.Value.Call(state.Args)

	switch len(res) {
	case 0:
		// Nothing to do.
	case 1:
		data := res[0].Interface()
		if e, ok := data.(error); ok {
			state.Error = e
		} else {
			state.Data = data
		}

	case 2:
		state.Data = res[0].Interface()

		err := res[1].Interface()
		if err != nil {
			var e error
			var ok bool
			if e, ok = err.(error); !ok {
				// 方法注册阶段应该做过校验，这行应该不会执行。
				PanicApiError(state, nil, "the second output parameter must be an error, got %T", err)
			}
			state.Error = e
		}

	default:
		PanicApiError(state, nil, "the return value of method '%s' cannot be greater than 2", state.Name)
	}
}

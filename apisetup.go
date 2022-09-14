package webapi

// ApiSetup 用于向 ApiHandler 注册 API 方法。
type ApiSetup struct {
	engine  *ApiEngine
	handler ApiHandler
}

// RegisterMethods 同 ApiMethodRegister.RegisterMethods 。
// 将给定的 struct 上的所有公开方法注册为 WebAPI 。若给定的不是 struct ，则 panic 。
// 返回 ApiSetup 实例自身，以便编码形成流式调用。
//
// 对方法名称使用一组约定（下划线使用名称中的第一个下划线）：
//   - 若方法名称格式为 Method__Name （使用两个下划线分割），则 Name 被注册为 WebAPI 名称；
//   - 若方法名称格式为 Method__ （使用两个下划线结尾）或 Method____ （两个下划线之后也只有下划线），则此方法不会被注册为 WebAPI ；
//   - 其余情况，均使用方法的原始名称作为 WebAPI 名称。
// 这里 Method 和 Name 均为可变量， Method 用于指代代码内有意义的方法名称， Name 指代 WebAPI 名称。例如 GetName__13 注册一个名称为
// “13”的 API 方法，其方法业务意义为 GetName 。
//
// 每个方法的注册逻辑与 RegisterMethod 一致。
// 特别的，如果格式为 Method____abc ，两个下划线之后存在有效名称，则 WebAPI 名称为 __abc ，从两个下划线后的下一个字符（还是下划线）开始取。
//
func (setup ApiSetup) RegisterMethods(providerStruct any) ApiSetup {
	setup.handler.RegisterMethods(providerStruct)
	return setup
}

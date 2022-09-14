package webapi

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/cmstar/go-conv"
)

// basicApiMethodRegister 提供 ApiMethodRegister 的标准实现。
type basicApiMethodRegister struct {
	methods *sync.Map
}

// NewBasicApiMethodRegister 返回一个预定义的 ApiMethodRegister 的标准实现。
// 当实现一个 ApiHandler 时，可基于此实例实现 ApiMethodRegister 。
func NewBasicApiMethodRegister() ApiMethodRegister {
	return &basicApiMethodRegister{
		methods: new(sync.Map),
	}
}

// RegisterOne implements ApiMethodRegister.RegisterOne
func (r *basicApiMethodRegister) RegisterMethod(m ApiMethod) {
	r.checkMethodOut(m.Value, m.Name)

	// 用于检索的名称忽略大小写。
	name := strings.ToLower(m.Name)
	r.methods.Store(name, m)
}

// Register implements ApiMethodRegister.Register
func (r *basicApiMethodRegister) RegisterMethods(providerStruct any) {
	if providerStruct == nil {
		panic("the given provider should not be nil")
	}

	t := reflect.TypeOf(providerStruct)
	if t.Kind() != reflect.Struct {
		panic("the given provider must be a struct")
	}

	v := reflect.ValueOf(providerStruct)
	num := t.NumMethod()

	for i := 0; i < num; i++ {
		typMethod := t.Method(i)
		name, ignore := r.fixNameOrIgnore(typMethod.Name)
		if ignore {
			continue
		}

		valMethod := v.Method(i)
		r.RegisterMethod(ApiMethod{name, valMethod, t.Name()})
	}
}

// GetMethod implements ApiMethodRegister.GetMethod
func (r *basicApiMethodRegister) GetMethod(name string) (method ApiMethod, ok bool) {
	if r.methods == nil {
		return ApiMethod{}, false
	}

	// 名称需忽略大小写。
	name = strings.ToLower(name)

	m, ok := r.methods.Load(name)
	if ok {
		method = m.(ApiMethod)
	}
	return
}

// checkMethodOut 校验方法的输出参数。在参数不合规时 panic 。
// 允许方法允许有0-2个输出参数。
// 1个参数时，参数可以是任意 struct/map[string]*/基础类型 或者此三类作为元素的 slice ，也可以是 error 。
// 2个参数时，第一个参数可以是  struct/map[string]*/基础类型 或者此三类作为元素的 slice ，第二个参数必须是 error 。
func (r *basicApiMethodRegister) checkMethodOut(method reflect.Value, webApiName string) {
	typ := method.Type()
	num := typ.NumOut()
	switch num {
	case 0:
		// Nothing to do.

	case 1:
		tOut := typ.Out(0)
		r.mustBeSupportedOut(method, webApiName, tOut, true)

	case 2:
		first := typ.Out(0)
		r.mustBeSupportedOut(method, webApiName, first, false)

		second := typ.Out(1)
		if !second.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			panic(fmt.Sprintf("the second output parameter of the API method '%v' must be an error", webApiName))
		}

	default:
		panic(fmt.Sprintf("the method the API method '%v' has more than 2 output parameters", webApiName))
	}
}

func (r *basicApiMethodRegister) mustBeSupportedOut(method reflect.Value, webApiName string, outTyp reflect.Type, canBeError bool) {
	kind := outTyp.Kind()
	ok := false

	switch {
	case kind == reflect.Interface:
		errTyp := reflect.TypeOf((*error)(nil)).Elem()
		if canBeError && outTyp.Implements(errTyp) {
			ok = true
		}

	case conv.IsSimpleType(outTyp) || kind == reflect.Struct:
		ok = true

	case kind == reflect.Slice || kind == reflect.Ptr:
		elemTyp := outTyp.Elem()
		r.mustBeSupportedOut(method, webApiName, elemTyp, false)
		ok = true

	case kind == reflect.Map:
		keyTyp := outTyp.Key()
		r.mustBeSupportedOut(method, webApiName, keyTyp, false)
		elemTyp := outTyp.Elem()
		r.mustBeSupportedOut(method, webApiName, elemTyp, false)
		ok = true
	}

	if !ok {
		panic(fmt.Sprintf("the type of the output parameter '%v' of method '%v' is not supported", outTyp, webApiName))
	}
}

// fixNameOrIgnore 判断方法名称的格式，提取出作为 WebAPI 的名称。
// 若方法不应被注册为 WebAPI ，则 ignore=false 。 fixedName 为提取出的名称，为方法名称的字串，可含大小写。
func (r *basicApiMethodRegister) fixNameOrIgnore(methodName string) (fixedName string, ignore bool) {
	// 对方法名称使用一组约定（下划线使用名称中的第一个下划线）：
	// - 若方法名称格式为 Method__Name （使用两个下划线分割），则 Name 被注册为 WebAPI 名称；
	// - 若方法名称格式为 Method__ （使用两个下划线结尾）或 Method____ （两个下划线之后也只有下划线），则此方法不会被注册为 WebAPI ；
	// - 其余情况，均使用方法的原始名称作为 WebAPI 名称。
	// 特别的，如果格式为 Method____abc ，两个下划线之后存在有效名称，则 WebAPI 名称为 __abc ，从两个下划线后的下一个字符（还是下划线）开始取。

	const delimiter = "__"

	underline := strings.Index(methodName, delimiter)
	if underline == -1 {
		fixedName = methodName
		ignore = false
		return
	}

	// 判断分隔符之后是不是只有下划线。
	ok := false
	i := underline + len(delimiter)
	for ; i < len(methodName); i++ {
		if methodName[i] == '_' {
			continue
		}

		ok = true
		break
	}

	// 只有下划线的忽略。
	if !ok {
		fixedName = ""
		ignore = true
		return
	}

	// 存在有效名称的，取分隔符后的下一个字符（不管是不是下划线）作为 WebAPI 名称。
	fixedName = methodName[underline+len(delimiter):]
	ignore = fixedName == ""
	return
}

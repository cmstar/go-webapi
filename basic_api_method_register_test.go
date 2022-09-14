package webapi

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func Test_basicApiMethodRegister_RegisterMethod(t *testing.T) {
	reg := NewBasicApiMethodRegister().(*basicApiMethodRegister)

	// @name 测试用例名称，也是注册的方法名称。
	// @panicPattern 若不为空，断言 panic 的消息，必须匹配此正则。
	// @f 要注册的函数。
	testOne := func(name, panicPattern string, f any) {
		t.Run(name, func(t *testing.T) {
			defer func() {
				var msg any
				if msg = recover(); msg == nil {
					return
				}

				if panicPattern == "" {
					t.Errorf("expect no error, got %v", msg)
					return
				}

				msgStr, ok := msg.(string)
				if !ok {
					t.Errorf("expect a string message, got: %T", msgStr)
					return
				}

				if match, _ := regexp.MatchString(panicPattern, msgStr); !match {
					t.Errorf("%q should match %q", msgStr, panicPattern)
					return
				}
			}()

			methodValue := reflect.ValueOf(f)
			reg.RegisterMethod(ApiMethod{name, methodValue, ""})

			found := false
			lowerName := strings.ToLower(name)

			reg.methods.Range(func(k, m any) bool {
				if k == lowerName && m.(ApiMethod).Value == methodValue {
					found = true
					return false
				}
				return true
			})

			if !found {
				t.Error("not found")
			}
		})
	}

	// 测试函数的各种输入输出值情况。
	testOne("empty", "", func() { panic("never run") })
	testOne("ErrOnly", "", func(a1 int) error { panic("never run") })
	testOne("ErrOnly2", "", func(a1, a2 string) ApiError { panic("never run") })
	testOne("int", "", func() int { panic("never run") })
	testOne("string", "", func() string { panic("never run") })
	testOne("slice", "", func() []int { panic("never run") })
	testOne("SliceSlice", "", func() [][]int { panic("never run") })
	testOne("map", "", func() map[string]int { panic("never run") })
	testOne("WithError", "", func() (map[string]int, error) { panic("never run") })
	testOne("ptr1", "", func() *string { panic("never run") })
	testOne("ptr2", "", func() ([]*string, error) { panic("never run") })
	testOne("ptr3", "", func() map[*string]**int { panic("never run") })

	// 非法情况。
	testOne("NotError", "'NotError' must be an error", func() (int, string) { panic("never run") })
	testOne("TooManyParam", "'TooManyParam' has more than 2 output parameters", func() (int, string, int) { panic("never run") })
	testOne("NotSupportedType", "'chan int' of method 'NotSupportedType' is not supported", func() chan int { panic("never run") })
}

func Test_basicApiMethodRegister_fixNameOrIgnore(t *testing.T) {
	tests := []struct {
		name          string
		args          string
		wantFixedName string
		wantIgnore    bool
	}{
		{"raw1", "methodname", "methodname", false},
		{"raw2", "name_act1", "name_act1", false},
		{"alias1", "name__act1", "act1", false},
		{"alias2", "name__33", "33", false},
		{"alias3", "__AfteR", "AfteR", false},
		{"alias3", "___F_3_", "_F_3_", false},
		{"ignore1", "name___", "", true},
		{"ignore2", "___", "", true},
	}

	reg := NewBasicApiMethodRegister().(*basicApiMethodRegister)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFixedName, gotIgnore := reg.fixNameOrIgnore(tt.args)

			if gotFixedName != tt.wantFixedName {
				t.Errorf("BasicApiMethodRegister.fixNameOrIgnore() gotFixedName = %v, want %v", gotFixedName, tt.wantFixedName)
			}

			if gotIgnore != tt.wantIgnore {
				t.Errorf("BasicApiMethodRegister.fixNameOrIgnore() gotIgnore = %v, want %v", gotIgnore, tt.wantIgnore)
			}
		})
	}
}

func Test_basicApiMethodRegister_GetMethod(t *testing.T) {
	reg := NewBasicApiMethodRegister().(*basicApiMethodRegister)

	testOne := func(name string, expectedOk bool, expectedMethod reflect.Value) {
		t.Run(name, func(t *testing.T) {
			method, ok := reg.GetMethod(name)

			if !ok && method != (ApiMethod{}) {
				t.Error("value must be the default value")
				return
			}

			if ok != expectedOk {
				t.Errorf("expect ok %v, got %v", ok, expectedOk)
				return
			}

			if method.Value != expectedMethod {
				t.Error("method mismatch")
				return
			}
		})
	}

	testOne("", false, reflect.Value{})

	// 测试用例和 RegisterMethod 方法相互验证。
	m := reflect.ValueOf(func() {})
	reg.RegisterMethod(ApiMethod{"", m, ""})
	testOne("", true, m)

	reg.RegisterMethod(ApiMethod{"AbCd", m, ""})
	testOne("AbCd", true, m)
	testOne("abcd", true, m)
	testOne("aBCd", true, m)

	// 重复注册，覆盖原有。
	m2 := reflect.ValueOf(func() {})
	reg.RegisterMethod(ApiMethod{"abcd", m2, ""})
	testOne("ABCD", true, m2)
	testOne("AbcD", true, m2)
}

func Test_basicApiMethodRegister_RegisterMethods(t *testing.T) {
	reg := NewBasicApiMethodRegister().(*basicApiMethodRegister)
	provider := basicApiMethodRegisterTestProvider{}
	valProvider := reflect.ValueOf(provider)
	reg.RegisterMethods(provider)

	// 这些调用只是为了屏蔽 Go lint 的警告，暂时不知道怎么通过设置或注释来关闭它。
	provider.noRegister()
	provider.__no()
	provider.__()

	count := 0
	check := func(funName, apiName string) {
		t.Run(funName, func(t *testing.T) {
			count++

			m := valProvider.MethodByName(funName)
			if (m == reflect.Value{}) {
				t.Errorf("method %v not found", funName)
				return
			}

			lowerName := strings.ToLower(apiName)
			out, ok := reg.methods.Load(lowerName)
			if !ok {
				t.Errorf("API name %v not found", apiName)
				return
			}

			m2 := out.(ApiMethod)
			if m2.Name != apiName {
				t.Errorf("ApiMethod.Name mismatch, expect %v, got %v", apiName, m2.Name)
				return
			}

			if m2.Provider != "basicApiMethodRegisterTestProvider" {
				t.Errorf("ApiMethod.Provider mismatch %v", m2.Provider)
				return
			}

			if m2.Value != m {
				t.Errorf("ApiMethod.Value mismatch %v", apiName)
				return
			}
		})
	}

	// 依次检测每个方法。
	check("A1", "A1")
	check("A1b2", "A1b2")
	check("A1__3", "3")
	check("Do___a", "_a")
	check("Do____a_B", "__a_B")

	// 确认没有其他方法被注册。
	c := 0
	reg.methods.Range(func(key, value any) bool { c++; return true })
	if count != c {
		t.Errorf("totally %d methods, expect %d", c, count)
	}
}

type basicApiMethodRegisterTestProvider struct{}

func (basicApiMethodRegisterTestProvider) noRegister() {}
func (basicApiMethodRegisterTestProvider) __no()       {}
func (basicApiMethodRegisterTestProvider) __()         {}
func (basicApiMethodRegisterTestProvider) _()          {}
func (basicApiMethodRegisterTestProvider) A1()         {}
func (basicApiMethodRegisterTestProvider) A1b2()       {}
func (basicApiMethodRegisterTestProvider) A1__3()      {}
func (basicApiMethodRegisterTestProvider) Do___()      {} // Ignored.
func (basicApiMethodRegisterTestProvider) Do____()     {} // Ignored.
func (basicApiMethodRegisterTestProvider) Do___a()     {}
func (basicApiMethodRegisterTestProvider) Do____a_B()  {}

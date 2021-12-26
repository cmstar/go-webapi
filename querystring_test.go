package webapi

import (
	"reflect"
	"testing"
)

func TestParseQueryString(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		want        QueryString
	}{
		{
			"none",
			"",
			QueryString{
				Nameless:    "",
				Named:       map[string]string{},
				HasNameless: false,
			},
		},

		{
			"p1",
			"?a=1",
			QueryString{
				Nameless:    "",
				Named:       map[string]string{"a": "1"},
				HasNameless: false,
			},
		},

		{
			"p2",
			"a=1&a=%E4%B8%AD%E6%96%87",
			QueryString{
				Nameless:    "",
				Named:       map[string]string{"a": "1,中文"},
				HasNameless: false,
			},
		},

		{
			"nameless0",
			"?",
			QueryString{
				Nameless:    "",
				Named:       map[string]string{},
				HasNameless: true,
			},
		},

		{
			"nameless1",
			"?a=1&b&a=%E4%B8%AD%E6%96%87",
			QueryString{
				Nameless:    "b",
				Named:       map[string]string{"a": "1,中文"},
				HasNameless: true,
			},
		},

		{
			"nameless2",
			"1&a=1&b&AbC=2&%E4%B8%AD%E6%96%87",
			QueryString{
				Nameless:    "1,b,中文",
				Named:       map[string]string{"a": "1", "abc": "2"},
				HasNameless: true,
			},
		},

		{
			"empty1",
			"?&",
			QueryString{
				Nameless:    ",",
				Named:       map[string]string{},
				HasNameless: true,
			},
		},

		{
			"empty2",
			"A&&",
			QueryString{
				Nameless:    "A,,",
				Named:       map[string]string{},
				HasNameless: true,
			},
		},

		{
			"empty3",
			"?%E4%B8%AD%E6%96%87&B=2&",
			QueryString{
				Nameless:    "中文,",
				Named:       map[string]string{"b": "2"},
				HasNameless: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseQueryString(tt.queryString); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseQueryString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryString_Get(t *testing.T) {
	qs := QueryString{
		Named: map[string]string{
			"a": "1",
			"b": "1",
		},
	}

	t.Run("a", func(t *testing.T) {
		v, ok := qs.Get("a")
		if !ok || v != "1" {
			t.Errorf("QueryString.Get(a) = %v, want 1", v)
		}

		v, ok = qs.Get("A")
		if !ok || v != "1" {
			t.Errorf("QueryString.Get(A) = %v, want 1", v)
		}
	})

	t.Run("miss", func(t *testing.T) {
		v, ok := qs.Get("x")
		if ok || v != "" {
			t.Errorf("QueryString.Get(x) = %v, want 'not-exist'", v)
		}
	})
}

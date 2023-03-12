package webapi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cmstar/go-errx"
	"github.com/stretchr/testify/assert"
)

func Test_basicApiResponseBuilder_BuildResponse(t *testing.T) {
	b := NewBasicApiResponseBuilder()

	t.Run("no-error", func(t *testing.T) {
		state := &ApiState{
			Data:  123,
			Error: nil,
		}
		b.BuildResponse(state)
		expect := ApiResponse[any]{
			Code:    0,
			Message: "",
			Data:    123,
		}
		assert.Equal(t, expect, *state.Response)
	})

	t.Run("biz", func(t *testing.T) {
		state := &ApiState{
			Data:  "d",
			Error: errx.NewBizError(12, "m", nil),
		}
		b.BuildResponse(state)
		expect := ApiResponse[any]{
			Code:    12,
			Message: "m",
			Data:    "d",
		}
		assert.Equal(t, expect, *state.Response)
	})

	t.Run("biz-wrap", func(t *testing.T) {
		state := &ApiState{
			Data:  "d",
			Error: errx.Wrap("p1", fmt.Errorf("p2: %w", errx.NewBizError(123, "mm", nil))),
		}
		b.BuildResponse(state)
		expect := ApiResponse[any]{
			Code:    123,
			Message: "mm",
			Data:    "d",
		}
		assert.Equal(t, expect, *state.Response)
	})

	t.Run("bad-request", func(t *testing.T) {
		state := &ApiState{
			Data:  "d",
			Error: CreateBadRequestError(nil, nil, "x"),
		}
		b.BuildResponse(state)
		expect := ApiResponse[any]{
			Code:    ErrorCodeBadRequest,
			Message: "x",
			Data:    "d",
		}
		assert.Equal(t, expect, *state.Response)
	})

	t.Run("bad-request-wrap-from-panic", func(t *testing.T) {
		var err error
		func() {
			defer func() {
				err = errx.PreserveRecover("gg", recover())
			}()
			panic(CreateBadRequestError(nil, nil, "x"))
		}()

		state := &ApiState{
			Data:  "d",
			Error: err,
		}
		b.BuildResponse(state)
		expect := ApiResponse[any]{
			Code:    ErrorCodeBadRequest,
			Message: "x",
			Data:    "d",
		}
		assert.Equal(t, expect, *state.Response)
	})

	t.Run("other", func(t *testing.T) {
		state := &ApiState{
			Data:  nil,
			Error: errors.New("gg"),
		}
		b.BuildResponse(state)
		expect := ApiResponse[any]{
			Code:    ErrorCodeInternalError,
			Message: "internal error",
			Data:    nil,
		}
		assert.Equal(t, expect, *state.Response)
	})

	t.Run("other-wrap", func(t *testing.T) {

	})
}

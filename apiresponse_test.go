package webapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessResponse(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		got := SuccessResponse(1)
		assert.Equal(t, 0, got.Code)
		assert.Equal(t, "", got.Message)
		assert.Equal(t, 1, got.Data)
	})

	t.Run("struct", func(t *testing.T) {
		data := struct{ X, Y int }{1, 2}
		got := SuccessResponse(data)
		assert.Equal(t, 0, got.Code)
		assert.Equal(t, "", got.Message)
		assert.Equal(t, data, got.Data)
	})
}

func TestBadRequestResponse(t *testing.T) {
	got := BadRequestResponse()
	assert.Equal(t, 400, got.Code)
	assert.Equal(t, "bad request", got.Message)
	assert.Equal(t, nil, got.Data)
}

func TestInternalErrorResponse(t *testing.T) {
	got := InternalErrorResponse()
	assert.Equal(t, 500, got.Code)
	assert.Equal(t, "internal error", got.Message)
	assert.Equal(t, nil, got.Data)
}

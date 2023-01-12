package webapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiUserHostResolver(t *testing.T) {
	testOne := func(ip, want string) {
		t.Run(ip, func(t *testing.T) {
			state := &ApiState{
				RawRequest: &http.Request{
					RemoteAddr: ip,
				},
			}
			NewBasicApiUserHostResolver().FillUserHost(state)
			assert.Equal(t, want, state.UserHost)
		})
	}

	testOne("", "")
	testOne("1.2.3.4", "1.2.3.4")
	testOne("1.2.3.4:666", "1.2.3.4")
	testOne("::1", "::1")
	testOne("[::1]", "::1")
	testOne("[::1]:1234", "::1")
	testOne("[1:2::3:4]:1234", "1:2::3:4")

	// Bad IPs.
	testOne(":", ":")
	testOne("::", "::")
	testOne("[", "[")
	testOne(":[", ":[")
	testOne("]", "]")
	testOne("100", "100")
}

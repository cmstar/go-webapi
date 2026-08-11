package webapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiUserHostResolver_ipFormat(t *testing.T) {
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

func TestBasicApiUserHostResolver_op(t *testing.T) {
	testOne := func(name, xff, want string, op BasicApiUserHostResolverOp) {
		resolver := NewBasicApiUserHostResolver(op)

		t.Run(name, func(t *testing.T) {
			state := &ApiState{
				RawRequest: &http.Request{
					RemoteAddr: "192.0.2.10:1234",
					Header: http.Header{
						"X-Forwarded-For": {xff},
					},
				},
			}

			resolver.FillUserHost(state)

			assert.Equal(t, want, state.UserHost)
		})
	}

	testOne("xff-first-ip", "198.51.100.1, 203.0.113.20", "198.51.100.1", BasicApiUserHostResolverOp{Source: FromXFF})
	testOne("xff-invalid", "invalid, 198.51.100.1", "192.0.2.10", BasicApiUserHostResolverOp{Source: FromXFF})
	testOne("remote-addr", "198.51.100.1, 203.0.113.20", "192.0.2.10", BasicApiUserHostResolverOp{Source: RemoteAddr})

	assert.Panics(t, func() {
		testOne("invalid-op", "", "192.0.2.10", BasicApiUserHostResolverOp{Source: BasicApiUserHostSource(99)})
	})
}

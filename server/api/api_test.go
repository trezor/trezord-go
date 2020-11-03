package api

import (
	"testing"
)

// Test the origin validation
func TestOriginValidator(t *testing.T) {
	testcases := []struct {
		origin string
		allow  bool
	}{
		// `null` should be denied
		{"null", false},
		// HTTPS for trezor.io should be allowed
		{"https://trezor.io", true},
		{"https://foo.trezor.io", true},
		{"https://bar.foo.trezor.io", true},
		// but HTTP for trezor.io should be denied
		{"http://trezor.io", false},
		{"http://foo.trezor.io", false},
		{"http://bar.foo.trezor.io", false},
		// Fakes should be denied
		{"https://faketrezor.io", false},
		{"https://foo.faketrezor.io", false},
		{"https://foo.trezor.ioo", false},
		{"http://faketrezor.io", false},
		{"http://foo.faketrezor.io", false},
		{"http://foo.trezor.ioo", false},
		// Trezor onion should be allowed
		{"http://trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", true},
		{"https://trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", true},
		{"http://foo.trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", true},
		{"https://foo.trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", true},
		{"http://bar.foo.trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", true},
		{"https://bar.foo.trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", true},
		// Fake trezor onions should be denied
		{"http://trezoriovpjcahpzkrewelclulmszwbqpzmzgub48gbcjlvluxtruqad.onion", false},
		{"https://trezoriovpjcahpzkrewelclulmszwbqpzmzgub48gbcjlvluxtruqad.onion", false},
		{"http://trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqbd.onion", false},
		{"https://trezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqbd.onion", false},
		{"http://trezoriowpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"https://trezoriowpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"http://foo.trezoriovpjcahpzkrewelclulmszwbqpzmzgub48gbcjlvluxtruqad.onion", false},
		{"https://bar.foo.trezoriovpjcahpzkrewelclulmszwbqpzmzgub48gbcjlvluxtruqad.onion", false},
		{"http://faketrezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"https://faketrezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"http://foo.faketrezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"https://foo.faketrezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"http://bar.foo.faketrezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		{"https://bar.foo.faketrezoriovpjcahpzkrewelclulmszwbqpzmzgub37gbcjlvluxtruqad.onion", false},
		// Localhost 8xxx and 5xxx should be allowed for local development
		{"https://localhost:8000", true},
		{"http://localhost:8000", true},
		{"http://localhost:8999", true},
		{"https://localhost:5000", true},
		{"http://localhost:5000", true},
		{"http://localhost:5999", true},
		// SatoshiLabs dev servers should be allowed
		{"https://sldev.cz", true},
		{"https://foo.sldev.cz", true},
		{"https://bar.foo.sldev.cz", true},
		// Fake SatoshiLabs dev servers should be denied
		{"https://fakesldev.cz", false},
		{"https://foo.fakesldev.cz", false},
		{"https://foo.sldev.czz", false},
		{"http://foo.trezor.sldev.cz", false},
		// Other ports should be denied
		{"http://localhost", false},
		{"http://localhost:1234", false},
	}
	validator := corsValidator()
	for _, tc := range testcases {
		allow := validator(tc.origin)
		if allow != tc.allow {
			t.Errorf("Origin %q: expected %v, got %v", tc.origin, tc.allow, allow)
		}
	}
}

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifiedClientIP(t *testing.T) {
	proxies, err := ParseTrustedProxyCIDRs("10.0.0.0/24, 2001:db8:1::/64")
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{trustedProxies: proxies}
	for _, tc := range []struct {
		name, peer string
		headers    []string
		want       string
	}{
		{"untrusted", "192.0.2.1:1234", []string{"198.51.100.1"}, "192.0.2.1"},
		{"missing", "10.0.0.1:1234", nil, "10.0.0.1"},
		{"client", "10.0.0.1:1234", []string{"198.51.100.1"}, "198.51.100.1"},
		{"spoofed left", "10.0.0.1:1234", []string{"203.0.113.99, 198.51.100.1, 10.0.0.2"}, "198.51.100.1"},
		{"multiple lines", "10.0.0.1:1234", []string{"203.0.113.99, 198.51.100.1", "10.0.0.2"}, "198.51.100.1"},
		{"invalid left", "10.0.0.1:1234", []string{"invalid, 198.51.100.1"}, "10.0.0.1"},
		{"empty token", "10.0.0.1:1234", []string{"198.51.100.1,"}, "10.0.0.1"},
		{"all trusted", "10.0.0.1:1234", []string{"10.0.0.2"}, "10.0.0.1"},
		{"port rejected", "10.0.0.1:1234", []string{"198.51.100.1:80"}, "10.0.0.1"},
		{"zone rejected", "10.0.0.1:1234", []string{"fe80::1%eth0"}, "10.0.0.1"},
		{"mapped client", "10.0.0.1:1234", []string{"::ffff:198.51.100.1"}, "198.51.100.1"},
		{"mapped peer", "[::ffff:10.0.0.1]:1234", []string{"198.51.100.1"}, "198.51.100.1"},
		{"ipv6", "[2001:db8:1::1]:1234", []string{"2001:0db8:2::0001"}, "2001:db8:2::1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = tc.peer
			for _, value := range tc.headers {
				r.Header.Add("X-Forwarded-For", value)
			}
			if got := h.clientIP(r); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
	for _, setting := range []string{"not-a-cidr", "10.0.0.1", "10.0.0.0/24,", "::ffff:10.0.0.0/120"} {
		if _, err := ParseTrustedProxyCIDRs(setting); err == nil {
			t.Fatalf("accepted invalid config %q", setting)
		}
	}
	if _, err := ParseTrustedProxyCIDRs(""); err != nil {
		t.Fatal(err)
	}
}

func TestProxyLoginRegistrationQuota(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		proxies, _ := ParseTrustedProxyCIDRs("10.0.0.0/24")
		h := &Handler{limits: make(map[string]*loginLimit)}
		if trusted {
			h.trustedProxies = proxies
		}
		call := func(client string, register bool) *httptest.ResponseRecorder {
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = "10.0.0.1:5000"
			r.Header.Set("X-Forwarded-For", client)
			w := httptest.NewRecorder()
			// Invalid media type stops before DB access, after enforcing the real limiter.
			if register {
				h.RegisterPatient(w, r)
			} else {
				h.Login(w, r)
			}
			return w
		}
		for i := 0; i < 5; i++ {
			if got := call("198.51.100.1", false).Code; got != http.StatusUnsupportedMediaType {
				t.Fatal(got)
			}
		}
		w := call("198.51.100.1", true)
		if w.Code != 429 || w.Header().Get("Retry-After") != "12" {
			t.Fatal("shared registration quota lost")
		}
		want := 429
		if trusted {
			want = 415
		}
		if got := call("198.51.100.2", false).Code; got != want {
			t.Fatalf("trusted=%v got=%d want=%d", trusted, got, want)
		}
	}
}

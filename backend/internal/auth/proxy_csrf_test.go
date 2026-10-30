package auth

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/csrf"
)

func TestCSRFThroughSeparateFrontendOrigin(t *testing.T) {
	handler := Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, csrf.Token(r))
	}), bytes.Repeat([]byte{7}, 32), true, "app.example.test")
	initial := httptest.NewRecorder()
	handler.ServeHTTP(initial, httptest.NewRequest("GET", "https://api.example.test/token", nil))
	for _, tc := range []struct {
		name, origin string
		token        bool
		status       int
	}{
		{"frontend with valid token", "https://app.example.test", true, 200},
		{"untrusted origin", "https://attacker.example.test", true, 403},
		{"similar hostname", "https://app.example.test.attacker.test", true, 403},
		{"missing token", "https://app.example.test", false, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "https://api.example.test/action", nil)
			r.Header.Set("Origin", tc.origin)
			if tc.token {
				r.Header.Set("X-CSRF-Token", initial.Body.String())
			}
			for _, cookie := range initial.Result().Cookies() {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("got status %d; want %d", w.Code, tc.status)
			}
		})
	}
}

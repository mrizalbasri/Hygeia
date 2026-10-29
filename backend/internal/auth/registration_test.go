package auth

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegistrationRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{
		`{"email":"invalid","full_name":"Synthetic","birth_year":2000,"sex":"unknown"}`,
		`{"email":"Name <test@example.test>","full_name":"Synthetic","birth_year":2000,"sex":"unknown"}`,
		fmt.Sprintf(`{"email":"test@example.test","full_name":"Synthetic","birth_year":%d,"sex":"unknown"}`, time.Now().Year()+1),
		`{"email":"test@example.test","full_name":"Synthetic","birth_year":2000,"sex":"invalid"}`,
		`{"email":"test@example.test","full_name":"Synthetic","birth_year":2000,"sex":"unknown","password":"short"}`,
		`{} {}`,
		`{"role":"admin"}`,
	} {
		h := &Handler{limits: make(map[string]*loginLimit)}
		r := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(input))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.RegisterPatient(w, r)
		if w.Code != 400 {
			t.Fatalf("invalid registration status=%d body=%s", w.Code, w.Body.String())
		}
	}
}

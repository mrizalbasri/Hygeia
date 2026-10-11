package laboratory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInputValidation(t *testing.T) {
	h := &Handler{}
	mux := http.NewServeMux()
	h.Register(mux)
	id := "00000000-0000-0000-0000-000000000001"
	for _, tc := range []struct {
		method, path, body, media string
		status                    int
	}{
		{"POST", "/api/v1/lab-requests", `{}`, "text/plain", 415},
		{"POST", "/api/v1/lab-requests", `{}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests", `{"episode_id":"` + id + `","lab_org_id":"` + id + `","test_code":"A","test_name":" ","doctor_id":"spoof"}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests", `{"test_name":"` + strings.Repeat("x", 17000) + `"}`, "application/json", 400},
		{"GET", "/api/v1/lab-requests?status=invalid", "", "", 400},
		{"GET", "/api/v1/lab-requests?limit=101", "", "", 400},
		{"GET", "/api/v1/laboratories?offset=-1", "", "", 400},
		{"GET", "/api/v1/lab-results/not-a-uuid", "", "", 400},
		{"POST", "/api/v1/lab-requests/" + id + "/approval", `{"decision":"approve"} {}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests/" + id + "/approval", `{"decision":"cancel"}`, "application/json", 400},
		{"PATCH", "/api/v1/lab-requests/" + id + "/status", `{"status":"completed"}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests/" + id + "/result", `{"value":"","flag":"normal"}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests/" + id + "/result", `{"value":"42","flag":"critical"}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests/" + id + "/result", `{"value":"42\u0000","flag":"normal"}`, "application/json", 400},
		{"POST", "/api/v1/lab-requests/" + id + "/result", `{"value":"42","flag":"normal","test_name":"spoof"}`, "application/json", 400},
	} {
		t.Run(tc.method+tc.path+tc.body[:min(len(tc.body), 20)], func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.media)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
	if !validResult(ResultInput{Value: "Non-numeric synthetic value", Flag: "unknown"}) {
		t.Fatal("text result rejected")
	}
	if validResult(ResultInput{Value: strings.Repeat("a", 501), Flag: "normal"}) || validResult(ResultInput{Value: "42", Flag: "normal", Note: strings.Repeat("a", 2001)}) {
		t.Fatal("oversized result accepted")
	}
}

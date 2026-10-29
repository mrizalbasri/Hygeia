package care

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimelineQueryValidation(t *testing.T) {
	mux := http.NewServeMux()
	(&Handler{}).Register(mux)
	for _, query := range []string{
		"kind=secret", "kind=Episode", "limit=0", "limit=101", "limit=-1", "limit=abc", "limit=",
		"before_seq=0", "before_seq=-1", "before_seq=1.5", "before_seq=abc", "before_seq=", "before_seq=9223372036854775808",
		"patient_id=someone", "patient_ref=TEST-other", "chain=someone", "offset=1", "sort=asc",
		"kind=episode&kind=consent", "limit=1&limit=2", "before_seq=5&before_seq=10", "kind=%zz", "kind=episode;limit=1",
	} {
		t.Run(query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/patient/timeline?"+query, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != 400 {
				t.Fatalf("want 400 got %d: %s", w.Code, w.Body.String())
			}
		})
	}
	for _, query := range []string{"", "kind=", "kind=episode", "kind=laboratory&limit=100", "kind=prescription&before_seq=42", "kind=consent&before_seq=9223372036854775807&limit=1"} {
		r := httptest.NewRequest("GET", "/api/v1/patient/timeline?"+query, nil)
		options, ok := timelineOptions(r)
		if !ok || options.Limit < 1 {
			t.Fatalf("valid query rejected: %s", query)
		}
	}
}

package prescription

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidPrescriptionRequests(t *testing.T) {
	h := &Handler{}
	mux := http.NewServeMux()
	h.Register(mux)
	id := "00000000-0000-0000-0000-000000000001"
	for _, tc := range []struct {
		method, path, body, media string
		status                    int
	}{
		{"POST", "/api/v1/prescriptions", `{}`, "text/plain", 415},
		{"POST", "/api/v1/prescriptions", `{}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"episode_id":"` + id + `","items":[]}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"episode_id":"` + id + `","items":[{"medication_name":"DEMO","dosage":"demo","instructions":"demo","quantity":1}],"pharmacy_org_id":"` + id + `"}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"episode_id":"` + id + `","items":[{"medication_name":"DEMO","dosage":"demo","instructions":"demo","quantity":0}]}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"episode_id":"` + id + `","items":[{"medication_name":"DEMO","dosage":"demo","instructions":"demo","quantity":1.5}]}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"episode_id":"` + id + `","items":[{"medication_name":"DEMO","dosage":"demo","instructions":"demo","quantity":1,"patient_id":"spoof"}]}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"episode_id":"` + id + `","items":[{"medication_name":"DEMO\u0000","dosage":"demo","instructions":"demo","quantity":1}]}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions", `{"items":"` + strings.Repeat("a", 65537) + `"}`, "application/json", 400},
		{"GET", "/api/v1/prescriptions?status=invalid", "", "", 400},
		{"GET", "/api/v1/prescriptions?limit=101", "", "", 400},
		{"GET", "/api/v1/prescriptions?offset=-1", "", "", 400},
		{"GET", "/api/v1/pharmacies?offset=10001", "", "", 400},
		{"GET", "/api/v1/prescriptions/not-a-uuid", "", "", 400},
		{"POST", "/api/v1/prescriptions/" + id + "/approval", `{"decision":"approve"}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions/" + id + "/approval", `{"decision":"approve","pharmacy_org_id":"bad"}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions/" + id + "/approval", `{"decision":"reject","pharmacy_org_id":"` + id + `"}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions/" + id + "/approval", `{"decision":"cancel"}`, "application/json", 400},
		{"POST", "/api/v1/prescriptions/" + id + "/approval", `{"decision":"reject"} {}`, "application/json", 400},
		{"PATCH", "/api/v1/prescriptions/" + id + "/status", `{"status":"sent"}`, "application/json", 400},
		{"PATCH", "/api/v1/prescriptions/" + id + "/status", `{"status":"accepted","pharmacy_org_id":"` + id + `"}`, "application/json", 400},
	} {
		t.Run(tc.method+tc.path+tc.body[:min(len(tc.body), 30)], func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.media)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("want %d got %d: %s", tc.status, w.Code, w.Body.String())
			}
		})
	}
}

func TestPrescriptionItemBounds(t *testing.T) {
	item := Item{MedicationName: "Obat sintetis", Dosage: "demo", Instructions: "Untuk demo saja", Quantity: 1}
	input := CreateInput{EpisodeID: "00000000-0000-0000-0000-000000000001", Items: []Item{item}}
	if !validCreate(input) {
		t.Fatal("valid synthetic item rejected")
	}
	for _, items := range [][]Item{nil, make([]Item, 21)} {
		input.Items = items
		if validCreate(input) {
			t.Fatal("invalid item count accepted")
		}
	}
	for _, bad := range []Item{
		{MedicationName: strings.Repeat("a", 201), Dosage: "x", Instructions: "x", Quantity: 1},
		{MedicationName: "x", Dosage: " ", Instructions: "x", Quantity: 1},
		{MedicationName: "x", Dosage: strings.Repeat("a", 201), Instructions: "x", Quantity: 1},
		{MedicationName: "x", Dosage: "x", Instructions: strings.Repeat("a", 1001), Quantity: 1},
		{MedicationName: "x", Dosage: "x", Instructions: "x", Quantity: 10001},
		{MedicationName: "x", Dosage: "x", Instructions: "x", Quantity: -1},
	} {
		input.Items = []Item{bad}
		if validCreate(input) {
			t.Fatal("invalid item accepted")
		}
	}
}

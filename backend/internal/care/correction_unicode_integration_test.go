package care

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
)

func TestCorrectionUnicodeBoundariesIntegration(t *testing.T) {
	f := integrationFixture(t)
	episode := activeLabEpisode(t, f)
	request := newLabRequest(t, f, episode)
	approveLab(t, f, request.ID)
	processLab(t, f, request.ID)
	result := submitLab(t, f, request.ID)
	for _, tc := range []struct {
		char           string
		length, status int
	}{
		{"界", 4, 400}, {"界", 5, 201}, {"界", 500, 201}, {"界", 501, 400},
		{"😀", 4, 400}, {"😀", 5, 201}, {"😀", 500, 201}, {"😀", 501, 400},
	} {
		body, _ := json.Marshal(laboratory.CorrectionInput{Reason: " \t" + strings.Repeat(tc.char, tc.length) + "\n ", Value: "SYNTHETIC", Flag: "normal"})
		expectStatus(t, f.request("lab", "POST", "/api/v1/lab-results/"+result.ID+"/corrections", string(body), true), tc.status)
	}
}

package carepass

import "testing"

func TestShareBase(t *testing.T) {
	for _, v := range []string{"", "http://example.com", "https://user:pass@example.com", "https://example.com/path", "https://example.com?token=x", "https://example.com#token=x", "javascript:alert(1)"} {
		if _, err := ShareBase(v, true); err == nil {
			t.Errorf("accepted %q", v)
		}
	}
	if v, err := ShareBase("https://example.com/", true); err != nil || v != "https://example.com" {
		t.Fatal(v, err)
	}
}

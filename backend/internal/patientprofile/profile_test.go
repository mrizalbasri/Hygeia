package patientprofile

import (
	"strings"
	"testing"
	"time"
)

func TestProfileValidation(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	base := Input{FullName: "Pasien Sintetis", Sex: "unknown"}
	tests := []struct {
		name   string
		change func(*Input)
		valid  bool
	}{
		{"optional fields empty", func(*Input) {}, true},
		{"leap date", func(in *Input) { in.BirthDate = "2000-02-29" }, true},
		{"today", func(in *Input) { in.BirthDate = "2026-10-23" }, true},
		{"future", func(in *Input) { in.BirthDate = "2026-10-24" }, false},
		{"invalid date", func(in *Input) { in.BirthDate = "2001-02-29" }, false},
		{"non ISO date", func(in *Input) { in.BirthDate = "01/02/2000" }, false},
		{"minimum date", func(in *Input) { in.BirthDate = "1900-01-01" }, true},
		{"too old", func(in *Input) { in.BirthDate = "1899-12-31" }, false},
		{"blank name", func(in *Input) { in.FullName = "  " }, false},
		{"Unicode name", func(in *Input) { in.FullName = strings.Repeat("漢", 200) }, true},
		{"long name", func(in *Input) { in.FullName = strings.Repeat("a", 201) }, false},
		{"invalid sex", func(in *Input) { in.Sex = "invalid" }, false},
		{"formatted phone", func(in *Input) { in.Phone = "+62 (812) 3456-7890" }, true},
		{"short phone", func(in *Input) { in.Phone = "123" }, false},
		{"long phone", func(in *Input) { in.Phone = "1234567890123456" }, false},
		{"letters in phone", func(in *Input) { in.Phone = "0812abc6789" }, false},
		{"misplaced plus", func(in *Input) { in.Phone = "0812+3456789" }, false},
		{"long address", func(in *Input) { in.Address = strings.Repeat("a", 1001) }, false},
		{"NUL address", func(in *Input) { in.Address = "test\x00address" }, false},
		{"invalid UTF8", func(in *Input) { in.Address = "\xff" }, false},
		{"emergency name only", func(in *Input) { in.EmergencyContactName = "Kontak Sintetis" }, false},
		{"emergency phone only", func(in *Input) { in.EmergencyContactPhone = "08123456789" }, false},
		{"emergency pair", func(in *Input) {
			in.EmergencyContactName = "Kontak Sintetis"
			in.EmergencyContactPhone = "08123456789"
		}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := base
			test.change(&in)
			in.normalize()
			if got := in.valid(now); got != test.valid {
				t.Fatalf("valid=%v; want %v", got, test.valid)
			}
		})
	}
}

func TestProfileNormalization(t *testing.T) {
	in := Input{
		FullName: "  Pasien Sintetis  ", Sex: "female", Phone: " 08123456789 ",
		Address: "  Jalan Sintetis\nBlok Demo  ", EmergencyContactName: "  Kontak Sintetis ",
		EmergencyContactPhone: " 08123456780 ",
	}
	in.normalize()
	if in.FullName != "Pasien Sintetis" || in.Address != "Jalan Sintetis\nBlok Demo" ||
		in.Phone != "08123456789" || in.EmergencyContactName != "Kontak Sintetis" ||
		in.EmergencyContactPhone != "08123456780" {
		t.Fatalf("unexpected normalization: %+v", in)
	}
}

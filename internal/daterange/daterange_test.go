package daterange

import (
	"strings"
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func TestParseSingleDate(t *testing.T) {
	got, err := Parse([]string{"20201218"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || !got[0].Equal(mustDate(t, "20201218")) {
		t.Fatalf("got %v, want a single 20201218", got)
	}
}

func TestParseRangeIsInclusive(t *testing.T) {
	got, err := Parse([]string{"20201218", "20201221"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"20201218", "20201219", "20201220", "20201221"}
	if len(got) != len(want) {
		t.Fatalf("got %d dates, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Format(Format) != w {
			t.Errorf("date %d = %s, want %s", i, got[i].Format(Format), w)
		}
	}
}

func TestParseRangeReversedIsNormalized(t *testing.T) {
	got, err := Parse([]string{"20201221", "20201218"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 4 || got[0].Format(Format) != "20201218" {
		t.Fatalf("got %v, want ascending from 20201218", got)
	}
}

func TestParseRangeCrossingDST(t *testing.T) {
	// AddDate は暦日ベースなので、DSTのある地域でも日付が飛んだり重複したりしない。
	got, err := Parse([]string{"20210313", "20210315"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"20210313", "20210314", "20210315"}
	if len(got) != len(want) {
		t.Fatalf("got %d dates, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Format(Format) != w {
			t.Errorf("date %d = %s, want %s", i, got[i].Format(Format), w)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   []string
	}{
		{"no values", nil},
		{"three values", []string{"20201218", "20201219", "20201220"}},
		{"malformed", []string{"2020-12-18"}},
		{"malformed second value", []string{"20201218", "nope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(tt.in); err == nil {
				t.Fatalf("Parse(%v) = nil error, want an error", tt.in)
			}
		})
	}
}

func TestEachRunsEveryDateAndAggregatesErrors(t *testing.T) {
	var seen []string
	err := Each([]string{"20201218", "20201220"}, func(d time.Time) error {
		seen = append(seen, d.Format(Format))
		if d.Format(Format) == "20201219" {
			return errTest
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected the failing date to surface as an error")
	}
	// 途中の失敗で打ち切らず、3日分すべて処理していること。
	if len(seen) != 3 {
		t.Fatalf("processed %v, want all 3 dates", seen)
	}
	if !strings.Contains(err.Error(), "20201219") {
		t.Errorf("error = %v, want it to name the failing date", err)
	}
}

var errTest = errTestType{}

type errTestType struct{}

func (errTestType) Error() string { return "test failure" }

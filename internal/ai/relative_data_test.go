package ai

import (
	"testing"
	"time"
)

func TestResolveStructuredExpenseDate(t *testing.T) {
	location := time.FixedZone("UTC+3", 360*60)
	now := time.Date(2026, time.October, 22, 16, 30, 0, 0, location)

	tests := []struct {
		name string
		args *expenseDateArgs
		want time.Time
	}{
		{
			name: "unspecified",
			args: nil,
			want: time.Date(2026, time.October, 22, 0, 0, 0, 0, location),
		},
		{
			name: "explicitly unspecified",
			args: &expenseDateArgs{Type: "unspecified"},
			want: time.Date(2026, time.October, 22, 0, 0, 0, 0, location),
		},
		{
			name: "relative past",
			args: &expenseDateArgs{
				Type:       "relative",
				OffsetDays: intPointer(-2),
			},
			want: time.Date(2026, time.October, 20, 0, 0, 0, 0, location),
		},
		{
			name: "future date",
			args: &expenseDateArgs{
				Type:       "relative",
				OffsetDays: intPointer(2),
			},
			want: time.Date(2026, time.October, 24, 0, 0, 0, 0, location),
		},
		{
			name: "calendar date with year",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Day:   intPointer(14),
				Month: intPointer(2),
				Year:  intPointer(2024),
			},
			want: time.Date(2024, time.February, 14, 0, 0, 0, 0, location),
		},
		{
			name: "calendar date without year",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Day:   intPointer(14),
				Month: intPointer(2),
			},
			want: time.Date(2026, time.February, 14, 0, 0, 0, 0, location),
		},
		{
			name: "leap day",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Day:   intPointer(29),
				Month: intPointer(2),
				Year:  intPointer(2024),
			},
			want: time.Date(2024, time.February, 29, 0, 0, 0, 0, location),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveStructuredExpenseDateAt(tt.args, location, now)
			if err != nil {
				t.Fatalf("resolveStructuredExpenseDateAt: %v", err)
			}

			if !got.Equal(tt.want) {
				t.Fatalf("date = %v, want %v", got, tt.want)
			}

			if got.Location() != location {
				t.Fatalf("location = %v, want %v", got.Location(), location)
			}
		})
	}

}

func TestResolveStructuredExpenseDateRejectsIncompleteAndInvalidDates(t *testing.T) {
	location := time.FixedZone("UTC+3", 360*60)
	now := time.Date(2026, time.October, 22, 16, 30, 0, 0, location)

	tests := []struct {
		name string
		args *expenseDateArgs
	}{
		{
			name: "relative date without offset",
			args: &expenseDateArgs{Type: "relative"},
		},
		{
			name: "calendar date without month",
			args: &expenseDateArgs{
				Type: "calendar_date",
				Day:  intPointer(1),
			},
		},
		{
			name: "calendar date without day",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Month: intPointer(1),
			},
		},
		{
			name: "day exceeds month length",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Day:   intPointer(31),
				Month: intPointer(4),
				Year:  intPointer(2025),
			},
		},
		{
			name: "invalid month",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Day:   intPointer(1),
				Month: intPointer(13),
				Year:  intPointer(2025),
			},
		},
		{
			name: "invalid leap day",
			args: &expenseDateArgs{
				Type:  "calendar_date",
				Day:   intPointer(29),
				Month: intPointer(2),
				Year:  intPointer(2025),
			},
		},
		{
			name: "unsupported date type",
			args: &expenseDateArgs{Type: "unsupported"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := resolveStructuredExpenseDateAt(tt.args, location, now); err == nil {
				t.Fatalf("expected error for %#v", tt.args)
			}
		})
	}

}

func intPointer(value int) *int {
	return &value
}

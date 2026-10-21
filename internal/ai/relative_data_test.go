package ai

import (
	"testing"
	"time"
)

func TestResolveStructuredExpenseDate(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)
	tests := []struct {
		name string
		args *expenseDateArgs
		want time.Time
	}{
		{"unspecified", nil, startOfDay(time.Now().In(location))},
		{"relative past", &expenseDateArgs{Type: "relative", OffsetDays: intPointer(-2)}, startOfDay(time.Now().In(location)).AddDate(0, 0, -2)},
		{"calendar date", &expenseDateArgs{Type: "calendar_date", Day: intPointer(14), Month: intPointer(2), Year: intPointer(2024)}, time.Date(2024, time.February, 14, 0, 0, 0, 0, location)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveStructuredExpenseDate(tt.args, location)
			if err != nil {
				t.Fatalf("resolveStructuredExpenseDate: %v", err)
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
	tests := []*expenseDateArgs{
		{Type: "relative"},
		{Type: "calendar_date", Day: intPointer(1)},
		{Type: "calendar_date", Day: intPointer(31), Month: intPointer(4), Year: intPointer(2025)},
		{Type: "calendar_date", Day: intPointer(1), Month: intPointer(13), Year: intPointer(2025)},
		{Type: "unsupported"},
	}
	for _, args := range tests {
		if _, err := resolveStructuredExpenseDate(args, time.UTC); err == nil {
			t.Errorf("expected error for %#v", args)
		}
	}
}

func TestResolveRelativeDateAndPeriod(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)
	now := time.Date(2026, time.October, 21, 16, 30, 0, 0, location)
	got, ok := resolveRelativeDate("траты 2 дня назад", now)
	if !ok || !got.Equal(time.Date(2026, time.October, 19, 0, 0, 0, 0, location)) {
		t.Fatalf("relative date = %v, ok=%v", got, ok)
	}
	from, to, matched := resolveExpensePeriod("за последние 5 дней", now)
	if !matched || from == nil || to == nil {
		t.Fatal("expected period to be matched")
	}
	if from.Format("2006-01-02") != "2026-10-17" || to.Format("2006-01-02 15:04:05.999999999") != "2026-10-21 23:59:59.999999999" {
		t.Fatalf("period = %v .. %v", from, to)
	}
}

func intPointer(value int) *int { return &value }

func TestHasExpenseDate(t *testing.T) {
	now := time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
	for _, message := range []string{"сегодня", "2 дня назад", "2026-10-15", "15.10.2026", "15/10/2026"} {
		if !hasExpenseDate(message, now) {
			t.Errorf("hasExpenseDate(%q) = false", message)
		}
	}
	if hasExpenseDate("кофе 350", now) {
		t.Fatal("unexpected date in plain expense message")
	}
}

func TestResolveExpenseDate(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)
	tests := []struct{ message, want string }{
		{"вчера купил кофе", "2026-10-20"},
		{"оплата 2026-10-05", "2026-10-05"},
		{"покупка 05.10.2026", "2026-10-05"},
		{"кофе без даты", "2026-10-21"},
	}
	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got, err := resolveExpenseDate(tt.message, location)
			if err != nil {
				t.Fatalf("resolveExpenseDate: %v", err)
			}
			if got.Format("2006-01-02") != tt.want {
				t.Fatalf("date = %s, want %s", got.Format("2006-01-02"), tt.want)
			}
			if got.Location() != location {
				t.Fatalf("location = %v, want %v", got.Location(), location)
			}
		})
	}
	if _, err := resolveExpenseDate("дата 2026-02-31", location); err == nil {
		t.Fatal("expected invalid ISO date error")
	}
	if _, err := resolveExpenseDate("дата 31.02.2026", location); err == nil {
		t.Fatal("expected invalid local date error")
	}
}

func TestResolveExpensePeriodCalendarRanges(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)
	now := time.Date(2026, time.October, 21, 16, 30, 0, 0, location)
	tests := []struct{ message, from, to string }{
		{"за последние 2 недели", "2026-10-08", "2026-10-21"},
		{"за последние 2 месяца", "2026-08-21", "2026-10-21"},
		{"за эту неделю", "2026-10-19", "2026-10-21"},
		{"за прошлую неделю", "2026-10-12", "2026-10-18"},
		{"за этот месяц", "2026-10-01", "2026-10-21"},
		{"за прошлый месяц", "2026-09-01", "2026-09-30"},
		{"за 2026-10-05", "2026-10-05", "2026-10-05"},
		{"за 05.10.2026", "2026-10-05", "2026-10-05"},
	}
	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			from, to, matched := resolveExpensePeriod(tt.message, now)
			if !matched || from == nil || to == nil {
				t.Fatalf("period not matched: %v %v %v", from, to, matched)
			}
			if from.Format("2006-01-02") != tt.from || to.Format("2006-01-02") != tt.to {
				t.Fatalf("range = %s .. %s, want %s .. %s", from.Format("2006-01-02"), to.Format("2006-01-02"), tt.from, tt.to)
			}
		})
	}
	if from, to, matched := resolveExpensePeriod("какие расходы", now); matched || from != nil || to != nil {
		t.Fatalf("unexpected match: %v %v %v", from, to, matched)
	}
}

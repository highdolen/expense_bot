package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	daysAgoPattern   = regexp.MustCompile(`(?i)(\d+)\s+д(?:ень|ня|ней|ни)\s+назад`)
	daysAfterPattern = regexp.MustCompile(`(?i)через\s+(\d+)\s+д(?:ень|ня|ней|ни)`)
	weeksAgoPattern  = regexp.MustCompile(`(?i)(\d+)\s+недел(?:ю|и|ь)\s+назад`)
	monthsAgoPattern = regexp.MustCompile(`(?i)(\d+)\s+месяц(?:а|ев)?\s+назад`)

	lastDaysPattern   = regexp.MustCompile(`(?i)(?:за\s+)?последние\s+(\d+)\s+д(?:ень|ня|ней|ни)`)
	lastWeeksPattern  = regexp.MustCompile(`(?i)(?:за\s+)?последние\s+(\d+)\s+недел(?:ю|и|ь)`)
	lastMonthsPattern = regexp.MustCompile(`(?i)(?:за\s+)?последние\s+(\d+)\s+месяц(?:а|ев)?`)

	isoDatePattern   = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	localDatePattern = regexp.MustCompile(`\b\d{1,2}[./]\d{1,2}[./]\d{4}\b`)
)

func startOfDay(date time.Time) time.Time {
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		0, 0, 0, 0, date.Location(),
	)
}

func endOfDay(date time.Time) time.Time {
	return startOfDay(date).
		AddDate(0, 0, 1).
		Add(-time.Nanosecond)
}

func resolveRelativeDate(message string, now time.Time) (time.Time, bool) {
	message = strings.ToLower(message)
	today := startOfDay(now)

	switch {
	case strings.Contains(message, "позавчера"):
		return today.AddDate(0, 0, -2), true
	case strings.Contains(message, "послезавтра"):
		return today.AddDate(0, 0, 2), true
	case strings.Contains(message, "вчера"):
		return today.AddDate(0, 0, -1), true
	case strings.Contains(message, "завтра"):
		return today.AddDate(0, 0, 1), true
	case strings.Contains(message, "сегодня"):
		return today, true
	case strings.Contains(message, "неделю назад"):
		return today.AddDate(0, 0, -7), true
	case strings.Contains(message, "месяц назад"):
		return today.AddDate(0, -1, 0), true
	}

	patterns := []struct {
		regex  *regexp.Regexp
		offset func(int) time.Time
	}{
		{
			regex: daysAgoPattern,
			offset: func(n int) time.Time {
				return today.AddDate(0, 0, -n)
			},
		},
		{
			regex: daysAfterPattern,
			offset: func(n int) time.Time {
				return today.AddDate(0, 0, n)
			},
		},
		{
			regex: weeksAgoPattern,
			offset: func(n int) time.Time {
				return today.AddDate(0, 0, -7*n)
			},
		},
		{
			regex: monthsAgoPattern,
			offset: func(n int) time.Time {
				return today.AddDate(0, -n, 0)
			},
		},
	}

	for _, pattern := range patterns {
		match := pattern.regex.FindStringSubmatch(message)
		if len(match) != 2 {
			continue
		}

		n, err := strconv.Atoi(match[1])
		if err != nil || n < 0 {
			return time.Time{}, false
		}

		return pattern.offset(n), true
	}

	return time.Time{}, false
}

func hasExpenseDate(message string, now time.Time) bool {
	if _, ok := resolveRelativeDate(message, now); ok {
		return true
	}

	return isoDatePattern.MatchString(message) ||
		localDatePattern.MatchString(message)
}

func resolveExpenseDate(
	message string,
	location *time.Location,
) (time.Time, error) {
	now := time.Now().In(location)

	if date, ok := resolveRelativeDate(message, now); ok {
		return date, nil
	}

	if match := isoDatePattern.FindString(message); match != "" {
		date, err := time.ParseInLocation("2006-01-02", match, location)
		if err != nil {
			return time.Time{}, fmt.Errorf(
				"invalid expense date %q: %w", match, err,
			)
		}
		return date, nil
	}

	if match := localDatePattern.FindString(message); match != "" {
		normalized := strings.ReplaceAll(match, "/", ".")
		date, err := time.ParseInLocation("02.01.2006", normalized, location)
		if err != nil {
			return time.Time{}, fmt.Errorf(
				"invalid expense date %q: %w", match, err,
			)
		}
		return date, nil
	}

	return startOfDay(now), nil
}

// resolveExpensePeriod определяет диапазон расходов по сообщению.
// Если распознать период не удалось, matched будет false.
func resolveExpensePeriod(
	message string,
	now time.Time,
) (from, to *time.Time, matched bool) {
	message = strings.ToLower(message)
	today := startOfDay(now)

	// Последние N календарных дней, включая сегодня.
	if match := lastDaysPattern.FindStringSubmatch(message); len(match) == 2 {
		n, err := strconv.Atoi(match[1])
		if err == nil && n > 0 {
			start := today.AddDate(0, 0, -(n - 1))
			end := endOfDay(today)
			return &start, &end, true
		}
	}

	// Последние N недель, включая сегодня.
	if match := lastWeeksPattern.FindStringSubmatch(message); len(match) == 2 {
		n, err := strconv.Atoi(match[1])
		if err == nil && n > 0 {
			start := today.AddDate(0, 0, -(n*7 - 1))
			end := endOfDay(today)
			return &start, &end, true
		}
	}

	// Последние N календарных месяцев.
	if match := lastMonthsPattern.FindStringSubmatch(message); len(match) == 2 {
		n, err := strconv.Atoi(match[1])
		if err == nil && n > 0 {
			start := today.AddDate(0, -n, 0)
			end := endOfDay(today)
			return &start, &end, true
		}
	}

	// Текущая неделя начинается в понедельник.
	if strings.Contains(message, "на этой неделе") ||
		strings.Contains(message, "за эту неделю") ||
		strings.Contains(message, "текущую неделю") {
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}

		start := today.AddDate(0, 0, -(weekday - 1))
		end := endOfDay(today)
		return &start, &end, true
	}

	// Предыдущая календарная неделя.
	if strings.Contains(message, "за прошлую неделю") ||
		strings.Contains(message, "на прошлой неделе") ||
		strings.Contains(message, "за прошлой неделей") {
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}

		thisMonday := today.AddDate(0, 0, -(weekday - 1))
		start := thisMonday.AddDate(0, 0, -7)
		end := thisMonday.Add(-time.Nanosecond)
		return &start, &end, true
	}

	// Текущий календарный месяц.
	if strings.Contains(message, "за этот месяц") ||
		strings.Contains(message, "за текущий месяц") ||
		strings.Contains(message, "в этом месяце") {
		start := time.Date(
			today.Year(), today.Month(), 1,
			0, 0, 0, 0, today.Location(),
		)
		end := endOfDay(today)
		return &start, &end, true
	}

	// Предыдущий календарный месяц.
	if strings.Contains(message, "за прошлый месяц") ||
		strings.Contains(message, "в прошлом месяце") {
		thisMonth := time.Date(
			today.Year(), today.Month(), 1,
			0, 0, 0, 0, today.Location(),
		)
		start := thisMonth.AddDate(0, -1, 0)
		end := thisMonth.Add(-time.Nanosecond)
		return &start, &end, true
	}

	// Конкретный день: вчера, позавчера, 2 дня назад и т. п.
	if date, ok := resolveRelativeDate(message, now); ok {
		start := startOfDay(date)
		end := endOfDay(start)
		return &start, &end, true
	}

	// Конкретная дата в формате YYYY-MM-DD или DD.MM.YYYY.
	if match := isoDatePattern.FindString(message); match != "" {
		date, err := time.ParseInLocation(
			"2006-01-02", match, now.Location(),
		)
		if err == nil {
			start := startOfDay(date)
			end := endOfDay(date)
			return &start, &end, true
		}
	}

	if match := localDatePattern.FindString(message); match != "" {
		normalized := strings.ReplaceAll(match, "/", ".")
		date, err := time.ParseInLocation(
			"02.01.2006", normalized, now.Location(),
		)
		if err == nil {
			start := startOfDay(date)
			end := endOfDay(date)
			return &start, &end, true
		}
	}

	return nil, nil, false
}

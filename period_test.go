package main

import (
	"testing"
	"time"

	"github.com/carlmjohnson/be"
)

func TestPeriodString(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected string
	}{
		{
			name:     "basic period",
			start:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "2023-12-01 - 2023-12-31",
		},
		{
			name:     "cross year period",
			start:    time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: "2023-12-15 - 2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Period{
				start: tt.start,
				end:   tt.end,
			}
			result := p.String()
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestPeriodStartDate(t *testing.T) {
	p := &Period{
		start: time.Date(2023, 12, 1, 10, 30, 45, 0, time.UTC),
		end:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	result := p.startDate()
	be.Equal(t, "2023-12-01", result)
}

func TestPeriodEndDate(t *testing.T) {
	p := &Period{
		start: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		end:   time.Date(2023, 12, 31, 10, 30, 45, 0, time.UTC),
	}

	result := p.endDate()
	be.Equal(t, "2023-12-31", result)
}

func TestPeriodSetPeriod(t *testing.T) {
	tests := []struct {
		name        string
		current     time.Time
		periodType  string
		expectStart time.Time
		expectEnd   time.Time
	}{
		{
			name:        "monthly period - mid month",
			current:     time.Date(2023, 12, 15, 10, 30, 0, 0, time.UTC),
			periodType:  monthlyPeriodType,
			expectStart: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:        "monthly period - start of month",
			current:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			periodType:  monthlyPeriodType,
			expectStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:        "annual period",
			current:     time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
			periodType:  annualPeriodType,
			expectStart: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:        "default to monthly",
			current:     time.Date(2023, 12, 15, 10, 30, 0, 0, time.UTC),
			periodType:  "invalid",
			expectStart: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			expectEnd:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Period{}
			p.setPeriod(tt.current, tt.periodType)

			be.Equal(t, tt.expectStart, p.start)
			be.Equal(t, tt.expectEnd, p.end)
		})
	}
}

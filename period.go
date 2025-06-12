package main

import (
	"fmt"
	"time"
)

type Period struct {
	start time.Time
	end   time.Time
}

func (p *Period) String() string {
	return fmt.Sprintf("%s - %s", p.start.Format("2006-01-02"), p.end.Format("2006-01-02"))
}

func (p *Period) startDate() string {
	return p.start.Format("2006-01-02")
}

func (p *Period) endDate() string {
	return p.end.Format("2006-01-02")
}

func (p *Period) setPeriod(current time.Time, periodType string) {
	switch periodType {
	case monthlyPeriodType:
		p.start = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
		p.end = time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location()).Add(-time.Second)
	case annualPeriodType:
		p.start = time.Date(current.Year(), 1, 1, 0, 0, 0, 0, current.Location())
		p.end = time.Date(current.Year()+1, 1, 1, 0, 0, 0, 0, current.Location()).Add(-time.Second)
	default:
		// default to month
		p.start = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
		p.end = time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location()).Add(-time.Second)
	}
}

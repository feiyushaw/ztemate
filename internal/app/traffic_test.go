package app

import (
	"testing"
	"time"
)

func TestTrafficCounterResetAndAlert(t *testing.T) {
	p := defaultPlan()
	p.Enabled = true
	p.Unit = "MB"
	p.Limit = 100
	p.AlertPercent = 80
	l := TrafficLedger{Plan: p}
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	v := 70 * multiplier("MB")
	if l.advance(now, &v, nil) {
		t.Fatal("premature alert")
	}
	v = 85 * multiplier("MB")
	if !l.advance(now, &v, nil) {
		t.Fatal("missing alert")
	}
	if l.advance(now, &v, nil) {
		t.Fatal("duplicate alert")
	}
	v = 5 * multiplier("MB")
	l.advance(now, &v, nil)
	if l.view().Used != 90 {
		t.Fatalf("counter reset lost usage: %v", l.view().Used)
	}
}
func TestTrafficBillingRolloverAndShortMonth(t *testing.T) {
	p := defaultPlan()
	p.AutoReset = true
	p.ResetDay = 31
	l := TrafficLedger{Plan: p}
	v := 100.0
	l.advance(time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC), &v, nil)
	v = 150
	l.advance(time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC), &v, nil)
	if l.UsedData != 0 || l.Cycle != "2026-02-28" {
		t.Fatalf("rollover %#v", l)
	}
	v = 160
	l.advance(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), &v, nil)
	if l.UsedData != 10 {
		t.Fatal("short month cycle drift")
	}
}

func TestTrafficMissingCounterKeepsBaseline(t *testing.T) {
	l := TrafficLedger{Plan: defaultPlan()}
	now := time.Now()
	value := 100.0
	l.advance(now, &value, nil)
	l.advance(now, nil, nil)
	value = 150
	l.advance(now, &value, nil)
	if l.UsedData != 150 {
		t.Fatalf("missing sample duplicated usage: %v", l.UsedData)
	}
}

func TestManualUsageBecomesBaselineOnFirstConnection(t *testing.T) {
	l := TrafficLedger{Plan: defaultPlan(), UsedData: 7.15 * multiplier("GB"), Calibrated: true}
	value := 40.0 * multiplier("GB")
	l.advance(time.Now(), &value, nil)
	if l.view().Used != 7.15 {
		t.Fatalf("manual calibration overwritten: %v", l.view().Used)
	}
	value += multiplier("GB")
	l.advance(time.Now(), &value, nil)
	if l.view().Used != 8.15 {
		t.Fatalf("increment not counted: %v", l.view().Used)
	}
}

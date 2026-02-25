package ui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{60, "1.0m"},
		{90, "1.5m"},
		{3600, "1.0h"},
		{3660, "1.0h"},
		{7200, "2.0h"},
		{5400, "1.5h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDuration(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatTokens(tt.input)
			if result != tt.expected {
				t.Errorf("FormatTokens(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "$0.00"},
		{0.5, "$0.50"},
		{1.0, "$1.00"},
		{10.5, "$10.50"},
		{100.25, "$100.25"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatCost(tt.input)
			if result != tt.expected {
				t.Errorf("FormatCost(%v) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	// Test with a known timestamp (using local timezone)
	ts := time.Date(2026, 2, 24, 22, 55, 0, 0, time.Local).Unix()
	expected := "2026-02-24 22:55"

	result := FormatDateTime(ts)
	if result != expected {
		t.Errorf("FormatDateTime(%d) = %s; want %s", ts, result, expected)
	}

	// Test with zero timestamp
	zeroResult := FormatDateTime(0)
	if zeroResult != "-" {
		t.Errorf("FormatDateTime(0) = %s; want -", zeroResult)
	}
}

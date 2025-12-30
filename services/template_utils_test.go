package services

import (
	"testing"
)

func TestToOrdinal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1", "1st"},
		{"2", "2nd"},
		{"3", "3rd"},
		{"4", "4th"},
		{"10", "10th"},
		{"11", "11th"},
		{"12", "12th"},
		{"13", "13th"},
		{"21", "21st"},
		{"22", "22nd"},
		{"23", "23rd"},
		{"101", "101st"},
		{"invalid", "invalid"}, // 非数字应返回原值
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toOrdinal(tt.input)
			if result != tt.expected {
				t.Errorf("toOrdinal(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatWithSign(t *testing.T) {
	tests := []struct {
		value    string
		negate   bool
		expected string
	}{
		{"2.5", false, "+2.5"},
		{"2.5", true, "-2.5"},
		{"-1.5", false, "-1.5"},
		{"-1.5", true, "+1.5"},
		{"0", false, "+0"},
		{"0", true, "+0"},
		{"invalid", false, "invalid"}, // 非数字应返回原值
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := formatWithSign(tt.value, tt.negate)
			if result != tt.expected {
				t.Errorf("formatWithSign(%s, %v) = %s; want %s", tt.value, tt.negate, result, tt.expected)
			}
		})
	}
}

func TestReplaceSpecifiers(t *testing.T) {
	tests := []struct {
		name       string
		specifiers string
		expected   string
	}{
		{
			name:       "Race to {pointnr} points",
			specifiers: "pointnr=3",
			expected:   "Race to 3 points",
		},
		{
			name:       "{!periodnr} period - total",
			specifiers: "periodnr=2",
			expected:   "2nd period - total",
		},
		{
			name:       "Handicap {+hcp}",
			specifiers: "hcp=2.5",
			expected:   "Handicap +2.5",
		},
		{
			name:       "Handicap {-hcp}",
			specifiers: "hcp=2.5",
			expected:   "Handicap -2.5",
		},
		{
			name:       "{!periodnr} period - {$competitor1} total {+hcp}",
			specifiers: "periodnr=1|hcp=1.5",
			expected:   "1st period - {$competitor1} total +1.5",
		},
		{
			name:       "No specifiers",
			specifiers: "",
			expected:   "No specifiers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceSpecifiers(tt.name, tt.specifiers)
			if result != tt.expected {
				t.Errorf("replaceSpecifiers(%s, %s) = %s; want %s", tt.name, tt.specifiers, result, tt.expected)
			}
		})
	}
}

func TestReplaceCompetitors(t *testing.T) {
	tests := []struct {
		name     string
		homeTeam string
		awayTeam string
		expected string
	}{
		{
			name:     "{$competitor1} to win",
			homeTeam: "Team A",
			awayTeam: "Team B",
			expected: "Team A to win",
		},
		{
			name:     "{$competitor2} to win",
			homeTeam: "Team A",
			awayTeam: "Team B",
			expected: "Team B to win",
		},
		{
			name:     "{$competitor1} vs {$competitor2}",
			homeTeam: "Team A",
			awayTeam: "Team B",
			expected: "Team A vs Team B",
		},
		{
			name:     "No competitors",
			homeTeam: "Team A",
			awayTeam: "Team B",
			expected: "No competitors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceCompetitors(tt.name, tt.homeTeam, tt.awayTeam)
			if result != tt.expected {
				t.Errorf("replaceCompetitors(%s, %s, %s) = %s; want %s", tt.name, tt.homeTeam, tt.awayTeam, result, tt.expected)
			}
		})
	}
}

func TestParseSpecifiers(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			input:    "pointnr=3",
			expected: map[string]string{"pointnr": "3"},
		},
		{
			input: "pointnr=3|hcp=-1.5",
			expected: map[string]string{
				"pointnr": "3",
				"hcp":     "-1.5",
			},
		},
		{
			input:    "",
			expected: map[string]string{},
		},
		{
			input:    "invalid",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSpecifiers(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseSpecifiers(%s) returned %d items; want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("parseSpecifiers(%s)[%s] = %s; want %s", tt.input, key, result[key], expectedValue)
				}
			}
		})
	}
}

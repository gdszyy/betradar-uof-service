package services

import (
	"testing"

	"betradar-uof-service/database"
)

// TestParseSpecifiers tests the specifier parsing function
func TestParseSpecifiers(t *testing.T) {
	service := &MarketTabChipService{}

	tests := []struct {
		input    string
		expected int
		name     string
	}{
		{
			input:    "quarternr=1",
			expected: 1,
			name:     "Single specifier",
		},
		{
			input:    "quarternr=1,goalnr=2",
			expected: 2,
			name:     "Multiple specifiers",
		},
		{
			input:    "setnr=1, gamenr=2, pointnr=3",
			expected: 3,
			name:     "Specifiers with spaces",
		},
		{
			input:    "",
			expected: 0,
			name:     "Empty string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := service.parseSpecifiers(test.input)
			if len(result) != test.expected {
				t.Errorf("Expected %d specifiers, got %d", test.expected, len(result))
			}
		})
	}
}

// TestDetermineTabID tests the tab determination logic
func TestDetermineTabID(t *testing.T) {
	service := &MarketTabChipService{}

	tests := []struct {
		groups   []string
		specs    []database.SpecifierPair
		expected string
		name     string
	}{
		{
			groups:   []string{"regular_play"},
			specs:    []database.SpecifierPair{},
			expected: "regular_play",
			name:     "Group-based tab: regular_play",
		},
		{
			groups:   []string{"player_props"},
			specs:    []database.SpecifierPair{},
			expected: "player_props",
			name:     "Group-based tab: player_props",
		},
		{
			groups: []string{},
			specs: []database.SpecifierPair{
				{Name: "quarternr", Value: "1"},
			},
			expected: "quarters",
			name:     "Specifier-based tab: quarters",
		},
		{
			groups: []string{},
			specs: []database.SpecifierPair{
				{Name: "setnr", Value: "1"},
			},
			expected: "sets",
			name:     "Specifier-based tab: sets",
		},
		{
			groups:   []string{},
			specs:    []database.SpecifierPair{},
			expected: "regular_play",
			name:     "Default tab: regular_play",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.determineTabID(test.groups, test.specs, "")
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != test.expected {
				t.Errorf("Expected tab %s, got %s", test.expected, result)
			}
		})
	}
}

// TestDetermineChipID tests the chip determination logic
func TestDetermineChipID(t *testing.T) {
	service := &MarketTabChipService{}

	tests := []struct {
		tabID    string
		specs    []database.SpecifierPair
		expected string
		name     string
	}{
		{
			tabID: "quarters",
			specs: []database.SpecifierPair{
				{Name: "quarternr", Value: "1"},
			},
			expected: "quarters_quarternr_1",
			name:     "Chip for quarters tab",
		},
		{
			tabID: "1st_half",
			specs: []database.SpecifierPair{
				{Name: "goalnr", Value: "2"},
			},
			expected: "1st_half_goalnr_2",
			name:     "Chip for 1st_half tab",
		},
		{
			tabID:    "regular_play",
			specs:    []database.SpecifierPair{},
			expected: "",
			name:     "No chip for regular_play tab",
		},
		{
			tabID: "sets",
			specs: []database.SpecifierPair{
				{Name: "setnr", Value: "1"},
				{Name: "gamenr", Value: "2"},
			},
			expected: "sets_setnr_1",
			name:     "Chip uses primary specifier",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := service.determineChipID(test.tabID, test.specs)
			if result != test.expected {
				t.Errorf("Expected chip %s, got %s", test.expected, result)
			}
		})
	}
}

// TestSpecifierPairMarshaling tests JSON marshaling of specifier pairs
func TestSpecifierPairMarshaling(t *testing.T) {
	specs := database.SpecifiersArray{
		{Name: "quarternr", Value: "1"},
		{Name: "goalnr", Value: "2"},
	}

	// Test marshaling
	data, err := specs.Value()
	if err != nil {
		t.Errorf("Failed to marshal specifiers: %v", err)
	}

	// Test unmarshaling
	var result database.SpecifiersArray
	err = result.Scan(data)
	if err != nil {
		t.Errorf("Failed to unmarshal specifiers: %v", err)
	}

	if len(result) != len(specs) {
		t.Errorf("Expected %d specifiers, got %d", len(specs), len(result))
	}
}

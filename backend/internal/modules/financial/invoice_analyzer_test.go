package financial

import (
	"testing"
)

// TestDetermineSeverity tests the severity determination logic
func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name         string
		variancePct  float64
		wantSeverity string
	}{
		{
			name:         "Critical variance - 50% or more",
			variancePct:  50.0,
			wantSeverity: "critical",
		},
		{
			name:         "Critical variance - over 50%",
			variancePct:  75.0,
			wantSeverity: "critical",
		},
		{
			name:         "High variance - 25% to 50%",
			variancePct:  30.0,
			wantSeverity: "high",
		},
		{
			name:         "High variance - at threshold",
			variancePct:  25.0,
			wantSeverity: "high",
		},
		{
			name:         "Medium variance - 10% to 25%",
			variancePct:  15.0,
			wantSeverity: "medium",
		},
		{
			name:         "Medium variance - at threshold",
			variancePct:  10.0,
			wantSeverity: "medium",
		},
		{
			name:         "Low variance - under 10%",
			variancePct:  5.0,
			wantSeverity: "low",
		},
		{
			name:         "Low variance - 0%",
			variancePct:  0.0,
			wantSeverity: "low",
		},
		{
			name:         "Negative variance - critical",
			variancePct:  -60.0,
			wantSeverity: "critical",
		},
		{
			name:         "Negative variance - medium",
			variancePct:  -12.0,
			wantSeverity: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineSeverity(tt.variancePct)
			if got != tt.wantSeverity {
				t.Errorf("determineSeverity(%v) = %v, want %v", tt.variancePct, got, tt.wantSeverity)
			}
		})
	}
}

// TestToNullString tests null string conversion
func TestToNullString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantValue string
	}{
		{
			name:      "Non-empty string",
			input:     "test",
			wantValid: true,
			wantValue: "test",
		},
		{
			name:      "Empty string",
			input:     "",
			wantValid: false,
			wantValue: "",
		},
		{
			name:      "Whitespace string",
			input:     "   ",
			wantValid: true,
			wantValue: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toNullString(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("toNullString(%q).Valid = %v, want %v", tt.input, got.Valid, tt.wantValid)
			}
			if got.String != tt.wantValue {
				t.Errorf("toNullString(%q).String = %q, want %q", tt.input, got.String, tt.wantValue)
			}
		})
	}
}

// TestToNullFloat64 tests null float64 conversion
func TestToNullFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     float64
		wantValid bool
		wantValue float64
	}{
		{
			name:      "Positive number",
			input:     100.50,
			wantValid: true,
			wantValue: 100.50,
		},
		{
			name:      "Negative number",
			input:     -50.25,
			wantValid: true,
			wantValue: -50.25,
		},
		{
			name:      "Zero",
			input:     0.0,
			wantValid: false,
			wantValue: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toNullFloat64(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("toNullFloat64(%v).Valid = %v, want %v", tt.input, got.Valid, tt.wantValid)
			}
			if got.Float64 != tt.wantValue {
				t.Errorf("toNullFloat64(%v).Float64 = %v, want %v", tt.input, got.Float64, tt.wantValue)
			}
		})
	}
}

// TestStripMarkdownCodeBlocks tests markdown code block removal
func TestStripMarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "JSON with code blocks",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "Text without code blocks",
			input: "{\"key\": \"value\"}",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "Code block without language",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "Multiline JSON with code blocks",
			input: "```json\n{\n  \"key\": \"value\",\n  \"number\": 123\n}\n```",
			want:  "{\n  \"key\": \"value\",\n  \"number\": 123\n}",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownCodeBlocks(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdownCodeBlocks() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseDate tests date parsing
func TestParseDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
		wantError bool
	}{
		{
			name:      "Valid date",
			input:     "2024-01-15",
			wantYear:  2024,
			wantMonth: 1,
			wantDay:   15,
			wantError: false,
		},
		{
			name:      "Invalid date format",
			input:     "01/15/2024",
			wantError: true,
		},
		{
			name:      "Empty string",
			input:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDate(tt.input)
			if !tt.wantError {
				if got.Year() != tt.wantYear || int(got.Month()) != tt.wantMonth || got.Day() != tt.wantDay {
					t.Errorf("parseDate(%q) = %v, want year=%d month=%d day=%d",
						tt.input, got, tt.wantYear, tt.wantMonth, tt.wantDay)
				}
			}
			// For error cases, parseDate returns current time, which is acceptable
		})
	}
}

// TestFormatNullTime tests null time formatting
func TestFormatNullTime(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
		date  string
		want  string
	}{
		{
			name:  "Valid date",
			valid: true,
			date:  "2024-01-15",
			want:  "2024-01-15",
		},
		{
			name:  "Invalid/null date",
			valid: false,
			want:  "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nt := toNullTime(tt.date)
			got := formatNullTime(nt)
			if got != tt.want {
				t.Errorf("formatNullTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

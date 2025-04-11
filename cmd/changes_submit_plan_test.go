package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestLoadAutoTagRulesFile(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		errorContains string
		numberOfRules int
	}{
		{
			name:          "NonExistent.yaml",
			fileContent:   "",
			errorContains: "does not exist",
		},
		{
			name:          "FailedToParse.yaml",
			fileContent:   "invalid yaml content",
			errorContains: "Failed to parse",
		},
		{
			name:          "MoreThan10Rules.yaml",
			fileContent:   generateRules(11),
			errorContains: "10 rules",
		},
		{
			name:          "ValidRules.yaml",
			fileContent:   generateRules(5),
			errorContains: "there should be no error",
			numberOfRules: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.name
			if tt.fileContent != "" {
				err := os.WriteFile(filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Errorf("Failed to write file %q: %v", filePath, err)
				}
				defer os.Remove(filePath)
			}

			result, err := loadAutoTagRulesFile(filePath)
			if err != nil && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error\n%q\nto contain\n%q", err.Error(), tt.errorContains)
			}
			if tt.numberOfRules > 0 {

				if len(result) != tt.numberOfRules {
					t.Errorf("Expected %d rules, got %d", tt.numberOfRules, len(result))
				}
			}
		})
	}
}

func generateRules(count int) string {
	rules := "rules:"
	rules += `
  - name: rule0
    tag_key: key0
    enabled: true
    instructions: Instructions for rule 0
    valid_values:
      - value 0`
	for i := 1; i < count; i++ {
		rules += fmt.Sprintf(`
  - name: rule%d
    tag_key: key%d
    enabled: true
    instructions: Instructions for rule %d
    valid_values: ["value %d"]`, i, i, i, i)
	}
	return rules
}

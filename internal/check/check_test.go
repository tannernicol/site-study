package check

import (
	"path/filepath"
	"testing"

	"design-public/internal/rules"
)

func TestBundledRulesParseAndHaveCheckers(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedChecks int
	}{
		{
			name:           "Seattle residential",
			path:           filepath.Join("..", "..", "rules", "seattle-residential.yml"),
			expectedChecks: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, err := rules.Load(tt.path)
			if err != nil {
				t.Fatalf("load bundled rules: %v", err)
			}
			if len(rs.Checks) != tt.expectedChecks {
				t.Fatalf("bundled rules contain %d checks, want %d", len(rs.Checks), tt.expectedChecks)
			}
			for _, rule := range rs.Checks {
				fn, ok := checkers[shapeOf(rule)]
				if !ok || fn == nil {
					t.Errorf("rule %q has no check function (shape %q)", rule.ID, shapeOf(rule))
				}
			}
		})
	}
}

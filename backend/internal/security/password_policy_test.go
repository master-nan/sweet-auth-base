package security

import (
	"backend/model"
	"testing"
)

func TestValidatePasswordByConfigureAppliesPolicyFloor(t *testing.T) {
	cfg := model.SysConfigure{
		PasswordLength:     6,
		PasswordComplexity: 1,
		PasswordPolicy:     "strong",
	}

	if err := ValidatePasswordByConfigure("Abcdef123456", cfg); err != nil {
		t.Fatalf("expected strong policy password to pass: %v", err)
	}
	if err := ValidatePasswordByConfigure("abcdef12", cfg); err == nil {
		t.Fatalf("expected strong policy to enforce minimum length")
	}
	if err := ValidatePasswordByConfigure("abcdefghij", cfg); err == nil {
		t.Fatalf("expected strong policy to enforce complexity")
	}
}

func TestValidatePasswordByConfigureSupportsLowMediumHighCustomPolicies(t *testing.T) {
	tests := []struct {
		name      string
		cfg       model.SysConfigure
		accepted  string
		rejected  string
		rejectMsg string
	}{
		{
			name: "low only enforces length",
			cfg: model.SysConfigure{
				PasswordLength:     6,
				PasswordComplexity: 1,
				PasswordPolicy:     "low",
			},
			accepted:  "abcdef",
			rejected:  "abc",
			rejectMsg: "expected low policy to reject short passwords",
		},
		{
			name: "medium enforces letters and digits",
			cfg: model.SysConfigure{
				PasswordLength:     6,
				PasswordComplexity: 1,
				PasswordPolicy:     "medium",
			},
			accepted:  "abcdef12",
			rejected:  "abcdefgh",
			rejectMsg: "expected medium policy to require letters and digits",
		},
		{
			name: "high enforces three classes",
			cfg: model.SysConfigure{
				PasswordLength:     6,
				PasswordComplexity: 1,
				PasswordPolicy:     "high",
			},
			accepted:  "Abcdef123456",
			rejected:  "abcdef123456",
			rejectMsg: "expected high policy to require three character classes",
		},
		{
			name: "custom keeps explicit length and complexity",
			cfg: model.SysConfigure{
				PasswordLength:     10,
				PasswordComplexity: 2,
				PasswordPolicy:     "custom",
			},
			accepted:  "abcdef1234",
			rejected:  "abcdefghi",
			rejectMsg: "expected custom policy to use configured complexity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePasswordByConfigure(tt.accepted, tt.cfg); err != nil {
				t.Fatalf("expected password to pass: %v", err)
			}
			if err := ValidatePasswordByConfigure(tt.rejected, tt.cfg); err == nil {
				t.Fatal(tt.rejectMsg)
			}
		})
	}
}

func TestGeneratePasswordByConfigureAppliesPolicyFloor(t *testing.T) {
	cfg := model.SysConfigure{
		PasswordLength:     6,
		PasswordComplexity: 1,
		PasswordPolicy:     "high",
	}

	password, err := GeneratePasswordByConfigure(cfg)
	if err != nil {
		t.Fatalf("generate password: %v", err)
	}
	if len(password) < 12 {
		t.Fatalf("expected high policy password length >= 12, got %d", len(password))
	}
	if err := ValidatePasswordByConfigure(password, cfg); err != nil {
		t.Fatalf("generated password should validate: %v", err)
	}
}

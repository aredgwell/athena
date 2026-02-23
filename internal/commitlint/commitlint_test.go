package commitlint

import (
	"testing"
)

func TestCommitLintCommand(t *testing.T) {
	t.Run("valid feat", func(t *testing.T) {
		result := Lint("feat(auth): add login endpoint", LintOptions{
			ValidTypes: DefaultTypes(),
		})
		if !result.Valid {
			t.Errorf("expected valid: %v", result.Errors)
		}
		if result.Parsed.Type != "feat" {
			t.Errorf("type: got %s, want feat", result.Parsed.Type)
		}
		if result.Parsed.Scope != "auth" {
			t.Errorf("scope: got %s, want auth", result.Parsed.Scope)
		}
	})

	t.Run("valid without scope", func(t *testing.T) {
		result := Lint("fix: correct null check", LintOptions{
			ValidTypes: DefaultTypes(),
		})
		if !result.Valid {
			t.Errorf("expected valid: %v", result.Errors)
		}
		if result.Parsed.Type != "fix" {
			t.Errorf("type: got %s, want fix", result.Parsed.Type)
		}
	})

	t.Run("breaking change bang", func(t *testing.T) {
		result := Lint("feat(api)!: remove deprecated endpoints", LintOptions{
			ValidTypes: DefaultTypes(),
		})
		if !result.Valid {
			t.Errorf("expected valid: %v", result.Errors)
		}
		if !result.Parsed.Breaking {
			t.Error("expected breaking=true")
		}
	})

	t.Run("breaking change footer", func(t *testing.T) {
		msg := "feat(api): update auth flow\n\nNew auth mechanism.\n\nBREAKING CHANGE: old tokens invalidated"
		parsed, err := Parse(msg)
		if err != nil {
			t.Fatal(err)
		}
		if !parsed.Breaking {
			t.Error("expected breaking=true from footer")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		result := Lint("yolo: do stuff", LintOptions{
			ValidTypes: DefaultTypes(),
		})
		if result.Valid {
			t.Error("expected invalid for unknown type")
		}
	})

	t.Run("not conventional", func(t *testing.T) {
		result := Lint("Update the readme", LintOptions{})
		if result.Valid {
			t.Error("expected invalid for non-conventional message")
		}
	})

	t.Run("require scope", func(t *testing.T) {
		result := Lint("feat: no scope", LintOptions{
			ValidTypes:  DefaultTypes(),
			RequireScope: true,
		})
		if result.Valid {
			t.Error("expected invalid when scope required but missing")
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		result := Lint("feat(unknown): something", LintOptions{
			ValidTypes:  DefaultTypes(),
			ValidScopes: []string{"auth", "api"},
		})
		if result.Valid {
			t.Error("expected invalid for unknown scope")
		}
	})

	t.Run("valid scope", func(t *testing.T) {
		result := Lint("feat(api): add endpoint", LintOptions{
			ValidTypes:  DefaultTypes(),
			ValidScopes: []string{"auth", "api"},
		})
		if !result.Valid {
			t.Errorf("expected valid: %v", result.Errors)
		}
	})
}

func TestParse(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		parsed, err := Parse("fix(db): handle null\n\nAdded null check for edge case.")
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Body != "Added null check for edge case." {
			t.Errorf("body: got %q", parsed.Body)
		}
	})

	t.Run("with body and footer", func(t *testing.T) {
		parsed, err := Parse("feat(auth): add SSO\n\nSSO integration.\n\nCloses #123")
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Body != "SSO integration." {
			t.Errorf("body: got %q", parsed.Body)
		}
		if parsed.Footer != "Closes #123" {
			t.Errorf("footer: got %q", parsed.Footer)
		}
	})
}

func TestLintAll(t *testing.T) {
	messages := []string{
		"feat(auth): add login",
		"not conventional",
		"fix: bug fix",
	}

	summary := LintAll(messages, LintOptions{ValidTypes: DefaultTypes()})

	if summary.Total != 3 {
		t.Errorf("total: got %d, want 3", summary.Total)
	}
	if summary.Valid != 2 {
		t.Errorf("valid: got %d, want 2", summary.Valid)
	}
	if summary.Invalid != 1 {
		t.Errorf("invalid: got %d, want 1", summary.Invalid)
	}
}

func TestDefaultTypes(t *testing.T) {
	types := DefaultTypes()
	if len(types) != 11 {
		t.Errorf("types: got %d, want 11", len(types))
	}
	expected := map[string]bool{
		"feat": false, "fix": false, "docs": false, "chore": false, "perf": false,
	}
	for _, typ := range types {
		if _, ok := expected[typ]; ok {
			expected[typ] = true
		}
	}
	for typ, found := range expected {
		if !found {
			t.Errorf("missing expected type: %s", typ)
		}
	}
}

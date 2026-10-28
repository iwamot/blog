package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutospace(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"empty", "", ""},
		{"latin", "Hello 2026", "Hello 2026"},
		{"japanese", "日本語ひらがなカタカナ", "日本語ひらがなカタカナ"},
		{"boundaries", "GoでOGPを生成2026年", "Go で OGP を生成 2026 年"},
		{"existing spaces", "Go で OGP を生成", "Go で OGP を生成"},
		{"punctuation", "Go、日本語。2026", "Go、日本語。2026"},
		{"accented latin", "日本語café", "日本語 café"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := autospace(tt.input); got != tt.want {
				t.Errorf("autospace(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if got := autospace(tt.want); got != tt.want {
				t.Errorf("autospace is not idempotent: %q", got)
			}
		})
	}
}

func TestAutospaceTitle(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"plain", "---\ntitle: Goで生成\n---\nGoで本文\n", "---\ntitle: Go で生成\n---\nGoで本文\n"},
		{"double quoted", "title: \"Goで生成\"\n", "title: \"Go で生成\"\n"},
		{"single quoted", "title: 'Goで生成'\n", "title: 'Go で生成'\n"},
		{"only first title", "title: Goで生成\ntitle: Goで本文\n", "title: Go で生成\ntitle: Goで本文\n"},
		{"no title", "---\ntags: [Go]\n---\n", "---\ntags: [Go]\n---\n"},
		{"empty title", "title: \n", "title: \n"},
		{"one quote", "title: '\n", "title: '\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := autospaceTitle(tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFrontMatter(t *testing.T) {
	for _, tt := range []struct {
		name, input, head, body string
		ok                      bool
	}{
		{"valid", "---\ntitle: Hello\n---\nBody\n", "title: Hello\n", "---\nBody\n", true},
		{"no opening fence", "title: Hello\n", "", "title: Hello\n", false},
		{"no closing fence", "---\ntitle: Hello\n", "", "---\ntitle: Hello\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			head, body, ok := frontMatter(tt.input)
			if head != tt.head || body != tt.body || ok != tt.ok {
				t.Errorf("got (%q, %q, %v), want (%q, %q, %v)", head, body, ok, tt.head, tt.body, tt.ok)
			}
		})
	}
}

func TestHasTags(t *testing.T) {
	for _, tt := range []struct {
		name, head string
		want       bool
	}{
		{"missing", "title: Hello\n", false},
		{"inline", "tags: [Go, Hugo]\n", true},
		{"inline empty", "tags: []\n", false},
		{"inline spaces", "tags: [  ]\n", false},
		{"block", "tags:\n  - Go\n", true},
		{"block with blank lines", "tags:\n\n  - Go\n", true},
		{"block empty item", "tags:\n  - \n", false},
		{"block non-item", "tags:\n  value\n", false},
		{"following field", "tags:\ntitle: Hello\n  - not a tag\n", false},
		{"empty", "tags:\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasTags(tt.head); got != tt.want {
				t.Errorf("hasTags(%q) = %v, want %v", tt.head, got, tt.want)
			}
		})
	}
}

func TestInsertField(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"insert", "---\ntitle: Hello\n---\nBody\n", "---\ntitle: Hello\ntags: [Go]\n---\nBody\n"},
		{"preserve existing", "---\ntags: [Hugo]\n---\n", "---\ntags: [Hugo]\n---\n"},
		{"no front matter", "Body\n", "Body\n"},
		{"unclosed front matter", "---\ntitle: Hello\n", "---\ntitle: Hello\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := insertField(tt.input, "tags", "tags: [Go]"); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureFields(t *testing.T) {
	input := "---\ntitle: Hello\n---\nBody\n"
	want := "---\ntitle: Hello\nauthor: [\"@iwamot\"]\ncategories: [\"Articles\"]\n---\nBody\n"
	if got := ensureFields(input); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	existing := "---\nauthor: [Alice]\ncategories: [Notes]\n---\nBody\n"
	if got := ensureFields(existing); got != existing {
		t.Errorf("existing fields changed: %q", got)
	}
}

func TestWithPlaceholderTags(t *testing.T) {
	for _, input := range []string{"---\ntitle: Hello\n---\nBody\n", "---\ntitle: Hello\ntags: []\n---\nBody\n"} {
		want := "---\ntitle: Hello\ntags: [\"x\"]\n---\nBody\n"
		if got := withPlaceholderTags(input); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestWithTagsDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	input := "info:\n  enabled: false\ntags:\n  enabled: true  \n  limit: 0\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := withTagsDisabled(path)
	want := "info:\n  enabled: false\ntags:\n  enabled: false\n  limit: 0\n"
	if err != nil || got != want {
		t.Fatalf("got (%q, %v), want (%q, nil)", got, err, want)
	}
	if err := os.WriteFile(path, []byte("tags:\n  enabled: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := withTagsDisabled(path); err == nil || !strings.Contains(err.Error(), "no enabled-true line") {
		t.Errorf("expected missing enabled-true diagnostic, got %v", err)
	}
	if _, err := withTagsDisabled(filepath.Join(dir, "missing")); err == nil {
		t.Error("expected read error")
	}
}

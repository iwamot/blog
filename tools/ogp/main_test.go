package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// Each token is one kind of character: Latin letters or digits, CJK, or
// something else. A space belongs between a Latin token and a CJK token, in
// either order, and nowhere else.
var autospaceTokens = []struct{ text, kind string }{
	{"Go", "latin"}, {"OGP", "latin"}, {"2026", "latin"}, {"Hello", "latin"}, {"café", "latin"},
	{"日本語", "cjk"}, {"ひらがな", "cjk"}, {"カタカナ", "cjk"}, {"で", "cjk"}, {"を", "cjk"}, {"生成", "cjk"}, {"年", "cjk"},
	{" ", "other"}, {"、", "other"}, {"。", "other"}, {"-", "other"}, {"!", "other"},
}

func TestAutospaceSpacesLatinFromCJKOnly(t *testing.T) {
	var walk func(input, want, last string, depth int)
	walk = func(input, want, last string, depth int) {
		if got := autospace(input); got != want {
			t.Errorf("autospace(%q) = %q, want %q", input, got, want)
		}
		if got := autospace(want); got != want {
			t.Errorf("autospace is not idempotent: %q", got)
		}
		if depth == 3 {
			return
		}
		for _, tok := range autospaceTokens {
			sep := ""
			if (last == "latin" && tok.kind == "cjk") || (last == "cjk" && tok.kind == "latin") {
				sep = " "
			}
			walk(input+tok.text, want+sep+tok.text, tok.kind, depth+1)
		}
	}
	walk("", "", "", 0)
}

// tagForms are the ways front matter writes tags, with whether each names one.
var tagForms = []struct {
	text string
	has  bool
}{
	{"", false},
	{"tags: [Go, Hugo]\n", true},
	{"tags: []\n", false},
	{"tags: [  ]\n", false},
	{"tags:\n  - Go\n", true},
	{"tags:\n\n  - Go\n", true},
	{"tags:\n  - \n", false},
	{"tags:\n  value\n", false},
	{"tags:\ntitle: Hello\n  - not a tag\n", false},
	{"tags:\n", false},
}

// tagContexts are the fields that may sit around tags in the same front matter.
var tagContexts = [][2]string{{"", ""}, {"title: T\n", ""}, {"", "draft: true\n"}, {"title: T\n", "draft: true\n"}}

func TestHasTagsIgnoresTheFieldsAround(t *testing.T) {
	for _, form := range tagForms {
		for _, around := range tagContexts {
			head := around[0] + form.text + around[1]
			if got := hasTags(head); got != form.has {
				t.Errorf("hasTags(%q) = %v, want %v", head, got, form.has)
			}
		}
	}
}

func TestWithPlaceholderTagsGivesOneTagAndKeepsTheRest(t *testing.T) {
	bodies := []string{"Body\n", "```yaml\ntags: []\n```\n", "---\ntags: []\n---\n", ""}
	for _, form := range tagForms {
		for _, around := range tagContexts {
			if around[0]+form.text+around[1] == "" {
				// Every article's front matter holds at least its title.
				continue
			}
			for _, body := range bodies {
				input := "---\n" + around[0] + form.text + around[1] + "---\n" + body
				out := withPlaceholderTags(input)
				head, rest, ok := frontMatter(out)
				if !ok || !hasTags(head) || strings.Count(head, "tags:") != 1 {
					t.Errorf("withPlaceholderTags(%q) = %q: want one tag in the front matter", input, out)
					continue
				}
				if rest != "---\n"+body {
					t.Errorf("withPlaceholderTags(%q) changed the body: %q", input, rest)
				}
				for _, field := range around {
					if !strings.Contains(head, field) {
						t.Errorf("withPlaceholderTags(%q) dropped %q", input, field)
					}
				}
			}
		}
	}
	if got := withPlaceholderTags("Body\n"); got != "Body\n" {
		t.Errorf("content without front matter changed: %q", got)
	}
}

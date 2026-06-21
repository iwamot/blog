// Command ogp generates OGP card images (ogp.png) for Hugo article bundles
// using tcardgen. Pass one or more article directories or index.md paths.
//
// The site relies on CSS `text-autospace` to space Japanese against Latin and
// numeric text, so article titles are written without manual spaces. tcardgen
// renders titles literally, so this command inserts the same spacing into the
// title before generation.
//
// tcardgen requires `author`, `categories`, and `tags` front matter. `author`
// and `categories` are never shown, so they are injected into a temporary copy.
// `tags` is shown when present; an article without tags is rendered with the
// tag row disabled (a placeholder tag is injected only to satisfy tcardgen).
//
// Cards are generated from temporary copies and written next to each article as
// ogp.png. The article files are not modified — `images: ["ogp.png"]` is
// expected to be in the front matter already.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	fontDir    = "assets/tcardgen/font"
	configFile = "assets/tcardgen/config.yaml"
	outputName = "ogp.png"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ogp <article-dir|index.md>...")
		os.Exit(2)
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ogp:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	notagsConfig, err := withTagsDisabled(configFile)
	if err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "ogp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	notagsPath := filepath.Join(tmp, "config-notags.yaml")
	if err := os.WriteFile(notagsPath, []byte(notagsConfig), 0o644); err != nil {
		return err
	}
	outDir := filepath.Join(tmp, "out")
	if err := os.Mkdir(outDir, 0o755); err != nil {
		return err
	}

	type job struct {
		bundle string
		card   string
	}
	var jobs []job
	var tagged, untagged []string

	for _, arg := range args {
		path, ok, err := resolve(arg)
		if err != nil {
			return err
		}
		if !ok {
			continue // path does not exist (e.g. an unmatched glob)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		head, _, _ := frontMatter(string(src))
		withTags := hasTags(head)

		content := ensureFields(autospaceTitle(string(src)))
		if !withTags {
			content = withPlaceholderTags(content)
		}

		key := strconv.Itoa(len(jobs))
		md := filepath.Join(tmp, key+".md")
		if err := os.WriteFile(md, []byte(content), 0o644); err != nil {
			return err
		}
		jobs = append(jobs, job{bundle: filepath.Dir(path), card: filepath.Join(outDir, key+".png")})
		if withTags {
			tagged = append(tagged, md)
		} else {
			untagged = append(untagged, md)
		}
	}
	if len(jobs) == 0 {
		return nil
	}

	// One tcardgen process per group loads the fonts only once.
	generate(configFile, outDir, tagged)
	generate(notagsPath, outDir, untagged)

	var missing []string
	for _, j := range jobs {
		if _, err := os.Stat(j.card); err != nil {
			missing = append(missing, j.bundle)
			continue
		}
		if err := copyFile(j.card, filepath.Join(j.bundle, outputName)); err != nil {
			return err
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("no card generated for: %s", strings.Join(missing, ", "))
	}
	return nil
}

// resolve turns an argument into an index.md path. A directory is joined with
// index.md; a path that does not exist is reported as skippable.
func resolve(arg string) (string, bool, error) {
	fi, err := os.Stat(arg)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if fi.IsDir() {
		return filepath.Join(arg, "index.md"), true, nil
	}
	return arg, true, nil
}

// generate renders every md in one tcardgen process, writing <basename>.png to
// outDir. Per-file failures surface as a missing output PNG, handled by run.
func generate(config, outDir string, mds []string) {
	if len(mds) == 0 {
		return
	}
	args := append([]string{"-f", fontDir, "-c", config, "-o", outDir + "/"}, mds...)
	cmd := exec.Command("tcardgen", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

var enabledTrue = regexp.MustCompile(`(?m)^(\s*)enabled: true[ \t]*$`)

// withTagsDisabled returns the drawing config with the tag row turned off. Only
// the tags block enables itself, so flipping every enabled-true line is safe.
func withTagsDisabled(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !enabledTrue.Match(b) {
		return "", fmt.Errorf("%s: no enabled-true line found to derive the no-tags config", path)
	}
	return enabledTrue.ReplaceAllString(string(b), "${1}enabled: false"), nil
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// frontMatter returns the YAML block between the leading --- fences.
func frontMatter(content string) (head, body string, ok bool) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content, false
	}
	rel := strings.Index(content[4:], "\n---")
	if rel < 0 {
		return "", content, false
	}
	return content[4 : 4+rel+1], content[4+rel+1:], true
}

var tagsLine = regexp.MustCompile(`(?m)^tags:[ \t]*(.*)$`)

// hasTags reports whether the front matter declares at least one tag, in either
// inline (`tags: ["a"]`) or block (`tags:\n  - a`) form.
func hasTags(head string) bool {
	loc := tagsLine.FindStringSubmatchIndex(head)
	if loc == nil {
		return false
	}
	if inline := strings.TrimSpace(head[loc[2]:loc[3]]); inline != "" {
		return strings.Trim(inline, "[] \t") != ""
	}
	for _, line := range strings.Split(head[loc[1]:], "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break // dedented: the next key
		}
		if item := strings.TrimSpace(line); strings.HasPrefix(item, "- ") && strings.TrimSpace(item[2:]) != "" {
			return true
		}
	}
	return false
}

// withPlaceholderTags ensures a non-empty tags field so tcardgen accepts the
// file. The card itself is rendered with the tag row disabled.
func withPlaceholderTags(content string) string {
	if strings.Contains(content, "tags: []") {
		return strings.Replace(content, "tags: []", `tags: ["x"]`, 1)
	}
	return insertField(content, "tags", `tags: ["x"]`)
}

var titleLine = regexp.MustCompile(`(?m)^(title:[ \t]*)(.*)$`)

// autospaceTitle rewrites the first front matter title line, inserting spacing
// into its value while preserving any surrounding quotes.
func autospaceTitle(content string) string {
	done := false
	return titleLine.ReplaceAllStringFunc(content, func(line string) string {
		if done {
			return line
		}
		done = true
		m := titleLine.FindStringSubmatch(line)
		prefix, value := m[1], m[2]
		quote := ""
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			quote = string(value[0])
			value = value[1 : len(value)-1]
		}
		return prefix + quote + autospace(value) + quote
	})
}

// injected lists front matter that tcardgen requires but the site and card do
// not display. Each is added to the temporary copy when the article omits it.
var injected = []struct{ key, line string }{
	{"author", `author: ["@iwamot"]`},
	{"categories", `categories: ["Articles"]`},
}

// ensureFields adds any missing injected fields to the front matter.
func ensureFields(content string) string {
	for _, f := range injected {
		content = insertField(content, f.key, f.line)
	}
	return content
}

// insertField adds line to the YAML front matter block when key is absent.
func insertField(content, key, line string) string {
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	rel := strings.Index(content[4:], "\n---")
	if rel < 0 {
		return content
	}
	head, body := content[:4+rel+1], content[4+rel+1:]
	if regexp.MustCompile(`(?m)^` + key + `:`).MatchString(head) {
		return content
	}
	return head + line + "\n" + body
}

// autospace inserts a single space at every boundary between a Latin/numeric
// rune and a CJK rune. An existing space breaks the boundary, so titles that
// already contain spaces are left unchanged.
func autospace(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if i > 0 && boundary(rs[i-1], r) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func boundary(a, b rune) bool {
	return (isLatin(a) && isCJK(b)) || (isCJK(a) && isLatin(b))
}

func isLatin(r rune) bool {
	return unicode.In(r, unicode.Latin) || (r >= '0' && r <= '9')
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
}

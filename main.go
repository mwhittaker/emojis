package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	// emojiRegex is a regex that matches a non-empty non-comment line from
	// emoji-test.txt.
	emojiRegex = regexp.MustCompile(`([0-9A-F ]*?);\s*(component|fully-qualified|minimally-qualified|unqualified)\s*# (.*) E[0-9]+.[0-9]+ (.*)`)

	// tokenRegex is a regex used to tokenize words like "animal-mammal" into
	// "animal" and "mammal".
	tokenRegex = regexp.MustCompile(`[^a-zA-Z]`)
)

// emoji represents an emoji or emoji sequence. Note that not every emoji is a
// single code point. For example, the black cat emoji is actually three code
// points: the cat code point, the zero width joiner code point, and the black
// square codepoint.
type emoji struct {
	Grapheme string   // the emoji or emoji sequence (e.g., 😀)
	Codes    []rune   // the code points in grapheme (e.g., [0x1F600])
	Name     string   // the name of the emoji (e.g., "grinning face")
	Group    string   // the emoji's group (e.g., "Smileys & Emotion")
	Subgroup string   // the emoji's subgroup (e.g., "face-smiling")
	Tags     []string // tags describing the emoji (e.g., "happy", "content")
}

// parse parses emojis from an emoji-test.txt file.
func parse(r io.Reader) ([]*emoji, error) {
	group := ""
	subgroup := ""

	var emojis []*emoji
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			// The line is empty.
			continue
		}
		if strings.HasPrefix(line, "# group: ") {
			// The line begins a group.
			group, _ = strings.CutPrefix(line, "# group: ")
			continue
		}
		if strings.HasPrefix(line, "# subgroup: ") {
			// The line begins a subgroup.
			subgroup, _ = strings.CutPrefix(line, "# subgroup: ")
			continue
		}
		if strings.HasPrefix(line, "#") {
			// The line is an uninteresting comment.
			continue
		}
		matches := emojiRegex.FindStringSubmatch(line)
		if matches == nil {
			// The line does not list an emoji.
			continue
		}

		// The line lists an emoji.
		codes := strings.Fields(matches[1])
		qualification := strings.TrimSpace(matches[2])
		grapheme := strings.TrimSpace(matches[3])
		name := strings.TrimSpace(matches[4])

		if qualification != "fully-qualified" {
			// Ignore component, minimally qualified, and unqualified emojis.
			// See https://unicode.org/reports/tr51/ for details.
			continue
		}

		// Double check that the grapheme's runes match the expected runes.
		// Some emoji data sources list incorrect graphemes.
		runes, err := parseCodes(codes)
		if err != nil {
			return nil, err
		}
		if !slices.Equal(runes, []rune(grapheme)) {
			return nil, fmt.Errorf("mismatched runes: got %v, want %v", runes, []rune(grapheme))
		}

		emoji := &emoji{
			Grapheme: grapheme,
			Codes:    runes,
			Name:     name,
			Group:    group,
			Subgroup: subgroup,
		}
		emojis = append(emojis, emoji)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return emojis, nil
}

// parseCodes parses a slice of unicode code points in hex (e.g., ["2639",
// "FE0F"]) into the corresponding runes (e.g., [0x2639, 0xFE0F]).
func parseCodes(codes []string) ([]rune, error) {
	var runes []rune
	for _, code := range codes {
		x, err := strconv.ParseInt(code, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("strconv.ParseInt(%s): %w", code, err)
		}
		runes = append(runes, rune(x))
	}
	return runes, nil
}

// parseTags parses tags from a data.json file.
func parseTags(r io.Reader) (map[string][]string, error) {
	type entry struct {
		Emoji string
		Tags  []string
		Skins []entry
	}

	decoder := json.NewDecoder(r)
	var entries []entry
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	tags := map[string][]string{}
	for _, entry := range entries {
		tags[entry.Emoji] = entry.Tags
		for _, skin := range entry.Skins {
			tags[skin.Emoji] = append(entry.Tags, skin.Tags...)
		}
	}
	return tags, nil
}

// tokenize tokenizes a set of strings. For example, calling tokenize on the
// strings ["Foo bar", "moo-cow"] will return ["bar", "cow" "foo", "moo"].
func tokenize(ss []string) []string {
	tokens := map[string]bool{}
	for _, s := range ss {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, ".", "")
		s = tokenRegex.ReplaceAllLiteralString(s, " ")
		for _, token := range strings.Fields(s) {
			tokens[strings.ToLower(token)] = true
		}
	}
	sorted := maps.Keys(tokens)
	sort.Strings(sorted)
	return sorted
}

func main() {
	// Parse emojis.
	in, err := os.Open("emoji-test.txt")
	if err != nil {
		panic(err)
	}
	emojis, err := parse(in)
	if err != nil {
		panic(err)
	}

	// Parse tags.
	data, err := os.Open("data.json")
	if err != nil {
		panic(err)
	}
	tags, err := parseTags(data)
	if err != nil {
		panic(err)
	}
	for _, emoji := range emojis {
		emoji.Tags = tags[emoji.Grapheme]
	}

	// Output the emojis as json.
	bytes, err := json.MarshalIndent(emojis, "", "    ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("emojis.json", bytes, 0644); err != nil {
		panic(err)
	}

	// Output tokens as go map.
	var b strings.Builder
	fmt.Fprintln(&b, "package main")
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "// Taken from https://github.com/mwhittaker/emojis.")
	fmt.Fprintln(&b, "var emojis = map[string][]string {")
	for _, emoji := range emojis {
		inputs := append(emoji.Tags, emoji.Name, emoji.Group, emoji.Subgroup)
		tokens := tokenize(inputs)
		formatted := make([]string, len(tokens))
		for i, token := range tokens {
			formatted[i] = fmt.Sprintf("%q", token)
		}
		fmt.Fprintf(&b, "\t%q: {%s},\n", emoji.Grapheme, strings.Join(formatted, ", "))
	}
	fmt.Fprintln(&b, "}")
	if err := os.WriteFile("emojis.go", []byte(b.String()), 0644); err != nil {
		panic(err)
	}
}

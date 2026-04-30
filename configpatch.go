package configpatch

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"go.uber.org/zap"
	"os"
	"regexp"
	"strings"
)

type Interval struct {
	From int
	To   int
}

// Apply hunks to files reversibly
type Patcher struct {
	// Maintained file
	Path         string
	// String representation
	Data         string
	// Comment format, defaults to '#' as Start
	CommentStart string
	CommentEnd   string
	// Turn on verbose screen debugging
	Debug  bool
	Logger *zap.Logger
	// Positions with existing patches
	ForbiddenZones []Interval
	// How to detect an existing marker
	MarkerRx     *regexp.Regexp
}

// Create a new Patcher object
func NewPatcher() *Patcher {
	patcher := Patcher{
		CommentStart: "#",
		Debug:        false,
		Logger:       zap.Must(zap.NewProduction()),
	}
	return &patcher
}

// Point the Patcher to a file
func (p *Patcher) Init(path string) error {
	p.Path = path
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(data)
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	p.Data = s

	if p.Debug {
		p.Logger = zap.NewExample()
	}

	p.MarkerRx = regexp.MustCompile(fmt.Sprintf("(?m)^%s\\(%s-(.*)-(.*?)\\)",
		regexp.QuoteMeta(p.CommentStart),
		"Config::Patch",
	))

	// Mark forbidden zones
	p.ForbiddenZones = []Interval{}
	p.Traverse(
		func(p *Patcher, h *Hunk) {
			p.ForbiddenZones = append(p.ForbiddenZones,
				Interval{From: h.PosFrom, To: h.PosTo})
		},
		func(p *Patcher, s string) {},
	)

	return nil
}

// Traverse the file and run callbacks on hunks and regular text sections
func (p *Patcher) Traverse(
	patchCB func(*Patcher, *Hunk),
	textCB func(*Patcher, string),
) bool {
	inPatch := false
	patchText := ""
	text := ""
	var startPos, endPos int
	pos := 0
	header := ""

	scanner := bufio.NewScanner(strings.NewReader(p.Data))
	for scanner.Scan() {
		line := scanner.Text()
		line += "\n"
		pos += len(line)

		match := p.MarkerRx.FindStringSubmatch(line)
		matched := match != nil

		if inPatch && !matched {
			patchText += line
		}

		// text line
		if !matched && !inPatch {
			text += line
		}

		// closing line of patch
		if matched && inPatch {
			endPos = pos - 1

			fullMatch := match[0]

			h := &Hunk{
				CommentStart:   p.CommentStart,
				CommentEnd:     p.CommentEnd,
				Key:            match[1],
				Mode:           match[2],
				Text:           patchText,
				PosFrom:        startPos,
				PosTo:          endPos,
				Header:         header,
				ContentPosFrom: startPos + len(fullMatch) + 1,
				ContentPosTo:   endPos - len(fullMatch),
				AsString:       p.Data[startPos : endPos+1],
				Logger:         p.Logger,
			}

			patchCB(p, h)
			patchText = ""
		}

		// toggle flag
		if matched {
			if !inPatch {
				// start line
				textCB(p, text)
				startPos = pos - len(line)
				header = line
			}
			text = ""
			inPatch = !inPatch
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if len(text) > 0 {
		textCB(p, text)
	}

	return true
}

// Write the file back to the original location
func (p *Patcher) Save() error {
	return p.SaveAs(p.Path)
}

// Write the file to disk under a new name
func (p *Patcher) SaveAs(path string) error {
	return os.WriteFile(path, []byte(p.Data), 0644)
}

// Apply a hunk in the mode specified
func (p *Patcher) Apply(h *Hunk) error {
	h.CommentStart = p.CommentStart
	h.CommentEnd = p.CommentEnd

	p.Logger.Debug("Apply hunk to",
		zap.String("path", p.Path), zap.String("mode", h.Mode))

	switch h.Mode {

	case "append":
		p.Data += h.StringGenerate()

	case "prepend":
		p.Data = h.StringGenerate() + p.Data

	case "replace":
		return p.patchByWedge(h, h.Text, "replace", false)

	case "insert-before":
		return p.patchByWedge(h, h.Text, "insert", false)

	case "insert-after":
		return p.patchByWedge(h, h.Text, "insert", true)

	// TODO: update, comment_out

	default:
		return fmt.Errorf("unknown mode: %s", h.Mode)
	}

	return nil
}

// Back out the hunk applied earlier under the given key
func (p *Patcher) Eject(key string) int {
	processed := ""
	ejected := 0

	p.Traverse(
		func(p *Patcher, h *Hunk) {
			p.Logger.Debug("Traverse, found hunk",
				zap.String("Key", h.Key),
				zap.String("Mode", h.Mode),
			)
			if h.Key != key {
				// Not our patch, keep
				processed += h.AsString
				return
			}
			if h.Mode == "replace" {
				hidden, _, err := h.ReplaceStringExtract()
				if err != nil {
					panic(err) // garbled base64
				}
				processed += hidden
				ejected += 1
			} else {
				// Replace by nothing
				ejected += 1
			}
		},
		func(p *Patcher, s string) {
			processed += s
		},
	)

	p.Data = processed
	return ejected
}

func (p *Patcher) Patched(key string) bool {
	found := false

	p.Traverse(
		func(p *Patcher, h *Hunk) {
			if key == h.Key {
				found = true
			}
		},
		func(p *Patcher, s string) {},
	)

	return found
}

func (p *Patcher) patchByWedge(h *Hunk, replace string, mode string, after bool) error {
	p.Logger.Debug("patchByWedge",
		zap.String("text", h.Text), zap.String("replace", replace),
		zap.String("mode", mode), zap.Bool("after", after))

	if p.Patched(h.Key) {
		return fmt.Errorf("Already patched with key %s", h.Key)
	}

	if h.Regex == nil {
		return fmt.Errorf("No regexp set in hunk")
	}

	// We can only handle replace strings with trailing newlines
	// TODO: why?
	if len(replace) > 0 && replace[len(replace)-1] != '\n' {
		replace += "\n"
	}

	data := p.Data
	positions := p.FullLineMatch(data, h.Regex)
	pieces := []string{}
	rest := data
	offset := 0

	p.Logger.Debug("patchByWedge", zap.String("pos", fmt.Sprintf("%v", positions)))

	for _, pos := range positions {
		var before, trail, hide string

		if mode == "insert" {
			if after {
				before = data[offset : pos.To+1]
				rest = data[pos.To+1:]
				hide = ""
				trail = ""
			} else {
				before = data[offset:pos.From]
				rest = data[pos.To+1:]
				hide = ""
				trail = data[pos.From : pos.To+1]
			}
		} else if mode == "replace" {
			before = data[offset:pos.From]
			rest = data[pos.To+1:]

			hide = h.ReplaceStringHide(data[pos.From : pos.To+1])
			trail = ""
		}

		p.Logger.Debug("patchByWedge",
			zap.String("before", before), zap.String("trail", trail),
			zap.String("hide", hide))

		h.Text = replace + hide

		pieces = append(pieces,
			before,
			h.StringGenerate(),
			trail,
		)

		offset = pos.To + 1
	}

	pieces = append(pieces, rest)

	p.Data = strings.Join(pieces, "")

	return nil
}

func (p *Patcher) FullLineMatch(s string, re *regexp.Regexp) []Interval {
	p.Logger.Debug("FullLineMatch of",
		zap.String("text", s), zap.String("re", re.String()))
	intervals := []Interval{}

	matches := re.FindAllStringIndex(s, -1)

	for _, m := range matches {
		first := m[0]
		last := m[1] - 1

		p.Logger.Debug("Found match at",
			zap.Int("from", first),
			zap.Int("to", last),
			zap.String("match", s[m[0]:m[1]]))

		if intersectsAny(p.ForbiddenZones, first, last) {
			p.Logger.Debug("Match was in forbidden zone - skipped")
			continue
		}

		// Move back to start of line
		for first > 0 && s[first-1] != '\n' {
			first--
		}

		// Move forward to end of line
		for last < len(s) && s[last] != '\n' {
			last++
		}

		p.Logger.Debug("Match positions corrected to",
			zap.Int("from", first),
			zap.Int("to", last))

		// Ignore overlapping matches
		if len(intervals) > 0 && intervals[len(intervals)-1].To > first {
			p.Logger.Debug("Overlap (two matches in same line) - skipped")
			continue
		}

		intervals = append(intervals, Interval{From: first, To: last})
	}

	return intervals
}

// Check if [a,b] intersects any forbidden interval
func intersectsAny(zones []Interval, a, b int) bool {
	for _, z := range zones {
		if !(b < z.From || a > z.To) {
			return true
		}
	}
	return false
}

// A hunk of patch data
type Hunk struct {
	// Key of this patch hunk, only one key per patch
	Key            string
	// How to apply (append, replace, etc.)
	Mode           string
	// Text to apply
	Text           string
	// Regex to find where to apply the hunk (replace mode)
	Regex          *regexp.Regexp
	// Once applied, here's text index positions
	PosFrom        int
	PosTo          int
	Header         string
	ContentPosFrom int
	ContentPosTo   int
	AsString       string
	Logger         *zap.Logger
	// Comment out configpatch's markers (set by Patcher)
	CommentStart string
	CommentEnd   string
}

// Create a new hunk. You needs to set Key and Mode fields afterwards before applying it.
func NewHunk() *Hunk {
	h := Hunk{
		CommentStart: "#",
		Logger:       zap.Must(zap.NewProduction()),
	}

	return &h
}

func (h *Hunk) PatchMarker() string {
	mode, _, _ := strings.Cut(h.Mode, "-")
	return fmt.Sprintf("%s(Config::Patch-%s-%s)%s\n",
		h.CommentStart, h.Key, mode, h.CommentEnd)
}

func (h *Hunk) ReplaceMarker() string {
	return fmt.Sprintf("%s(Config::Patch::replace)%s\n",
		h.CommentStart, h.CommentEnd)
}

func (h *Hunk) StringGenerate() string {
	if !strings.HasSuffix(h.Text, "\n") {
		h.Text += "\n"
	}
	m := h.PatchMarker()
	return m + h.Text + m
}

func (h *Hunk) Freeze(s string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	re := regexp.MustCompile(`(?m)^`)
	re.ReplaceAllString(enc, h.CommentStart+" ")
	return enc
}

func (h *Hunk) Thaw(enc string) (string, error) {
	re := regexp.MustCompile(`(?m)^` + h.CommentStart + " ")
	enc = re.ReplaceAllString(enc, "")

	dec, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", fmt.Errorf("%v (enc=%s)", err, enc)
	}
	return string(dec), nil
}

// Find the replacement string in a patch. Return the thawed replacement string and
// the hunk text with the replacement section removed.
func (h *Hunk) ReplaceStringExtract() (string, string, error) {
	replaceMarker := regexp.QuoteMeta(h.ReplaceMarker())
	re := regexp.MustCompile(`(?ms)^` + replaceMarker + `(.*?)^` + replaceMarker)

	matches := re.FindStringSubmatch(h.Text)
	if matches == nil {
		return "", "", nil
	}
	repl := matches[1]

	cleaned := re.ReplaceAllString(h.Text, "")

	// Remove comments from base64 encoded replacement
	reUncomment := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*|%s$`,
		regexp.QuoteMeta(h.CommentStart),
		regexp.QuoteMeta(h.CommentEnd)))
	repl = reUncomment.ReplaceAllString(repl, "")

	h.Logger.Debug("Replace extract", zap.String("src", repl))

	thawed, err := h.Thaw(repl)
	return thawed, cleaned, err
}

func (h *Hunk) ReplaceStringHide(s string) string {
	return h.ReplaceMarker() +
		h.CommentStart + " " +
		h.Freeze(s) +
		h.CommentEnd +
		"\n" +
		h.ReplaceMarker()
}

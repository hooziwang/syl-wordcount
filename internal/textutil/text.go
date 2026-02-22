package textutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	TabWidth = 4
)

type Decoded struct {
	Text     string
	Encoding string
}

type Metrics struct {
	Chars        int
	Lines        int
	MaxLineWidth int
	AvgLineWidth int
	LineEnding   string
	Language     string
	LinesText    []string
}

type Position struct {
	Line   int
	Column int
}

type MatchPosition struct {
	Line                 int
	Column               int
	OverflowStartColumn  int
	LineEndColumn        int
	Snippet              string
	MatchText            string
	MatchStartByteOffset int
}

func DetectBinary(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}
	ctl := 0
	for _, b := range sample {
		if b == 0 {
			return true
		}
		if b == 9 || b == 10 || b == 13 {
			continue
		}
		if b < 32 || b == 127 {
			ctl++
		}
	}
	ratio := float64(ctl) / float64(len(sample))
	return ratio > 0.30
}

func Decode(data []byte) (Decoded, error) {
	if utf8.Valid(data) {
		return Decoded{Text: string(data), Encoding: "utf-8"}, nil
	}
	if out, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data); err == nil && utf8.Valid(out) {
		return Decoded{Text: string(out), Encoding: "gb18030"}, nil
	}
	if out, err := simplifiedchinese.GBK.NewDecoder().Bytes(data); err == nil && utf8.Valid(out) {
		return Decoded{Text: string(out), Encoding: "gbk"}, nil
	}
	return Decoded{}, fmt.Errorf("无法识别文本编码（支持 utf-8/gbk/gb18030）")
}

func ComputeMetrics(text string) Metrics {
	lineEnding := detectLineEnding(text)
	norm := strings.ReplaceAll(text, "\r\n", "\n")
	norm = strings.ReplaceAll(norm, "\r", "\n")

	if norm == "" {
		return Metrics{LineEnding: lineEnding, Language: "unknown", LinesText: []string{}}
	}
	lines := strings.Split(norm, "\n")
	if strings.HasSuffix(norm, "\n") {
		lines = lines[:len(lines)-1]
	}
	chars := utf8.RuneCountInString(norm)
	maxW := 0
	totalW := 0
	for _, ln := range lines {
		w := DisplayWidth(ln)
		totalW += w
		if w > maxW {
			maxW = w
		}
	}
	avg := 0
	if len(lines) > 0 {
		avg = totalW / len(lines)
	}
	return Metrics{
		Chars:        chars,
		Lines:        len(lines),
		MaxLineWidth: maxW,
		AvgLineWidth: avg,
		LineEnding:   lineEnding,
		Language:     GuessLanguage(norm),
		LinesText:    lines,
	}
}

func DisplayWidth(s string) int {
	col := 0
	for _, r := range s {
		if r == '\t' {
			col += TabWidth - (col % TabWidth)
			continue
		}
		w := runewidth.RuneWidth(r)
		if w <= 0 {
			w = 1
		}
		col += w
	}
	return col
}

func GuessLanguage(s string) string {
	if s == "" {
		return "unknown"
	}
	han := 0
	alpha := 0
	runes := 0
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		runes++
		if unicode.Is(unicode.Han, r) {
			han++
		}
		if r <= unicode.MaxASCII && unicode.IsLetter(r) {
			alpha++
		}
	}
	if runes == 0 {
		return "unknown"
	}
	if float64(han)/float64(runes) > 0.20 {
		return "zh"
	}
	if float64(alpha)/float64(runes) > 0.40 {
		return "en"
	}
	return "unknown"
}

func detectLineEnding(s string) string {
	if s == "" {
		return "none"
	}
	hasCRLF := strings.Contains(s, "\r\n")
	stripped := strings.ReplaceAll(s, "\r\n", "")
	hasLF := strings.Contains(stripped, "\n")
	hasCR := strings.Contains(stripped, "\r")
	switch {
	case hasCRLF && !hasLF && !hasCR:
		return "crlf"
	case !hasCRLF && hasLF && !hasCR:
		return "lf"
	case !hasCRLF && !hasLF && !hasCR:
		return "none"
	default:
		return "mixed"
	}
}

func HashSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func RuneColumnAtByteOffset(line string, byteOffset int) int {
	if byteOffset <= 0 {
		return 1
	}
	if byteOffset > len(line) {
		byteOffset = len(line)
	}
	return utf8.RuneCountInString(line[:byteOffset]) + 1
}

func LineAndColumnByOffset(lines []string, offsets []int, off int) Position {
	if len(lines) == 0 {
		return Position{Line: 0, Column: 0}
	}
	i := sort.Search(len(offsets), func(i int) bool { return offsets[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	line := lines[i]
	col := RuneColumnAtByteOffset(line, off-offsets[i])
	return Position{Line: i + 1, Column: col}
}

func BuildLineOffsets(lines []string) []int {
	off := 0
	o := make([]int, 0, len(lines))
	for _, ln := range lines {
		o = append(o, off)
		off += len(ln) + 1 // + '\n'
	}
	return o
}

func SnippetByRune(text string, offByte int, contextChars int) string {
	if text == "" {
		return ""
	}
	r := []rune(text)
	idxRune := utf8.RuneCountInString(text[:clamp(offByte, 0, len(text))])
	start := idxRune - contextChars
	if start < 0 {
		start = 0
	}
	end := idxRune + contextChars
	if end > len(r) {
		end = len(r)
	}
	snip := string(r[start:end])
	if start > 0 {
		snip = "..." + snip
	}
	if end < len(r) {
		snip += "..."
	}
	return snip
}

func CompilePattern(p string, caseSensitive bool) (*regexp.Regexp, error) {
	if caseSensitive {
		return regexp.Compile(p)
	}
	return regexp.Compile("(?i)" + p)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

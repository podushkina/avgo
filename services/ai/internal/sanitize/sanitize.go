package sanitize

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

const placeholder = "avito-dostavka-pay.example/order"

var urlRe = regexp.MustCompile(
	`(?i)\b(?:https?://)?[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}(?:/[^\s]*)?`)

func isForeignScript(r rune) bool {
	switch {
	case r >= 0x3040 && r <= 0x30FF,
		r >= 0x3400 && r <= 0x4DBF,
		r >= 0x4E00 && r <= 0x9FFF,
		r >= 0xAC00 && r <= 0xD7AF,
		r >= 0xF900 && r <= 0xFAFF:
		return true
	default:
		return false
	}
}

func DropForeignScript(s string) string {
	if !strings.ContainsFunc(s, isForeignScript) {
		return s
	}

	var out strings.Builder
	for _, word := range strings.SplitAfter(s, " ") {
		if !strings.ContainsFunc(word, isForeignScript) {
			out.WriteString(word)
		}
	}
	return out.String()
}

func Text(s string) string {
	s = DropForeignScript(s)
	return urlRe.ReplaceAllStringFunc(s, func(match string) string {
		host := match
		if i := strings.Index(host, "://"); i >= 0 {
			host = host[i+3:]
		}
		if i := strings.IndexByte(host, '/'); i >= 0 {
			host = host[:i]
		}
		if strings.EqualFold(host, "example") || strings.HasSuffix(strings.ToLower(host), ".example") {
			return match
		}
		return placeholder
	})
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '…'
}

func TrimToSentence(s string) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) == 0 {
		return ""
	}
	if isSentenceEnd(runes[len(runes)-1]) {
		return string(runes)
	}

	for i := len(runes) - 1; i >= 0; i-- {
		if isSentenceEnd(runes[i]) {
			return strings.TrimSpace(string(runes[:i+1]))
		}
	}
	return string(runes)
}

type Streamer struct {
	buf      strings.Builder
	emit     func(string) error
	sentence strings.Builder
	seen     map[string]bool
	stopped  bool
}

var ErrRepeat = errors.New("модель начала повторяться")

func NewStreamer(emit func(string) error) *Streamer {
	return &Streamer{emit: emit, seen: map[string]bool{}}
}

func normalizeSentence(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Streamer) trackRepeat(text string) bool {
	for _, r := range text {
		s.sentence.WriteRune(r)
		if r != '.' && r != '!' && r != '?' {
			continue
		}

		key := normalizeSentence(s.sentence.String())
		s.sentence.Reset()
		if len(key) < 12 {
			continue
		}
		if s.seen[key] {
			return true
		}
		s.seen[key] = true
	}
	return false
}

func (s *Streamer) Push(token string) error {
	if s.stopped {
		return ErrRepeat
	}
	s.buf.WriteString(token)

	current := s.buf.String()
	cut := strings.LastIndexFunc(current, unicode.IsSpace)
	if cut < 0 {
		return nil
	}

	ready, rest := current[:cut+1], current[cut+1:]
	s.buf.Reset()
	s.buf.WriteString(rest)

	repeated := s.trackRepeat(ready)
	if err := s.emit(Text(ready)); err != nil {
		return err
	}
	if repeated {
		s.stopped = true
		return ErrRepeat
	}
	return nil
}

func (s *Streamer) Close() error {
	if s.stopped || s.buf.Len() == 0 {
		return nil
	}
	rest := s.buf.String()
	s.buf.Reset()
	return s.emit(Text(rest))
}

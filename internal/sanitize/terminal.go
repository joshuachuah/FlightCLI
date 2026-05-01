package sanitize

import (
	"strings"
	"unicode/utf8"
)

// TerminalString removes terminal control sequences from untrusted text while
// preserving printable Unicode.
func TerminalString(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == 0x1b:
			i += skipEscapeSequence(s[i:])
			continue
		case c == 0x9b:
			i++
			i += skipCSISequence(s[i:])
			continue
		case c == 0x9d:
			i++
			i += skipStringControlSequence(s[i:])
			continue
		case isControlByte(c):
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			i++
			continue
		}
		switch r {
		case 0x9b:
			i += size
			i += skipCSISequence(s[i:])
			continue
		case 0x9d:
			i += size
			i += skipStringControlSequence(s[i:])
			continue
		case 0x90, 0x98, 0x9e, 0x9f:
			i += size
			i += skipStringControlSequence(s[i:])
			continue
		}
		if isControlRune(r) {
			i += size
			continue
		}

		b.WriteString(s[i : i+size])
		i += size
	}

	return b.String()
}

func skipEscapeSequence(s string) int {
	if len(s) < 2 {
		return 1
	}

	switch s[1] {
	case '[':
		return 2 + skipCSISequence(s[2:])
	case ']':
		return 2 + skipStringControlSequence(s[2:])
	case 'P', '^', '_', 'X':
		return 2 + skipStringControlSequence(s[2:])
	}

	if s[1] >= 0x40 && s[1] <= 0x5f {
		return 2
	}
	if strings.ContainsRune("()*+-./", rune(s[1])) {
		if len(s) >= 3 {
			return 3
		}
		return 2
	}

	return 2
}

func skipCSISequence(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x40 && s[i] <= 0x7e {
			return i + 1
		}
	}
	return len(s)
}

func skipStringControlSequence(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 0x07, 0x9c:
			return i + 1
		case 0x1b:
			if i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		if strings.HasPrefix(s[i:], "\u009c") {
			return i + len("\u009c")
		}
	}
	return len(s)
}

func isControlByte(c byte) bool {
	return c < 0x20 || (c >= 0x7f && c <= 0x9f)
}

func isControlRune(r rune) bool {
	return r < 0x20 || (r >= 0x7f && r <= 0x9f)
}

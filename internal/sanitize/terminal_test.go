package sanitize

import "testing"

func TestTerminalStringStripsANSIAndControlSequences(t *testing.T) {
	input := "safe\x1b[31mred\x1b[0m " +
		"\x1b]8;;https://example.test\aClick\x1b]8;;\a " +
		"\x1b]0;spoofed-title\x1b\\done"

	got := TerminalString(input)
	want := "safered Click done"
	if got != want {
		t.Fatalf("TerminalString() = %q, want %q", got, want)
	}
}

func TestTerminalStringStripsControls(t *testing.T) {
	input := "a\x00b\rc\td\ne\u0085f"
	got := TerminalString(input)
	want := "abcdef"
	if got != want {
		t.Fatalf("TerminalString() = %q, want %q", got, want)
	}
}

func TestTerminalStringPreservesPrintableUnicode(t *testing.T) {
	input := "München → 東京 ✈️"
	if got := TerminalString(input); got != input {
		t.Fatalf("TerminalString() = %q, want %q", got, input)
	}
}

func TestTerminalStringStripsRawC1CSI(t *testing.T) {
	input := string([]byte{'a', 0x9b, '3', '1', 'm', 'b'})
	got := TerminalString(input)
	want := "ab"
	if got != want {
		t.Fatalf("TerminalString() = %q, want %q", got, want)
	}
}

func TestTerminalStringStripsUTF8EncodedC1Sequences(t *testing.T) {
	input := "a\u009b31mred\u009d0;spoofed-title\ab"
	got := TerminalString(input)
	want := "aredb"
	if got != want {
		t.Fatalf("TerminalString() = %q, want %q", got, want)
	}
}

func TestTerminalStringStripsEncodedC1Sequences(t *testing.T) {
	input := "a\u009b31mb \u009d0;spoof\u009cc"
	got := TerminalString(input)
	want := "ab c"
	if got != want {
		t.Fatalf("TerminalString() = %q, want %q", got, want)
	}
}

package bot

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCountLine(t *testing.T) {
	if got, want := countLine("Now dead:", "player(s)", []string{"Alice", "Bob"}), "Now dead: 2 player(s): Alice, Bob"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := countLine("Cleared", "character(s)", nil), "Cleared 0 character(s)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChunkLines(t *testing.T) {
	lines := []string{strings.Repeat("a", 6), strings.Repeat("b", 3), strings.Repeat("c", 4)}
	got := chunkLines(lines, 10)
	want := []string{"aaaaaa\nbbb", "cccc"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	// A long line is cut between characters, never inside one.
	for _, chunk := range chunkLines([]string{strings.Repeat("é", 6)}, 4) {
		if !utf8.ValidString(chunk) || utf8.RuneCountInString(chunk) > 4 {
			t.Errorf("multi-byte line cut to %q, want at most 4 whole characters", chunk)
		}
	}
}

package bot

import (
	"slices"
	"strings"
	"testing"
)

func TestGrimoireSummary(t *testing.T) {
	players := map[string]string{"1": "Alice", "2": "Bob", "3": "Carol", "4": "Dave"}
	chars := map[string]*Character{
		"1": {Team: TeamEvil, Alive: true, AnnouncedAlive: true},
		"2": {Team: TeamGood, Alive: false, AnnouncedAlive: true},
		"3": {Team: TeamGood, Alive: true, AnnouncedAlive: true},
	}

	got := grimoireSummary(players, chars)
	want := "Alive 2/3 · Good 2 · Evil 1 · 1 without a character · 1 change(s) not yet announced"
	if got != want {
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
}

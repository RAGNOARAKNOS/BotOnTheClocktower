package bot

import (
	"fmt"
	"slices"
	"strings"
)

// Helpers for building replies, shared by the commands.

// countLine writes a count and the names counted, e.g. "Added 2 player(s): Alice, Bob",
// or "Added 0 player(s)" if there are none.
func countLine(label, unit string, names []string) string {
	line := fmt.Sprintf("%s %d %s", label, len(names), unit)
	if len(names) > 0 {
		line += ": " + strings.Join(names, ", ")
	}
	return line
}

// listLine formats a reply line such as "\nSkipped: Alice, Bob", or "" if there are no items.
func listLine(label string, items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "\n" + label + ": " + strings.Join(items, ", ")
}

// failureList formats failures as a warning to append to a reply, or "" if there are none.
func failureList(failures []string) string {
	if len(failures) == 0 {
		return ""
	}
	return "\nWarning:\n- " + strings.Join(failures, "\n- ")
}

// plural writes a count and a unit, adding "s" unless the count is 1.
func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// sortedNames returns the names (values) of an ID-to-name map in alphabetical order.
func sortedNames(players map[string]string) []string {
	names := make([]string, 0, len(players))
	for _, name := range players {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// chunkLines joins lines with newlines into messages no longer than limit.
// A single line longer than limit is cut.
func chunkLines(lines []string, limit int) []string {
	var chunks []string
	var sb strings.Builder
	for _, line := range lines {
		line = truncate(line, limit)
		if sb.Len() > 0 && sb.Len()+1+len(line) > limit {
			chunks = append(chunks, sb.String())
			sb.Reset()
		}
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(line)
	}
	if sb.Len() > 0 {
		chunks = append(chunks, sb.String())
	}
	return chunks
}

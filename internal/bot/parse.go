package bot

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// mentionPattern matches a user mention as it appears in raw message content.
var mentionPattern = regexp.MustCompile(`<@!?(\d+)>`)

// teamWordNames are characters whose names start with a team word, so that
// `assign @player Evil Twin` isn't read as team Evil, character "Twin".
var teamWordNames = map[string]Team{
	"evil twin": TeamEvil,
}

// parseTeam reads "good" or "evil", ignoring capitals.
func parseTeam(word string) (Team, bool) {
	switch strings.ToLower(word) {
	case "good":
		return TeamGood, true
	case "evil":
		return TeamEvil, true
	}
	return "", false
}

// splitTeam takes the optional team word off the front of a character name.
// Without one, the character is Good.
func splitTeam(name string) (Team, string, error) {
	if team, ok := teamWordNames[strings.ToLower(name)]; ok {
		return team, name, nil
	}

	first, rest := name, ""
	if i := strings.IndexFunc(name, unicode.IsSpace); i >= 0 {
		first, rest = name[:i], strings.TrimSpace(name[i:])
	}

	team, isTeam := parseTeam(first)
	if !isTeam {
		return TeamGood, name, nil
	}
	if rest == "" {
		return "", "", errors.New("put the character's name after the team")
	}
	return team, rest, nil
}

// splitAtMention finds the first user mention on the first line of a raw command
// message. It returns the mentioned user's ID, the rest of that line, and the
// lines after it, untouched.
func splitAtMention(content string) (userID, restOfLine, laterLines string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	firstLine, laterLines, _ := strings.Cut(content, "\n")

	match := mentionPattern.FindStringSubmatchIndex(firstLine)
	if match == nil {
		return "", "", "", errors.New("mention the player on the first line")
	}

	restOfLine = firstLine[match[1]:]
	if mentionPattern.MatchString(restOfLine) {
		return "", "", "", errors.New("mention exactly one player")
	}

	return firstLine[match[2]:match[3]], strings.TrimSpace(restOfLine), laterLines, nil
}

// parseAssignment reads `!botc character assign @player [good|evil] <Character>`
// plus optional guidance on the following lines.
func parseAssignment(content string) (userID string, team Team, name, guidance string, err error) {
	userID, name, guidance, err = splitAtMention(content)
	if err != nil {
		return "", "", "", "", err
	}
	if name == "" {
		return "", "", "", "", errors.New("put the character's name after the mention, on the same line")
	}

	team, name, err = splitTeam(name)
	if err != nil {
		return "", "", "", "", err
	}

	guidance = strings.TrimSpace(guidance)
	switch {
	case utf8.RuneCountInString(name) > maxCharacterName:
		return "", "", "", "", fmt.Errorf("the character's name is longer than %d characters", maxCharacterName)
	case utf8.RuneCountInString(guidance) > maxEmbedDescription:
		return "", "", "", "", fmt.Errorf("the guidance is longer than Discord's limit of %d characters", maxEmbedDescription)
	}

	return userID, team, name, guidance, nil
}

// parseWhisper reads `!botc whisper @player <text>`, where the text starts after
// the mention and can carry on over the following lines.
func parseWhisper(content string) (userID, text string, err error) {
	userID, restOfLine, laterLines, err := splitAtMention(content)
	if err != nil {
		return "", "", err
	}

	text = strings.TrimSpace(restOfLine + "\n" + laterLines)
	switch {
	case text == "":
		return "", "", errors.New("there's nothing to whisper")
	case utf8.RuneCountInString(text) > maxEmbedDescription:
		return "", "", fmt.Errorf("the message is longer than Discord's limit of %d characters", maxEmbedDescription)
	}

	return userID, text, nil
}

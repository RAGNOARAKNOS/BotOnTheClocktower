package bot

import (
	"strings"
	"testing"
)

func TestParseAssignment(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantUserID   string
		wantName     string
		wantGuidance string
		wantErr      bool
	}{
		{
			name:       "name only",
			content:    "!botc character assign <@123> Imp",
			wantUserID: "123",
			wantName:   "Imp",
		},
		{
			name:         "multi-line guidance",
			content:      "!botc character assign <@123> Fortune Teller\nEach night, choose 2 players.\nOne good player registers as a Demon.",
			wantUserID:   "123",
			wantName:     "Fortune Teller",
			wantGuidance: "Each night, choose 2 players.\nOne good player registers as a Demon.",
		},
		{
			name:       "nickname mention",
			content:    "!botc character assign <@!456> Washerwoman",
			wantUserID: "456",
			wantName:   "Washerwoman",
		},
		{
			name:       "mixed case command",
			content:    "!BOTC Character Assign <@123>   Imp  ",
			wantUserID: "123",
			wantName:   "Imp",
		},
		{
			name:         "windows line endings and blank lines around guidance",
			content:      "!botc character assign <@123> Monk\r\n\r\nProtect a player.\r\n\r\n",
			wantUserID:   "123",
			wantName:     "Monk",
			wantGuidance: "Protect a player.",
		},
		{
			name:         "markdown kept",
			content:      "!botc character assign <@123> Spy\n**Bold** and _italic_\n- a list item",
			wantUserID:   "123",
			wantName:     "Spy",
			wantGuidance: "**Bold** and _italic_\n- a list item",
		},
		{
			name:    "no name",
			content: "!botc character assign <@123>\nguidance with no name",
			wantErr: true,
		},
		{
			name:    "no mention",
			content: "!botc character assign Imp",
			wantErr: true,
		},
		{
			name:    "mention on the second line",
			content: "!botc character assign\n<@123> Imp",
			wantErr: true,
		},
		{
			name:    "two mentions",
			content: "!botc character assign <@123> <@456> Imp",
			wantErr: true,
		},
		{
			name:    "guidance too long",
			content: "!botc character assign <@123> Imp\n" + strings.Repeat("x", maxEmbedDescription+1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, name, guidance, err := parseAssignment(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got user %q name %q guidance %q", userID, name, guidance)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if userID != tt.wantUserID || name != tt.wantName || guidance != tt.wantGuidance {
				t.Errorf("got (%q, %q, %q), want (%q, %q, %q)", userID, name, guidance, tt.wantUserID, tt.wantName, tt.wantGuidance)
			}
		})
	}
}

func TestParseWhisper(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantUserID string
		wantText   string
		wantErr    bool
	}{
		{
			name:       "one line",
			content:    "!botc whisper <@123> Your number is 1.",
			wantUserID: "123",
			wantText:   "Your number is 1.",
		},
		{
			name:       "text starts on the next line",
			content:    "!botc whisper <@123>\nYou learn:\n**Alice** is the Imp",
			wantUserID: "123",
			wantText:   "You learn:\n**Alice** is the Imp",
		},
		{
			name:       "text on both lines",
			content:    "!botc whisper <@!123> Tonight:\nwake at 3",
			wantUserID: "123",
			wantText:   "Tonight:\nwake at 3",
		},
		{
			name:    "empty",
			content: "!botc whisper <@123>   \n  ",
			wantErr: true,
		},
		{
			name:    "no mention",
			content: "!botc whisper hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, text, err := parseWhisper(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got user %q text %q", userID, text)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if userID != tt.wantUserID || text != tt.wantText {
				t.Errorf("got (%q, %q), want (%q, %q)", userID, text, tt.wantUserID, tt.wantText)
			}
		})
	}
}

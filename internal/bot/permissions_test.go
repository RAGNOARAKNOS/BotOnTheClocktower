package bot

import (
	"slices"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestMissingPermissions(t *testing.T) {
	tests := []struct {
		name  string
		have  int64
		needs []permission
		want  []string
	}{
		{"has everything", discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceMoveMembers, roomNeeds, nil},
		{"can't view", discordgo.PermissionSendMessages, []permission{viewChannel, sendMessages}, []string{"View Channels"}},
		{"nothing", 0, roomNeeds, []string{"View Channels", "Send Messages", "Send TTS Messages", "Connect", "Move Members"}},
		{"administrator covers all", discordgo.PermissionAdministrator, roomNeeds, nil},
		{"admin channel needs no TTS", discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory, adminChannelNeeds, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := missingPermissions(tt.have, tt.needs); !slices.Equal(got, tt.want) {
				t.Errorf("missingPermissions() = %q, want %q", got, tt.want)
			}
		})
	}
}

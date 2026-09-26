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
		{"has everything", discordgo.PermissionViewChannel | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceMoveMembers, roomNeeds, nil},
		{"can't view", discordgo.PermissionSendMessages, []permission{viewChannel, sendMessages}, []string{"View Channels"}},
		{"nothing", 0, roomNeeds, []string{"View Channels", "Connect", "Move Members"}},
		{"administrator covers all", discordgo.PermissionAdministrator, gameChannelNeeds, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := missingPermissions(tt.have, tt.needs); !slices.Equal(got, tt.want) {
				t.Errorf("missingPermissions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsVillageRoom(t *testing.T) {
	if !isVillageRoom("Town Square") || !isVillageRoom("Storyteller's Corner") {
		t.Error("village rooms not recognised")
	}
	if isVillageRoom("General") || isVillageRoom("town square") {
		t.Error("non-village (or wrongly capitalised) name recognised as a village room")
	}
}

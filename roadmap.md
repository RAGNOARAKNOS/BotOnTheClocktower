# User Management Roadmap

## Priority Improvements

1. Implement `pmove` and `cmove` command handlers end-to-end, including argument validation and move error handling.
2. Track and persist a canonical main channel (Town Square), and add an explicit recall command to bring users back there.
3. Build a durable player registry during player mapping (user ID, display name, current channel, role flags).
4. Populate and enforce `StoryTellerId` during registration so storyteller is excluded from mass moves.
5. Add voice-state event handling (join/leave/move) so player-channel state stays current.
6. Add permission checks so only storyteller/mod/admin users can run management commands.
7. Add move safety checks (skip disconnected users, skip users already in destination, report partial failures).
8. Improve command feedback with concise move summaries and failure details.
9. Replace fragile channel-name matching with stable channel IDs or explicit config mapping.
10. Replace global mutable game state with per-guild state for multi-server support.

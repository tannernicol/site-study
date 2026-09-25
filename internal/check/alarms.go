package check

import "design-public/internal/rules"

// checkAlarms evaluates the smoke/CO alarm requirements for a room. The
// current room-spec format has no field recording which alarms are
// actually installed, so this can never be verified from a room.yml today
// — it is always SKIPPED, loudly, rather than silently assumed compliant.
// If the room spec later grows an alarms: section, this stays a one-line
// data change here plus real fields on room.Room; no rule-file change
// needed.
func checkAlarms(section string, r rules.Rule) []Finding {
	const label = "smoke/CO alarms"
	if r.IsUnverified() {
		return []Finding{skip(section, r, label, "unverified", "unverified rule")}
	}
	return []Finding{skip(section, r, label, "no data", "no alarm-presence data in room spec")}
}

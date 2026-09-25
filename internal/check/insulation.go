package check

import "design-public/internal/rules"

// checkInsulation evaluates the below-grade wall / slab-edge R-value
// requirement. In the current rules file this is confidence: unverified
// with null R-values, so it always SKIPs — that is the point of the rule,
// not a bug: never guess an R-value. It is a project-level check (there is
// no per-room envelope entity in the room spec), so it is evaluated once.
func checkInsulation(r rules.Rule) []Finding {
	const label = "below-grade wall / slab insulation"
	if r.IsUnverified() {
		return []Finding{skip("Project", r, label, "unverified", "unverified rule")}
	}
	if r.BelowGradeWallRValue == nil || r.SlabEdgeRValue == nil {
		return []Finding{skip("Project", r, label, "unverified", "null limit value")}
	}
	// If a future rules revision fills in real, verified R-values, this
	// is where they'd be compared against measured/spec'd values — but
	// the room spec has no such field today, so there is nothing to
	// compare against yet.
	return []Finding{skip("Project", r, label, "no data", "no envelope data in room spec")}
}

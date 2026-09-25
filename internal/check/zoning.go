package check

import "design-public/internal/rules"

// checkZoning evaluates a zoning development-standards rule (FAR, height,
// setbacks, facade length) against the building ENVELOPE -- a fundamentally
// different kind of check from every other shape in this package, which all
// evaluate a room/stair/envelope-insulation fact already present in
// room.Project. room.Project carries no site/zoning fact (no parcel, no
// zone, no proposed FAR/height/setback geometry) for this rule to compare
// against, so it always SKIPs -- same reasoning and convention as
// checkInsulation's own "no room data" skip: the rule is real, cited, and
// surfaced directly, it just has nothing in THIS project model to check yet. A
// future revision that models a proposed envelope's actual FAR/height/
// setbacks would compare those values here instead.
func checkZoning(r rules.Rule) []Finding {
	const label = "zoning development standards (FAR/height/setbacks/facade length)"
	return []Finding{skip("Project", r, label, "no data", "no site/zoning facts in room spec")}
}

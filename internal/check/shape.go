package check

import "design-public/internal/rules"

// shape identifies which small checker a rule's data belongs to. It is
// derived from applies_when keys and which limit fields the rule declares —
// never from the rule's id string — so a new rule using an existing shape
// (new jurisdiction variant, new numbers, new citation) is dispatched
// correctly without a code change. Only a genuinely new combination of
// applies_when/fields needs a new shape and a new checker.
type shape string

const (
	shapeCeilingHeight     shape = "ceiling_height"
	shapeEgressOpening     shape = "egress_opening"
	shapeWindowWell        shape = "window_well"
	shapeHabitableRoomSize shape = "habitable_room_size"
	shapeLightVentilation  shape = "light_ventilation"
	shapeStairs            shape = "stairs"
	shapeAlarms            shape = "alarms"
	shapeInsulation        shape = "insulation"
	shapeZoning            shape = "zoning"
	shapeUnknown           shape = "unknown"
)

// shapeOfElement maps an applies_when.element value to its shape — split
// out of shapeOf so that function's own branching stays under gocyclo's
// threshold as new element-keyed shapes (zoning, ...) are added.
func shapeOfElement(element string) (shape, bool) {
	switch element {
	case "stair":
		return shapeStairs, true
	case "envelope":
		return shapeInsulation, true
	case "zoning":
		return shapeZoning, true
	}
	return shapeUnknown, false
}

// shapeOfLimitFields disambiguates the remaining space_type-gated
// room-level rules by which limit fields they declare — split out of
// shapeOf for the same gocyclo reason as shapeOfElement above.
func shapeOfLimitFields(r rules.Rule) shape {
	switch {
	case r.MinClearHeightIn != nil:
		return shapeCeilingHeight
	case r.SmokeAlarmEachSleepingRoom != nil || r.SmokeAlarmOutsideSleepingArea != nil ||
		r.SmokeAlarmEachLevel != nil || r.CoAlarmOutsideSleepingArea != nil:
		return shapeAlarms
	case r.MinGlazingPctOfFloorArea != nil || r.MinOpenablePctOfFloorArea != nil:
		return shapeLightVentilation
	case r.MinAreaSqft != nil || r.MinHorizontalDimensionIn != nil:
		return shapeHabitableRoomSize
	}
	return shapeUnknown
}

func shapeOf(r rules.Rule) shape {
	if element, ok := r.AppliesWhen["element"].(string); ok {
		if s, ok := shapeOfElement(element); ok {
			return s
		}
	}

	if _, ok := r.AppliesWhen["sill_below_grade"]; ok {
		return shapeWindowWell
	}
	if _, ok := r.AppliesWhen["requires_eero"]; ok {
		return shapeEgressOpening
	}

	return shapeOfLimitFields(r)
}

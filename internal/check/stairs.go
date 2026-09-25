package check

import (
	"fmt"

	"design-public/internal/room"
	"design-public/internal/rules"
)

// stairDim is one stairway dimension checked against one limit. The four
// dimensions differ only in name, limit, measured value and comparison
// direction, so they are data rather than four copies of the same block.
type stairDim struct {
	label string
	limit *float64 // nil when the rule does not carry this limit
	value float64
	// atLeastCmp reports a minimum (value >= limit); otherwise a maximum.
	atLeastCmp bool
}

// checkStairs evaluates stairway geometry. A violation is a FAIL for new
// construction, but only a FLAG for an existing stair — existing stairs may
// be grandfathered, so the design is not failed outright.
func checkStairs(section string, r rules.Rule, st *room.Stair) []Finding {
	dims := []stairDim{
		{"headroom", r.MinHeadroomIn, st.HeadroomIn, true},
		{"riser", r.MaxRiserIn, st.RiserIn, false},
		{"tread", r.MinTreadIn, st.TreadIn, true},
		{"width", r.MinWidthIn, st.WidthIn, true},
	}

	out := make([]Finding, 0, len(dims))
	for _, d := range dims {
		out = append(out, checkStairDim(section, r, st, d))
	}
	return out
}

func checkStairDim(section string, r rules.Rule, st *room.Stair, d stairDim) Finding {
	if r.IsUnverified() || d.limit == nil {
		return skip(section, r, d.label, "unverified", "unverified rule")
	}

	limit := *d.limit
	ok := atMost(d.value, limit)
	pass, fail := "<=", ">"
	if d.atLeastCmp {
		ok = atLeast(d.value, limit)
		pass, fail = ">=", "<"
	}

	bound := "max"
	if d.atLeastCmp {
		bound = "min"
	}

	detail := fmt.Sprintf("%s %s %s %s", fmtIn(d.value), pass, fmtIn(limit), bound)
	if !ok {
		detail = fmt.Sprintf("%s %s %s %s", fmtIn(d.value), fail, fmtIn(limit), bound)
		if st.Existing {
			detail += " — may be grandfathered, verify with the local authority"
		}
	}
	return dimFinding(section, r, d.label, stairStatus(ok, st.Existing), detail,
		d.value, limit, d.atLeastCmp, fmtIn)
}

// stairStatus downgrades a violation to FLAG on an existing stair, which may
// be grandfathered, rather than failing the whole design.
func stairStatus(ok, existing bool) Status {
	switch {
	case ok:
		return Pass
	case existing:
		return Flag
	default:
		return Fail
	}
}

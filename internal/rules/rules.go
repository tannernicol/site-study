// Package rules loads the building-code rules file (data, not code) and
// provides the confidence/applies-when handling that keeps a fabricated
// number from ever reaching a finding.
package rules

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Meta carries the jurisdiction-level facts printed on every run.
type Meta struct {
	Jurisdiction       string `yaml:"jurisdiction"`
	AppliesTo          string `yaml:"applies_to"`
	CodeCycleConfirmed bool   `yaml:"code_cycle_confirmed"`
	SourceEdition      string `yaml:"source_edition"`
	LastReviewed       string `yaml:"last_reviewed"`
}

// Units documents the unit system the numeric fields below are expressed in.
type Units struct {
	Length string `yaml:"length"`
	Area   string `yaml:"area"`
}

// Confidence levels. Anything other than Verified/Likely must never be
// trusted for a pass/fail decision.
const (
	ConfidenceVerified   = "verified"
	ConfidenceLikely     = "likely"
	ConfidenceUnverified = "unverified"
)

// Rule is one entry from checks: in the rules YAML. Every numeric limit is a
// pointer so a YAML `null` (or an omitted key) is distinguishable from a real
// zero — both must be treated as "no verified number", never as zero.
type Rule struct {
	ID          string         `yaml:"id"`
	Label       string         `yaml:"label"`
	Confidence  string         `yaml:"confidence"`
	AppliesWhen map[string]any `yaml:"applies_when"`
	Citation    string         `yaml:"citation"`
	Note        string         `yaml:"note"`

	// FieldConfidence overrides Confidence for individual limits within a
	// rule. A rule can be sourced from a cited document overall while one of
	// its numbers came from somewhere weaker — the egress net-clear-AREA
	// minimum is the live example. Reporting the rule-level confidence for
	// such a field overstates how much the number can be trusted, which is
	// the one failure mode this tool exists to avoid.
	FieldConfidence map[string]string `yaml:"field_confidence"`

	// Ceiling height shape.
	MinClearHeightIn      *float64 `yaml:"min_clear_height_in"`
	MinUnderObstructionIn *float64 `yaml:"min_under_obstruction_in"`

	// Egress opening shape.
	MinNetClearWidthIn  *float64 `yaml:"min_net_clear_width_in"`
	MinNetClearHeightIn *float64 `yaml:"min_net_clear_height_in"`
	MaxSillHeightIn     *float64 `yaml:"max_sill_height_in"`
	MinNetClearAreaSqft *float64 `yaml:"min_net_clear_area_sqft"`

	// Window-well and habitable-room-size shapes share these two field
	// names; internal/check disambiguates by applies_when, not by field
	// presence alone.
	MinHorizontalDimensionIn    *float64 `yaml:"min_horizontal_dimension_in"`
	MinAreaSqft                 *float64 `yaml:"min_area_sqft"`
	LadderRequiredIfDepthOverIn *float64 `yaml:"ladder_required_if_depth_over_in"`

	// Light and ventilation shape.
	MinGlazingPctOfFloorArea  *float64 `yaml:"min_glazing_pct_of_floor_area"`
	MinOpenablePctOfFloorArea *float64 `yaml:"min_openable_pct_of_floor_area"`

	// Stair geometry shape.
	MinHeadroomIn *float64 `yaml:"min_headroom_in"`
	MaxRiserIn    *float64 `yaml:"max_riser_in"`
	MinTreadIn    *float64 `yaml:"min_tread_in"`
	MinWidthIn    *float64 `yaml:"min_width_in"`

	// Alarms shape.
	SmokeAlarmEachSleepingRoom    *bool `yaml:"smoke_alarm_each_sleeping_room"`
	SmokeAlarmOutsideSleepingArea *bool `yaml:"smoke_alarm_outside_sleeping_area"`
	SmokeAlarmEachLevel           *bool `yaml:"smoke_alarm_each_level"`
	CoAlarmOutsideSleepingArea    *bool `yaml:"co_alarm_outside_sleeping_area"`

	// Insulation / envelope shape.
	BelowGradeWallRValue *float64 `yaml:"below_grade_wall_r_value"`
	SlabEdgeRValue       *float64 `yaml:"slab_edge_r_value"`

	// Zoning development-standards shape (e.g. LR1 lowrise multifamily) —
	// FAR/height/setback/facade limits against the building ENVELOPE, not
	// an interior room/stair/window against IRC (this package's original
	// scope). Reuses the same Rule/confidence/citation/applies_when
	// machinery; applies_when: {element: zoning} distinguishes it exactly
	// like element: stair / element: envelope already do above.
	MaxFAR                 *float64 `yaml:"max_far"`
	MaxHeightFt            *float64 `yaml:"max_height_ft"`
	FrontSetbackAvgFt      *float64 `yaml:"front_setback_avg_ft"`
	FrontSetbackMinFt      *float64 `yaml:"front_setback_min_ft"`
	SideSetbackFt          *float64 `yaml:"side_setback_ft"`
	RearSetbackNoAlleyFt   *float64 `yaml:"rear_setback_no_alley_ft"`
	RearSetbackWithAlleyFt *float64 `yaml:"rear_setback_with_alley_ft"`
	MaxFacadePctOfLotLine  *float64 `yaml:"max_facade_pct_of_lot_line"`
}

// Rules is the top-level document.
type Rules struct {
	Meta   Meta   `yaml:"meta"`
	Units  Units  `yaml:"units"`
	Checks []Rule `yaml:"checks"`
}

// Load reads and parses a rules YAML file.
func Load(path string) (*Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file %s: %w", path, err)
	}
	var rs Rules
	if err := yaml.Unmarshal(data, &rs); err != nil {
		return nil, fmt.Errorf("parse rules file %s: %w", path, err)
	}
	return &rs, nil
}

// IsUnverified reports whether this rule's numbers must never be relied on
// for a pass/fail decision, regardless of what any individual field holds.
func (r Rule) IsUnverified() bool {
	return r.Confidence == ConfidenceUnverified
}

// ConfidenceFor returns the confidence to report for a single YAML limit key,
// falling back to the rule-level confidence when no override is declared.
// An override may only ever weaken confidence, never strengthen it: a rule
// sourced at "likely" cannot promote one of its fields to "verified", because
// the citation backing the report is the rule's.
func (r Rule) ConfidenceFor(field string) string {
	override, ok := r.FieldConfidence[field]
	if !ok {
		return r.Confidence
	}
	if confidenceRank(override) < confidenceRank(r.Confidence) {
		return override
	}
	return r.Confidence
}

// confidenceRank orders confidence levels from least to most trusted so that
// ConfidenceFor can enforce weaken-only overrides. Unknown values rank lowest,
// so a typo in the rules file fails safe rather than silently reading as
// verified.
func confidenceRank(c string) int {
	switch c {
	case ConfidenceVerified:
		return 2
	case ConfidenceLikely:
		return 1
	default:
		return 0
	}
}

// Matches reports whether the rule's applies_when conditions are satisfied
// by the given facts. A key present in applies_when but absent from facts
// never matches — there is no default, ambiguous match.
func (r Rule) Matches(facts map[string]any) bool {
	for key, want := range r.AppliesWhen {
		got, ok := facts[key]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(got, want) {
			return false
		}
	}
	return true
}

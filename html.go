package main

import (
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"design-public/internal/check"
	"design-public/internal/room"
	"design-public/internal/rules"
)

// printHTML renders a single, self-contained HTML report per
// docs/SPEC-report.md: inline CSS, zero network requests, no JS required to
// read it. This is the whole adoption surface for someone who will never
// clone the repo or install Go — every dimensional finding carries actual,
// limit, and margin so "PASS" is never the only signal, and a rule that
// couldn't be verified renders as a visible SKIPPED row rather than being
// silently dropped.
func printHTML(w io.Writer, rs *rules.Rules, proj *room.Project, findings []check.Finding, summary check.Summary, generatedAt time.Time) error {
	data := htmlData{
		Project:      proj.Project,
		Source:       proj.Source,
		MeasuredOn:   proj.MeasuredOn,
		Jurisdiction: rs.Meta.Jurisdiction,
		Generated:    generatedAt.Format("2006-01-02 15:04 MST"),
		Summary:      summary,
		Sections:     buildHTMLSections(proj, findings),
	}
	if !rs.Meta.CodeCycleConfirmed {
		edition := rs.Meta.SourceEdition
		if edition == "" {
			edition = "an unrecorded edition"
		}
		data.ShowCodeCycleWarning = true
		data.CodeCycleEdition = edition
	}
	return htmlTemplate.Execute(w, data)
}

// htmlData is the top-level template context for the report.
type htmlData struct {
	Project              string
	Source               string
	MeasuredOn           string
	Jurisdiction         string
	Generated            string
	ShowCodeCycleWarning bool
	CodeCycleEdition     string
	Summary              check.Summary
	Sections             []htmlSection
}

// htmlSection is one room, stair, or project-level block of findings.
type htmlSection struct {
	Heading string
	Rows    []htmlRow
}

// htmlRow is one findings-table row, in the architect's vocabulary — never
// a YAML key name.
type htmlRow struct {
	StatusClass string
	StatusGlyph string
	StatusText  string
	Check       string
	Actual      string
	Limit       string
	Margin      string
	Confidence  string
	Citation    string
}

// buildHTMLSections groups findings into the report's per-room / per-stair
// / project blocks, in the same order the CLI table uses, so the two
// outputs never silently diverge on structure.
func buildHTMLSections(proj *room.Project, findings []check.Finding) []htmlSection {
	var sections []htmlSection
	for _, name := range sectionOrder(proj, findings) {
		var rows []htmlRow
		for _, f := range findings {
			if f.Section == name {
				rows = append(rows, htmlRowFor(f))
			}
		}
		if len(rows) == 0 {
			continue
		}
		sections = append(sections, htmlSection{
			Heading: htmlSectionHeading(proj, name),
			Rows:    rows,
		})
	}
	return sections
}

// htmlRowFor turns one Finding into a table row. A blank Actual/Limit/Margin
// (SKIPPED rows, and the well-ladder trigger check, which is a
// required/not-required flag rather than a min/max measurement) renders as
// an em dash rather than an empty cell, so a reader never mistakes "no
// value" for "zero".
func htmlRowFor(f check.Finding) htmlRow {
	class, glyph, text := statusDisplay(f)
	return htmlRow{
		StatusClass: class,
		StatusGlyph: glyph,
		StatusText:  text,
		Check:       f.Label,
		Actual:      dashIfEmpty(f.Actual),
		Limit:       dashIfEmpty(f.Limit),
		Margin:      dashIfEmpty(f.Margin),
		Confidence:  capitalize(f.Confidence),
		Citation:    f.Citation,
	}
}

// statusDisplay returns the CSS class, glyph, and text label for a finding's
// status. PASS/FAIL/FLAG/SKIPPED never rely on color alone: each carries a
// distinct glyph and a spelled-out word.
func statusDisplay(f check.Finding) (class, glyph, text string) {
	switch f.Status {
	case check.Pass:
		return "pass", "✓", "PASS"
	case check.Fail:
		return "fail", "✕", "FAIL"
	case check.Flag:
		return "flag", "⚠", "FLAG"
	case check.Skipped:
		return "skip", "⦸", fmt.Sprintf("SKIPPED (%s)", f.Reason)
	default:
		return "", "", string(f.Status)
	}
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// htmlSectionHeading renders a room/stair heading in the architect's
// vocabulary (e.g. "Rec room — habitable room, existing basement"), never
// the raw space_type YAML value.
func htmlSectionHeading(proj *room.Project, name string) string {
	if name == "Project" {
		return "Project-level checks (envelope / insulation)"
	}
	for _, rm := range proj.Rooms {
		if rm.Name == name {
			label := humanizeSpaceType(rm.SpaceType)
			if proj.ExistingBasement {
				label += ", existing basement"
			}
			return fmt.Sprintf("%s — %s", rm.Name, label)
		}
	}
	for _, st := range proj.Stairs {
		if st.Name == name {
			if st.Existing {
				return fmt.Sprintf("%s — existing", st.Name)
			}
			return st.Name
		}
	}
	return name
}

// humanizeSpaceType turns a room-spec space_type value into the word an
// architect would actually use. Unknown values fall back to swapping
// underscores for spaces rather than leaking a raw YAML enum into the
// report.
func humanizeSpaceType(spaceType string) string {
	switch spaceType {
	case "habitable":
		return "habitable room"
	case "dwelling_unit":
		return "dwelling unit"
	case "storage":
		return "storage room"
	case "mechanical":
		return "mechanical room"
	default:
		return strings.ReplaceAll(spaceType, "_", " ")
	}
}

// htmlTemplate is the entire report: inline CSS, no external stylesheet, no
// script tag, no remote fonts or images. html/template auto-escapes every
// data value, so a citation or project name containing "<" or "&" can never
// break the markup.
//
// Theme: CSS variables default to a light palette; a screen-only media
// query swaps in a dark palette under prefers-color-scheme, and @media
// print unconditionally forces the light palette back (screen dark mode
// must never leak into a printed page). Room blocks set break-inside:avoid
// so a table is never split across a page boundary, and the disclaimer
// footer is position:fixed under @media print so it repeats on every
// printed page.
var htmlTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Project}} — design-stage code check</title>
<style>
:root{
  --bg:#ffffff; --fg:#161616; --muted:#5a5a5a; --border:#d6d6d6; --card-bg:#f6f6f6;
  --pass:#146c2e; --fail:#a3211a; --flag:#8a5a00; --skip:#4b4b4b;
  --banner-bg:#fff4e0; --banner-border:#c98a00;
}
@media screen and (prefers-color-scheme: dark){
  :root{
    --bg:#14171a; --fg:#e9e9e9; --muted:#a8adb3; --border:#3a3f46; --card-bg:#1c2024;
    --pass:#5fd489; --fail:#ff8078; --flag:#f0b429; --skip:#b7bcc2;
    --banner-bg:#3a2c05; --banner-border:#caa23a;
  }
}
*{box-sizing:border-box;}
body{
  margin:0; padding:24px 24px 72px; background:var(--bg); color:var(--fg);
  font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  line-height:1.45;
}
header.report-header{border-bottom:2px solid var(--border); padding-bottom:12px; margin-bottom:18px;}
h1{font-size:1.35rem; margin:0 0 6px;}
p.meta-line{color:var(--muted); font-size:0.9rem; margin:2px 0;}
.disclaimer{
  background:var(--card-bg); border:1px solid var(--border); border-left:4px solid var(--muted);
  padding:10px 14px; margin:12px 0; font-size:0.85rem;
}
.code-cycle-banner{
  background:var(--banner-bg); border:1px solid var(--banner-border); border-left:5px solid var(--banner-border);
  padding:10px 14px; margin:12px 0; font-weight:600;
}
p.summary{margin:16px 0 0; font-size:1rem;}
p.summary strong{font-variant-numeric:tabular-nums;}
section.room{
  border:1px solid var(--border); border-radius:6px; padding:14px 16px; margin:0 0 20px;
  background:var(--card-bg);
}
section.room h2{margin:0 0 10px; font-size:1.05rem;}
table{width:100%; border-collapse:collapse; font-size:0.85rem;}
th,td{text-align:left; padding:6px 8px; border-bottom:1px solid var(--border); vertical-align:top;}
th{font-size:0.72rem; text-transform:uppercase; letter-spacing:0.03em; color:var(--muted);}
td.status{white-space:nowrap; font-weight:600;}
td.status.pass{color:var(--pass);}
td.status.fail{color:var(--fail);}
td.status.flag{color:var(--flag);}
td.status.skip{color:var(--skip);}
td.margin{white-space:nowrap; font-variant-numeric:tabular-nums;}
td.citation{color:var(--muted); font-size:0.8rem;}
footer.disclaimer-footer{
  margin-top:24px; padding-top:10px; border-top:1px solid var(--border);
  color:var(--muted); font-size:0.75rem;
}
@media print{
  :root, body{background:#ffffff !important; color:#000000 !important;}
  body{padding:12px 16px 70px;}
  .code-cycle-banner{background:#ffffff !important; border-color:#000000 !important; color:#000000 !important;}
  section.room{background:#ffffff !important; border-color:#000000 !important; break-inside:avoid; page-break-inside:avoid;}
  td.status.pass, td.status.fail, td.status.flag, td.status.skip{color:#000000 !important;}
  th,td{border-color:#000000 !important;}
  footer.disclaimer-footer{
    position:fixed; bottom:0; left:0; right:0; background:#ffffff; color:#000000;
    border-top:1px solid #000000; padding:6px 16px; font-size:8pt; margin:0;
  }
}
</style>
</head>
<body>
<header class="report-header">
  <h1>{{.Project}}</h1>
  <p class="meta-line">Date measured: {{.MeasuredOn}} &middot; Measurement source: {{.Source}}</p>
  {{if .Jurisdiction}}<p class="meta-line">Jurisdiction: {{.Jurisdiction}}</p>{{end}}
  <p class="meta-line">Report generated: {{.Generated}}</p>
  <div class="disclaimer">Design-stage sanity check &mdash; <strong>not</strong> a code compliance
  determination. Confirm all findings against the locally adopted code and authority review.
  Rules marked <em>unverified</em> were not checked.</div>
  {{if .ShowCodeCycleWarning}}<div class="code-cycle-banner">CODE CYCLE WARNING: these numbers were
  read from {{.CodeCycleEdition}}, which is NOT confirmed current. Verify against the edition your
  jurisdiction has adopted today before relying on them.</div>{{end}}
  <p class="summary"><strong>{{.Summary.Pass}}</strong> pass &middot; <strong>{{.Summary.Fail}}</strong> fail
  &middot; <strong>{{.Summary.Flag}}</strong> flag &middot; <strong>{{.Summary.Skipped}}</strong> skipped (unverified)</p>
</header>
<main>
{{range .Sections}}
<section class="room">
  <h2>{{.Heading}}</h2>
  <table>
    <thead>
      <tr>
        <th scope="col">Status</th>
        <th scope="col">Check</th>
        <th scope="col">Actual</th>
        <th scope="col">Limit</th>
        <th scope="col">Margin</th>
        <th scope="col">Confidence</th>
        <th scope="col">Citation</th>
      </tr>
    </thead>
    <tbody>
      {{range .Rows}}
      <tr>
        <td class="status {{.StatusClass}}">{{.StatusGlyph}} {{.StatusText}}</td>
        <td>{{.Check}}</td>
        <td>{{.Actual}}</td>
        <td>{{.Limit}}</td>
        <td class="margin">{{.Margin}}</td>
        <td>{{.Confidence}}</td>
        <td class="citation">{{.Citation}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</section>
{{end}}
</main>
<footer class="disclaimer-footer">Design-stage sanity check &mdash; <strong>not</strong> a code compliance
determination. Confirm all findings against the locally adopted code and authority review.
Rules marked <em>unverified</em> were not checked.</footer>
</body>
</html>
`))

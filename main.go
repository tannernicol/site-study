// Command housecheck reads a room-spec YAML and a building-code rules YAML
// and prints objective, cited pass/fail/flag findings. It is a design-stage
// sanity check, not a code compliance determination.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"design-public/internal/check"
	"design-public/internal/room"
	"design-public/internal/rules"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("housecheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of the table")
	htmlOut := fs.Bool("html", false, "emit a single self-contained HTML report instead of the table")
	rulesPath := fs.String("rules", "rules/seattle-residential.yml", "path to the rules YAML")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "usage: housecheck [--json|--html] [--rules path] <room-file.yml>\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	roomPath := fs.Arg(0)

	rs, err := rules.Load(*rulesPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	proj, err := room.Load(roomPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	findings := check.Run(rs, proj)
	summary := check.Summarize(findings)
	exitCode := check.ExitCode(findings)

	switch {
	case *htmlOut:
		if err := printHTML(stdout, rs, proj, findings, summary, time.Now()); err != nil {
			_, _ = fmt.Fprintln(stderr, "error:", err)
			return 2
		}
	case *jsonOut:
		printJSON(stdout, rs, proj, findings, summary)
	default:
		printTable(stdout, rs, proj, findings, summary)
	}

	return exitCode
}

// statusWord renders a Finding's status column, expanding SKIPPED into the
// specific reason it was skipped (e.g. "SKIPPED (unverified)") so a skip
// never reads like a quiet pass.
func statusWord(f check.Finding) string {
	if f.Status == check.Skipped {
		return fmt.Sprintf("SKIPPED (%s)", f.Reason)
	}
	return string(f.Status)
}

// sectionOrder returns the section names in the order they should be
// printed: rooms in project order, then stairs in project order, then the
// project-level section (envelope/insulation and similar), if any findings
// landed there.
func sectionOrder(proj *room.Project, findings []check.Finding) []string {
	var order []string
	for _, rm := range proj.Rooms {
		order = append(order, rm.Name)
	}
	for _, st := range proj.Stairs {
		order = append(order, st.Name)
	}
	order = append(order, "Project")
	return order
}

func sectionHeader(proj *room.Project, name string) string {
	for _, rm := range proj.Rooms {
		if rm.Name == name {
			label := rm.SpaceType
			if proj.ExistingBasement {
				label += ", existing basement"
			}
			return fmt.Sprintf("%s  (%s)", rm.Name, label)
		}
	}
	for _, st := range proj.Stairs {
		if st.Name == name {
			if st.Existing {
				return fmt.Sprintf("%s  (existing)", st.Name)
			}
			return st.Name
		}
	}
	return name
}

func printTable(w io.Writer, rs *rules.Rules, proj *room.Project, findings []check.Finding, summary check.Summary) {
	_, _ = fmt.Fprintf(w, "%s   (source: %s, %s)\n\n", proj.Project, proj.Source, proj.MeasuredOn)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, section := range sectionOrder(proj, findings) {
		var rows []check.Finding
		for _, f := range findings {
			if f.Section == section {
				rows = append(rows, f)
			}
		}
		if len(rows) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(tw, "%s\n", sectionHeader(proj, section))
		for _, f := range rows {
			_, _ = fmt.Fprintf(tw, "  %s\t%s\t%s\t[%s] %s\n", statusWord(f), f.Label, f.Detail, f.Confidence, f.Citation)
		}
		_, _ = fmt.Fprintln(tw)
	}
	_ = tw.Flush()

	_, _ = fmt.Fprintf(w, "Summary: %d pass · %d fail · %d flag · %d skipped\n", summary.Pass, summary.Fail, summary.Flag, summary.Skipped)

	if !rs.Meta.CodeCycleConfirmed {
		edition := rs.Meta.SourceEdition
		if edition == "" {
			edition = "an unrecorded edition"
		}
		_, _ = fmt.Fprintf(w, "CODE CYCLE WARNING: these numbers were read from %s, which is NOT confirmed current. Verify against the locally adopted edition before relying on them.\n", edition)
	}
	_, _ = fmt.Fprintln(w, "NOT a code compliance determination. Confirm with the local authority before submitting anything.")
}

// jsonOutput is the machine-readable shape for --json.
type jsonOutput struct {
	Project            string          `json:"project"`
	Source             string          `json:"source"`
	MeasuredOn         string          `json:"measured_on"`
	Jurisdiction       string          `json:"jurisdiction"`
	CodeCycleConfirmed bool            `json:"code_cycle_confirmed"`
	Findings           []check.Finding `json:"findings"`
	Summary            check.Summary   `json:"summary"`
}

func printJSON(w io.Writer, rs *rules.Rules, proj *room.Project, findings []check.Finding, summary check.Summary) {
	out := jsonOutput{
		Project:            proj.Project,
		Source:             proj.Source,
		MeasuredOn:         proj.MeasuredOn,
		Jurisdiction:       rs.Meta.Jurisdiction,
		CodeCycleConfirmed: rs.Meta.CodeCycleConfirmed,
		Findings:           findings,
		Summary:            summary,
	}
	if findings == nil {
		out.Findings = []check.Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
}

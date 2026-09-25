package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunRejectsInvalidInput(t *testing.T) {
	code, _, stderr := runCLI(t)
	if code != 2 || !strings.Contains(stderr, "usage:") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	missing := filepath.Join(t.TempDir(), "missing.yml")
	code, _, stderr = runCLI(t, "--rules", missing, "examples/synthetic-room.yml")
	if code != 2 || !strings.Contains(stderr, "error:") {
		t.Fatalf("missing rules: code=%d stderr=%q", code, stderr)
	}
}

func TestRunUsesGenericSyntheticExample(t *testing.T) {
	code, stdout, stderr := runCLI(t, "--rules", "rules/template.yml", "examples/synthetic-room.yml")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{"Sample remodel", "SKIPPED (unverified)", "Summary:"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunJSONAndHTMLAreSelfContained(t *testing.T) {
	code, stdout, stderr := runCLI(t, "--json", "--rules", "rules/template.yml", "examples/synthetic-room.yml")
	if code != 0 || stderr != "" {
		t.Fatalf("json: code=%d stderr=%q", code, stderr)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json output: %v", err)
	}
	if _, ok := payload["findings"]; !ok {
		t.Fatal("json has no findings")
	}

	code, stdout, stderr = runCLI(t, "--html", "--rules", "rules/template.yml", "examples/synthetic-room.yml")
	if code != 0 || stderr != "" {
		t.Fatalf("html: code=%d stderr=%q", code, stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout), "<!doctype html>") {
		t.Fatal("html is not a complete document")
	}
	if m := regexp.MustCompile(`(?i)https?://|<script src|@import`).FindString(stdout); m != "" {
		t.Fatalf("html makes a network request: %q", m)
	}
	if !strings.Contains(stdout, "SKIPPED (unverified)") {
		t.Fatal("html hides unverified finding")
	}
}

package county

import (
	"context"
	"strings"
	"testing"
)

func TestKingCountyRejectsShortAddressWithoutNetwork(t *testing.T) {
	_, err := (KingCounty{}).Lookup(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "at least 3") {
		t.Fatalf("Lookup short address error = %v", err)
	}
}

func TestAerialLinkIsAnImageRequest(t *testing.T) {
	link := aerialLink(-122.33, 47.61)
	for _, want := range []string{"World_Imagery", "bbox=", "format=jpg"} {
		if !strings.Contains(link, want) {
			t.Errorf("aerial link missing %q: %s", want, link)
		}
	}
}

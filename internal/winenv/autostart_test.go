package winenv

import (
	"slices"
	"strings"
	"testing"
)

func TestAutostartArgsCreate(t *testing.T) {
	exe := `C:\Program Files\Skvoz\skvoz.exe`
	got := autostartArgs("create", exe)
	want := []string{"/Create", "/TN", "Skvoz", "/SC", "ONLOGON", "/RL", "HIGHEST", "/TR", `"` + exe + `"`, "/F"}
	if !slices.Equal(got, want) {
		t.Fatalf("create args:\n got=%v\nwant=%v", got, want)
	}
	// The exe (which may contain spaces) must be quoted inside the /TR value.
	tr := got[slices.Index(got, "/TR")+1]
	if !strings.HasPrefix(tr, `"`) || !strings.HasSuffix(tr, `"`) {
		t.Fatalf("/TR value must be quoted, got %q", tr)
	}
}

func TestAutostartArgsDeleteAndQuery(t *testing.T) {
	if got, want := autostartArgs("delete", "x"), []string{"/Delete", "/TN", "Skvoz", "/F"}; !slices.Equal(got, want) {
		t.Fatalf("delete args: got=%v want=%v", got, want)
	}
	if got, want := autostartArgs("query", "x"), []string{"/Query", "/TN", "Skvoz"}; !slices.Equal(got, want) {
		t.Fatalf("query args: got=%v want=%v", got, want)
	}
}

func TestAutostartArgsUnknownAction(t *testing.T) {
	if got := autostartArgs("bogus", "x"); got != nil {
		t.Fatalf("unknown action should yield nil, got %v", got)
	}
}

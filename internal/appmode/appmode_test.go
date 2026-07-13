package appmode

import "testing"

func TestIsGUI(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"nil (double-click)", nil, true},
		{"empty (double-click)", []string{}, true},
		{"cli flags", []string{"--strategy", "fake"}, false},
		{"service run", []string{"--service", "run"}, false},
		{"single arg", []string{"--version"}, false},
	}
	for _, c := range cases {
		if got := IsGUI(c.args); got != c.want {
			t.Errorf("%s: IsGUI(%v)=%v, want %v", c.name, c.args, got, c.want)
		}
	}
}

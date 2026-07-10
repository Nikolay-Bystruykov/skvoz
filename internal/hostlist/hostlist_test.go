package hostlist

import (
	"strings"
	"testing"
)

func TestMatch_SubdomainAndExact(t *testing.T) {
	l := New()
	l.Add("youtube.com")
	l.Add("discord.gg")

	match := []string{
		"youtube.com",
		"www.youtube.com",
		"r5---sn-abc.googlevideo.youtube.com",
		"YouTube.com",   // case-insensitive
		"discord.gg.",   // trailing dot
	}
	for _, h := range match {
		if !l.Match(h) {
			t.Errorf("Match(%q) = false, want true", h)
		}
	}

	noMatch := []string{
		"notyoutube.com",   // suffix trickery must not match
		"youtube.com.evil.com",
		"example.com",
		"com",
		"",
	}
	for _, h := range noMatch {
		if l.Match(h) {
			t.Errorf("Match(%q) = true, want false", h)
		}
	}
}

func TestLoadReader_SkipsCommentsAndBlanks(t *testing.T) {
	data := `# YouTube domains
youtube.com

  googlevideo.com
# comment
ytimg.com
`
	l := New()
	n, err := l.LoadReader(strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("loaded %d domains, want 3", n)
	}
	if !l.Match("cdn.ytimg.com") {
		t.Error("expected ytimg.com subdomain to match")
	}
	if l.Match("comment") {
		t.Error("comment line must not become a domain")
	}
}

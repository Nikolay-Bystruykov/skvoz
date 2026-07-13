package assets

import "testing"

func TestListsEmbedsDomains(t *testing.T) {
	l, err := Lists()
	if err != nil {
		t.Fatal(err)
	}
	if !l.Match("youtube.com") || !l.Match("discord.com") {
		t.Fatalf("expected youtube+discord domains embedded, got %d entries", l.Len())
	}
}

func TestYouTubeAndDiscordListsAreSeparate(t *testing.T) {
	yt, err := YouTubeList()
	if err != nil {
		t.Fatal(err)
	}
	if !yt.Match("youtube.com") {
		t.Fatal("youtube list missing youtube.com")
	}
	if yt.Match("discord.com") {
		t.Fatal("youtube list should not contain discord.com")
	}

	dc, err := DiscordList()
	if err != nil {
		t.Fatal(err)
	}
	if !dc.Match("discord.com") {
		t.Fatal("discord list missing discord.com")
	}
	if dc.Match("youtube.com") {
		t.Fatal("discord list should not contain youtube.com")
	}
}

func TestDriverFilesPresent(t *testing.T) {
	files := DriverFiles()
	for _, name := range []string{"WinDivert.dll", "WinDivert64.sys"} {
		if len(files[name]) == 0 {
			t.Fatalf("driver file %q is empty", name)
		}
	}
}

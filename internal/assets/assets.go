// Package assets embeds the resources Skvoz needs to run as a single
// self-contained executable: the target domain lists and the WinDivert driver
// binaries. Embedding them means the user downloads exactly one file, with no
// zip to unpack and no loose DLL/SYS to keep next to the exe.
package assets

import (
	"bytes"
	_ "embed"

	"github.com/skvoz/skvoz/internal/hostlist"
)

//go:embed lists/list-youtube.txt
var youtube []byte

//go:embed lists/list-discord.txt
var discord []byte

// The driver binaries are fetched into internal/assets/bin/ at build time
// (scripts/package.sh and the release workflow). Placeholder files are checked
// in only so `go:embed` resolves during development; the real binaries overwrite
// them before a shipping build.
//
//go:embed bin/WinDivert.dll
var winDivertDLL []byte

//go:embed bin/WinDivert64.sys
var winDivertSys []byte

// listFrom builds a *hostlist.List from one or more embedded list blobs.
func listFrom(blobs ...[]byte) (*hostlist.List, error) {
	l := hostlist.New()
	for _, b := range blobs {
		if _, err := l.LoadReader(bytes.NewReader(b)); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// Lists returns every embedded target domain (YouTube + Discord combined).
func Lists() (*hostlist.List, error) { return listFrom(youtube, discord) }

// YouTubeList returns only the embedded YouTube domains, so the tray can toggle
// YouTube independently of Discord.
func YouTubeList() (*hostlist.List, error) { return listFrom(youtube) }

// DiscordList returns only the embedded Discord domains.
func DiscordList() (*hostlist.List, error) { return listFrom(discord) }

// DriverFiles returns the embedded WinDivert binaries keyed by file name, ready
// to be written next to each other so the loader can find them.
func DriverFiles() map[string][]byte {
	return map[string][]byte{
		"WinDivert.dll":   winDivertDLL,
		"WinDivert64.sys": winDivertSys,
	}
}

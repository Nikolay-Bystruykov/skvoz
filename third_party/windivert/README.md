# WinDivert binaries

Skvoz uses [WinDivert](https://github.com/basil00/WinDivert) to capture and
re-inject packets on Windows. WinDivert is **not** checked into this repository;
its binaries are fetched during packaging (`scripts/package.sh` and the
`release` GitHub Action) and shipped inside the release zip next to `skvoz.exe`:

- `WinDivert.dll`
- `WinDivert64.sys`

WinDivert is distributed under the LGPLv3 / GPLv3 dual license. See the
[WinDivert project](https://github.com/basil00/WinDivert) for full terms. Skvoz
loads WinDivert at runtime through its public C API and does not modify it.

To run a locally built `skvoz.exe`, download WinDivert 2.2.x and place
`WinDivert.dll` and `WinDivert64.sys` in the same folder as the executable.

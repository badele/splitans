## [1.0.0] (2026-02-25)

### Features

- update rising error (#30)
- define default SAUCE date (#29)
- fix neotex parser (#25)
- ignore null & space character (#22)
- update metadata (#21)
- update neotex versioning (#20)
- add hyperlink and foreground and background hover color (#18)
- **hyperlink:** implement OSC 8 hyperlink handling with CP437 legacy mode (#16)
- add SAUCE metadata support for import and export (#14)
- add content bounds (#9)
- **virtualterminal:** add GetContentBounds (#8)
- **exporter:** add inline output mode for single-line export (#5)
- **neotex:** parse !TW width header and propagate to output width (#3)
- add public splitans API and improve ANSI handling (#2)

### Bug Fixes

- sauce dimension (#26)
- **neotex:** calculate true width correctly in inline mode (#7)
- **virtualterminal:** ignore CR/LF after soft wrap (#4)

### Miscellaneous

- fix release please version (#27)
- refactor: add EXPORTED/PRIVATE section separators to Go files (#10)

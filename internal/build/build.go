// Package build carries the release identity stamped in at link time.
package build

import "runtime/debug"

// Overridden by the release build with -ldflags -X.
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

// Info describes the running binary.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
}

// Current reports the running binary's identity, falling back to the module
// version recorded by the toolchain so `go install` builds still say something
// truthful.
func Current() Info {
	info := Info{Version: Version, Commit: Commit, Date: Date, Go: "unknown"}

	if read, ok := debug.ReadBuildInfo(); ok {
		info.Go = read.GoVersion
		if info.Version == "" && read.Main.Version != "" {
			info.Version = read.Main.Version
		}
		for _, setting := range read.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = setting.Value
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = setting.Value
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}
	return info
}

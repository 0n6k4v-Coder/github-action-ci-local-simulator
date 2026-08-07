package app

import (
	"fmt"
	"runtime/debug"
)

// Version is the application version.
// This is set at build time via -ldflags.
var Version = "dev"

// Commit is the git commit hash.
// This is set at build time via -ldflags.
var Commit = ""

// Date is the build date.
// This is set at build time via -ldflags.
var Date = ""

func init() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			Version = info.Main.Version
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if len(setting.Value) >= 7 {
						Commit = setting.Value[:7]
					} else {
						Commit = setting.Value
					}
				case "vcs.time":
					Date = setting.Value
				}
			}
		}
	}
}

// VersionString returns the formatted version string.
func VersionString() string {
	return fmt.Sprintf("gacils version %s\ncommit: %s\ndate: %s", Version, Commit, Date)
}

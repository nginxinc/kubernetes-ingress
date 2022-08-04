package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

func getBuildInfo() (string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			commitHash = kv.Value
		case "vcs.time":
			commitTime = kv.Value
		case "vcs.modified":
			dirtyBuild = kv.Value == "true"
		}
	}
	binaryInfo := fmt.Sprintf("Commit=%v Date=%v DirtyState=%v Arch=%v/%v Go=%v", commitHash, commitTime, dirtyBuild, runtime.GOOS, runtime.GOARCH, runtime.Version())
	versionInfo := fmt.Sprintf("Version=%v", version)

	return versionInfo, binaryInfo
}

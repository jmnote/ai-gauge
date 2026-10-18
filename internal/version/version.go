// Package version holds the app version. build.ps1 sets it at link time with
// -ldflags "-X github.com/jmnote/ai-gauge/internal/version.Version=...", so it
// already has its final value before any package initializes.
package version

var Version = "v0.0.0"

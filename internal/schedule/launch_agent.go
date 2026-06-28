package schedule

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecutablePath}}</string>
        <string>summary</string>
        <string>--ai</string>
        <string>--share</string>
        <string>{{.SharePlatform}}</string>
        {{if .Profile}}
        <string>--profile</string>
        <string>{{.Profile}}</string>
        {{end}}
    </array>
    <key>StartCalendarInterval</key>
    {{if eq .Day -1}}
    <dict>
        <key>Hour</key>
        <integer>{{.Hour}}</integer>
        <key>Minute</key>
        <integer>{{.Minute}}</integer>
    </dict>
    {{else}}
    <dict>
        <key>Day</key>
        <integer>{{.Day}}</integer>
        <key>Hour</key>
        <integer>{{.Hour}}</integer>
        <key>Minute</key>
        <integer>{{.Minute}}</integer>
    </dict>
    {{end}}
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.ErrorPath}}</string>
</dict>
</plist>`

// LaunchAgentConfig parameters for an automated summary job.
type LaunchAgentConfig struct {
	Label          string
	ExecutablePath string
	SharePlatform  string
	Hour           int
	Minute         int
	Day            int // macOS plist: 0 is Sunday, 1 is Monday... -1 for daily
	Profile        string
	LogPath        string
	ErrorPath      string
}

// InstallLaunchAgent generates and registers a macOS background job.
func InstallLaunchAgent(cfg LaunchAgentConfig) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Library", "LaunchAgents")
	_ = os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, cfg.Label+".plist")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, _ := template.New("plist").Parse(plistTemplate)
	return tmpl.Execute(f, cfg)
}

// RenderLaunchAgent generates the XML content of a LaunchAgent for testing.
func RenderLaunchAgent(cfg LaunchAgentConfig) (string, error) {
	var b strings.Builder
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(&b, cfg)
	return b.String(), err
}

// RemoveLaunchAgent deletes a macOS background job.
func RemoveLaunchAgent(label string) error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library", "LaunchAgents", label+".plist")
	return os.Remove(path) // err might be os.ErrNotExist
}

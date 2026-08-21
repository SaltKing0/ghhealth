package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Desktop sends a desktop notification via notify-send (Linux) or
// osascript (macOS). Returns nil if the platform is unsupported or the
// command fails (best-effort).
func Desktop(e Event, opts Options) error {
	app := opts.AppName
	if app == "" {
		app = "ghhealth"
	}
	switch runtime.GOOS {
	case "linux":
		icon := "dialog-warning"
		if e.Severity == SeverityCritical {
			icon = "dialog-error"
		}
		return exec.Command("notify-send",
			"-i", icon,
			"-a", app,
			e.Title,
			e.Description,
		).Run()
	case "darwin":
		msg := fmt.Sprintf("%s: %s", e.Title, e.Description)
		return exec.Command("osascript", "-e",
			fmt.Sprintf(`display notification "%s" with title "%s"`, msg, app),
		).Run()
	}
	return nil
}

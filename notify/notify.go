// Package notify provides shared notification channels (desktop + Telegram)
// for the kami family tools.
package notify

// Severity mirrors the statuspage impact levels.
type Severity string

const (
	SeverityMinor    Severity = "minor"
	SeverityMajor    Severity = "major"
	SeverityCritical Severity = "critical"
)

// Event is fired when a status change is detected.
type Event struct {
	Title       string
	Description string
	Severity    Severity
	URL         string
}

// Options configures the notification channels.
type Options struct {
	// TelegramBotToken and TelegramChatID enable Telegram alerts. Empty
	// values disable the channel (best-effort).
	TelegramBotToken string
	TelegramChatID   string
	// AppName is the app name shown in desktop notifications.
	// Defaults to "ghhealth" when empty.
	AppName string
}

// Send dispatches the event to all configured channels (desktop + telegram).
// Errors are best-effort — they are returned but the other channels still fire.
func Send(e Event, opts Options) []error {
	var errs []error
	if err := Desktop(e, opts); err != nil {
		errs = append(errs, err)
	}
	if err := Telegram(e, opts); err != nil {
		errs = append(errs, err)
	}
	return errs
}

package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var sendMessageURL = "https://api.telegram.org/bot%s/sendMessage"

// Telegram sends a notification via the Telegram Bot API. Returns nil when
// the options do not carry credentials (best-effort).
func Telegram(e Event, opts Options) error {
	if opts.TelegramBotToken == "" || opts.TelegramChatID == "" {
		return nil
	}

	emoji := "⚠️"
	if e.Severity == SeverityCritical {
		emoji = "🔥"
	}

	text := fmt.Sprintf("%s *%s*\n%s\n%s", emoji, e.Title, e.Description, e.URL)
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":                  opts.TelegramChatID,
		"text":                     text,
		"parse_mode":               "Markdown",
		"disable_web_page_preview": true,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf(sendMessageURL, opts.TelegramBotToken),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}

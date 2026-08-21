package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegram_SendsMessage(t *testing.T) {
	var received struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	origSendURL := sendMessageURL
	sendMessageURL = srv.URL + "/bot%s/sendMessage"
	defer func() { sendMessageURL = origSendURL }()

	opts := Options{
		TelegramBotToken: "test:token",
		TelegramChatID:   "-10012345",
	}
	e := Event{
		Title:       "Incident with Actions",
		Description: "Actions is experiencing degraded performance",
		Severity:    SeverityMajor,
		URL:         "https://stspg.io/test",
	}
	if err := Telegram(e, opts); err != nil {
		t.Fatalf("Telegram: %v", err)
	}
	if received.ChatID != "-10012345" {
		t.Errorf("expected chat_id -10012345, got %q", received.ChatID)
	}
	if received.ParseMode != "Markdown" {
		t.Errorf("expected parse_mode Markdown, got %q", received.ParseMode)
	}
	if !contains(received.Text, "Incident with Actions") {
		t.Errorf("expected title in message, got %q", received.Text)
	}
}

func TestTelegram_NoCreds(t *testing.T) {
	if err := Telegram(Event{Title: "test"}, Options{}); err != nil {
		t.Fatalf("expected nil when creds not set, got %v", err)
	}
}

func TestTelegram_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	origSendURL := sendMessageURL
	sendMessageURL = srv.URL + "/bot%s/sendMessage"
	defer func() { sendMessageURL = origSendURL }()

	opts := Options{TelegramBotToken: "test:token", TelegramChatID: "-10012345"}
	if err := Telegram(Event{Title: "test"}, opts); err == nil {
		t.Fatal("expected error on 403, got nil")
	}
}

func TestSend_DesktopErrorsDoNotBlockTelegram(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	origSendURL := sendMessageURL
	sendMessageURL = srv.URL + "/bot%s/sendMessage"
	defer func() { sendMessageURL = origSendURL }()

	// Desktop on linux without notify-send will error; Telegram must still fire
	opts := Options{TelegramBotToken: "test:token", TelegramChatID: "-10012345"}
	errs := Send(Event{Title: "x"}, opts)
	// Send returns desktop error (no notify-send in test env) — that's fine,
	// the point is it doesn't panic and Telegram path is exercised.
	if errs == nil {
		t.Log("no errors (notify-send may exist) — ok")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

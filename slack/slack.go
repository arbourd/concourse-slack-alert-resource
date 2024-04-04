package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Message represents a Slack API message
// https://api.slack.com/docs/messages
type Message struct {
	Attachments []Attachment `json:"attachments"`
	Channel     string       `json:"channel,omitempty"`
}

// Attachment represents a Slack API message attachment
// https://api.slack.com/docs/message-attachments
type Attachment struct {
	Fallback   string  `json:"fallback"`
	Color      string  `json:"color"`
	AuthorName string  `json:"author_name"`
	Fields     []Field `json:"fields"`
	Footer     string  `json:"footer"`
	FooterIcon string  `json:"footer_icon"`
	Text       string  `json:"text"`
}

// Field represents a Slack API message attachment's fields
// https://api.slack.com/docs/message-attachments
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Send sends the message to the webhook URL.
func Send(url string, m *Message, maxRetryTime time.Duration) error {
	buf, err := json.Marshal(m)
	if err != nil {
		return err
	}

	err = backoff.Retry(
		func() error {
			r, err := http.Post(url, "application/json", bytes.NewReader(buf))
			if err != nil {
				return err
			}
			defer r.Body.Close()

			if r.StatusCode > 399 {
				return fmt.Errorf("unexpected response status code: %d", r.StatusCode)
			}
			return nil
		},
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxRetryTime)),
	)

	if err != nil {
		return err
	}
	return nil
}

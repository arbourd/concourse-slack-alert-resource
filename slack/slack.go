package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v7"
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

	// The context bounds in-flight requests, which WithMaxElapsedTime alone does not.
	ctx, cancel := context.WithTimeout(context.Background(), maxRetryTime)
	defer cancel()

	_, err = backoff.Retry(ctx,
		func() (struct{}, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
			if err != nil {
				return struct{}{}, backoff.Permanent(err)
			}
			req.Header.Set("Content-Type", "application/json")

			r, err := http.DefaultClient.Do(req)
			if err != nil {
				return struct{}{}, err
			}
			defer r.Body.Close()
			// Drain so the connection can be reused by the next attempt.
			_, _ = io.Copy(io.Discard, r.Body)

			if r.StatusCode <= 399 {
				return struct{}{}, nil
			}
			err = fmt.Errorf("unexpected response status code: %d", r.StatusCode)

			if r.StatusCode == http.StatusTooManyRequests {
				if secs, perr := strconv.Atoi(r.Header.Get("Retry-After")); perr == nil && secs > 0 {
					return struct{}{}, backoff.RetryAfter(time.Duration(secs)*time.Second, err)
				}
			} else if r.StatusCode < 500 {
				return struct{}{}, backoff.Permanent(err)
			}
			return struct{}{}, err
		},
		backoff.WithMaxElapsedTime(maxRetryTime),
	)

	// Surface the underlying error rather than backoff's wrapper.
	if re := backoff.AsRetryError(err); re != nil && re.LastErr != nil {
		return re.LastErr
	}
	return err
}

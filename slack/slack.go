package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Payload struct {
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Fallback   string  `json:"fallback"`
	Color      string  `json:"color"`
	AuthorName string  `json:"author_name"`
	Fields     []Field `json:"fields"`
	Footer     string  `json:"footer"`
	FooterIcon string  `json:"footer_icon"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Send makes a request to the URL.
func Send(url string, p *Payload) error {
	buf, err := json.Marshal(p)
	if err != nil {
		return err
	}

	res, err := http.Post(url, "application/json", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	return nil
}

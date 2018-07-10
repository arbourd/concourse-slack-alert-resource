package main

import (
	"reflect"
	"testing"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
)

func TestNewAlert(t *testing.T) {
	cases := map[string]struct {
		input *concourse.OutRequest
		want  Alert
	}{
		// Default and overrides.
		"default": {
			input: &concourse.OutRequest{},
			want:  Alert{Type: "default", Color: "#35495c", IconURL: "https://ci.concourse-ci.org/public/images/favicon-pending.png"},
		},
		"custom params": {
			input: &concourse.OutRequest{
				Source: concourse.Source{Channel: "general"},
				Params: concourse.OutParams{Channel: "custom-channel", Color: "#ffffff", Message: "custom-message", Disable: true},
			},
			want: Alert{Type: "default", Channel: "custom-channel", Color: "#ffffff", IconURL: "https://ci.concourse-ci.org/public/images/favicon-pending.png", Message: "custom-message", Disabled: true},
		},
		"custom source": {
			input: &concourse.OutRequest{
				Source: concourse.Source{Channel: "general", Disable: true},
			},
			want: Alert{Type: "default", Channel: "general", Color: "#35495c", IconURL: "https://ci.concourse-ci.org/public/images/favicon-pending.png", Disabled: true},
		},

		// Alert types.
		"success": {
			input: &concourse.OutRequest{Params: concourse.OutParams{AlertType: "success"}},
			want:  Alert{Type: "success", Color: "#32cd32", IconURL: "https://ci.concourse-ci.org/public/images/favicon-succeeded.png", Message: "Success"},
		},
		"failed": {
			input: &concourse.OutRequest{Params: concourse.OutParams{AlertType: "failed"}},
			want:  Alert{Type: "failed", Color: "#d00000", IconURL: "https://ci.concourse-ci.org/public/images/favicon-failed.png", Message: "Failed"},
		},
		"started": {
			input: &concourse.OutRequest{Params: concourse.OutParams{AlertType: "started"}},
			want:  Alert{Type: "started", Color: "#f7cd42", IconURL: "https://ci.concourse-ci.org/public/images/favicon-started.png", Message: "Started"},
		},
		"aborted": {
			input: &concourse.OutRequest{Params: concourse.OutParams{AlertType: "aborted"}},
			want:  Alert{Type: "aborted", Color: "#8d4b32", IconURL: "https://ci.concourse-ci.org/public/images/favicon-aborted.png", Message: "Aborted"},
		},
		"fixed": {
			input: &concourse.OutRequest{Params: concourse.OutParams{AlertType: "fixed"}},
			want:  Alert{Type: "fixed", Color: "#32cd32", IconURL: "https://ci.concourse-ci.org/public/images/favicon-succeeded.png", Message: "Fixed"},
		},
		"broke": {
			input: &concourse.OutRequest{Params: concourse.OutParams{AlertType: "broke"}},
			want:  Alert{Type: "broke", Color: "#d00000", IconURL: "https://ci.concourse-ci.org/public/images/favicon-failed.png", Message: "Broke"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := NewAlert(c.input)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("unexpected Alert from NewAlert:\n\t(GOT): %#v\n\t(WNT): %#v", got, c.want)
			}
		})
	}
}

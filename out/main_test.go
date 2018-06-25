package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
	"github.com/arbourd/concourse-slack-alert-resource/slack"
)

func TestOut(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bad.Close()

	env := map[string]string{
		"ATC_EXTERNAL_URL":    "https://ci.example.com",
		"BUILD_TEAM_NAME":     "main",
		"BUILD_PIPELINE_NAME": "demo",
		"BUILD_JOB_NAME":      "test",
		"BUILD_NAME":          "2",
	}

	cases := map[string]struct {
		outRequest *concourse.OutRequest
		want       *concourse.OutResponse
		env        map[string]string
		err        bool
	}{
		"default alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"success alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "success"},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "success"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"failed alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "failed"},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "failed"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"started alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "started"},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "started"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"aborted alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "aborted"},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "aborted"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"custom alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{
					AlertType: "non-existant-type",
					Message:   "Deploying",
					Color:     "#ffffff",
				},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"override channel at Source": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, Channel: "#source"},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "channel", Value: "#source"},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"override channel at Params": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, Channel: "#source"},
				Params: concourse.OutParams{Channel: "#params"},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "channel", Value: "#params"},
					concourse.Metadata{Name: "alerted", Value: "true"},
				},
			},
			env: env,
		},
		"disable alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: bad.URL},
				Params: concourse.OutParams{Disable: true},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "channel", Value: ""},
					concourse.Metadata{Name: "alerted", Value: "false"},
				},
			},
			env: env,
		},
		"error without Slack URL": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ""},
			},
			env: env,
			err: true,
		},
		"error with bad request": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: bad.URL},
			},
			env: env,
			err: true,
		},
		"error without basic auth for fixed type": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, Username: "", Password: ""},
				Params: concourse.OutParams{AlertType: "fixed"},
			},
			env: env,
			err: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range c.env {
				os.Setenv(k, v)
			}

			got, err := out(c.outRequest)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from out:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from out:\n\t(GOT): nil")
			} else if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("unexpected concourse.OutResponse value from out:\n\t(GOT): %#v\n\t(WNT): %#v", got, c.want)
			}
		})
	}
}
func TestBuildSlackMessage(t *testing.T) {
	cases := map[string]struct {
		alert    *Alert
		metadata *concourse.BuildMetadata
		want     *slack.Payload
	}{
		"empty channel": {
			alert: &Alert{
				Type:    "default",
				Color:   "#ffffff",
				IconURL: "",
				Message: "Testing",
			},
			want: &slack.Payload{
				Attachments: []slack.Attachment{
					slack.Attachment{
						Fallback:   "Testing: demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Color:      "#ffffff",
						AuthorName: "Testing",
						Fields: []slack.Field{
							slack.Field{Title: "Job", Value: "demo/test", Short: true},
							slack.Field{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
				Channel: ""},
		},
		"channel and url set": {
			alert: &Alert{
				Type:    "default",
				Channel: "general",
				Color:   "#ffffff",
				IconURL: "",
				Message: "Testing",
			},
			want: &slack.Payload{
				Attachments: []slack.Attachment{
					slack.Attachment{
						Fallback:   "Testing: demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Color:      "#ffffff",
						AuthorName: "Testing",
						Fields: []slack.Field{
							slack.Field{Title: "Job", Value: "demo/test", Short: true},
							slack.Field{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
				Channel: "general"},
		},
	}

	metadata := &concourse.BuildMetadata{
		URL:          "https://ci.example.com",
		TeamName:     "main",
		PipelineName: "demo",
		JobName:      "test",
		BuildName:    "1",
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := buildSlackMessage(c.alert, metadata)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("unexpected slack.Payload value from buildSlackMessage:\n\t(GOT): %#v\n\t(WNT): %#v", got, c.want)
			}
		})
	}
}

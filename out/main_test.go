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

	cases := map[string]struct {
		outRequest *concourse.OutRequest
		want       *concourse.OutResponse
		err        bool
	}{
		"default alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL: ok.URL,
				},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "alerted", Value: "true"},
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "message", Value: ""},
					concourse.Metadata{Name: "color", Value: "#35495c"},
				},
			},
		},
		"params override": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL: ok.URL,
				},
				Params: concourse.OutParams{
					AlertType: "non-legit-type",
					Message:   "Deploying",
					Color:     "#ffffff",
				},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "alerted", Value: "true"},
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "message", Value: "Deploying"},
					concourse.Metadata{Name: "color", Value: "#ffffff"},
				},
			},
		},
		"disable does not send alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL: bad.URL,
				},
				Params: concourse.OutParams{
					Disable: true,
				},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
					concourse.Metadata{Name: "alerted", Value: "false"},
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "message", Value: ""},
					concourse.Metadata{Name: "color", Value: "#35495c"},
				},
			},
		},
		"error without Slack URL": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL: "",
				},
			},
			err: true,
		},
		"error with bad request": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL: bad.URL,
				},
			},
			err: true,
		},
		"error without basic auth for fixed type": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL:      ok.URL,
					Username: "",
					Password: "",
				},
				Params: concourse.OutParams{
					AlertType: "fixed",
				},
			},
			err: true,
		},
	}

	os.Setenv("ATC_EXTERNAL_URL", "https://concourse.com")
	os.Setenv("BUILD_TEAM_NAME", "main")
	os.Setenv("BUILD_PIPELINE_NAME", "demo")
	os.Setenv("BUILD_JOB_NAME", "test")
	os.Setenv("BUILD_NAME", "2")

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
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
	alert := &Alert{
		Name:    "default",
		Color:   "#ffffff",
		IconURL: "",
		Message: "Testing",
	}
	metadata := &concourse.BuildMetadata{
		URL:          "https://ci.example.com",
		TeamName:     "main",
		PipelineName: "demo",
		JobName:      "test",
		BuildName:    "1",
	}

	cases := map[string]struct {
		channel  string
		alert    *Alert
		metadata *concourse.BuildMetadata
		want     *slack.Payload
	}{
		"empty channel": {
			channel:  "",
			alert:    alert,
			metadata: metadata,
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
			channel:  "general",
			alert:    alert,
			metadata: metadata,
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

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := buildSlackMessage(c.channel, c.alert, c.metadata)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("unexpected slack.Payload value from buildSlackMessage:\n\t(GOT): %#v\n\t(WNT): %#v", got, c.want)
			}
		})
	}
}

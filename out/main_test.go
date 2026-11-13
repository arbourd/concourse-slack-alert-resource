package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
	"github.com/arbourd/concourse-slack-alert-resource/slack"
	"github.com/google/go-cmp/cmp"
)

func TestOut(t *testing.T) {
	// Keep the "bad request" retry loop short.
	old := maxElapsedTime
	maxElapsedTime = 500 * time.Millisecond
	t.Cleanup(func() { maxElapsedTime = old })

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bad.Close()

	// Fake Concourse API that reports the previous build (build 1) with a fixed status.
	concourseWith := func(status string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(concourse.Build{Name: "1", Status: status})
		}))
	}
	prevOK := concourseWith("succeeded")
	defer prevOK.Close()
	prevFailed := concourseWith("failed")
	defer prevFailed.Close()

	alerted := func(atype, channel, alerted string) *concourse.OutResponse {
		return &concourse.OutResponse{
			Version: concourse.Version{"ver": "static"},
			Metadata: []concourse.Metadata{
				{Name: "type", Value: atype},
				{Name: "channel", Value: channel},
				{Name: "alerted", Value: alerted},
			},
		}
	}

	env := map[string]string{
		"ATC_EXTERNAL_URL":    "https://ci.example.com",
		"BUILD_TEAM_NAME":     "main",
		"BUILD_PIPELINE_NAME": "demo",
		"BUILD_JOB_NAME":      "test",
		"BUILD_NAME":          "2",
	}

	// The first build has no previous build to compare against.
	firstBuild := map[string]string{}
	for k, v := range env {
		firstBuild[k] = v
	}
	firstBuild["BUILD_NAME"] = "1"

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
			want: alerted("default", "", "true"),
			env:  env,
		},
		"success alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "success"},
			},
			want: alerted("success", "", "true"),
			env:  env,
		},
		"failed alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "failed"},
			},
			want: alerted("failed", "", "true"),
			env:  env,
		},
		"started alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "started"},
			},
			want: alerted("started", "", "true"),
			env:  env,
		},
		"aborted alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "aborted"},
			},
			want: alerted("aborted", "", "true"),
			env:  env,
		},
		"custom alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{
					AlertType: "non-existent-type",
					Message:   "Deploying",
					Color:     "#ffffff",
				},
			},
			want: alerted("default", "", "true"),
			env:  env,
		},
		"override channel at Source": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, Channel: "#source"},
			},
			want: alerted("default", "#source", "true"),
			env:  env,
		},
		"override channel at Params": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, Channel: "#source"},
				Params: concourse.OutParams{Channel: "#params"},
			},
			want: alerted("default", "#params", "true"),
			env:  env,
		},
		"disable alert": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: bad.URL},
				Params: concourse.OutParams{Disable: true},
			},
			want: alerted("default", "", "false"),
			env:  env,
		},
		"fixed after failure": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, ConcourseURL: prevFailed.URL},
				Params: concourse.OutParams{AlertType: "fixed"},
			},
			want: alerted("fixed", "", "true"),
			env:  env,
		},
		"fixed after success is suppressed": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, ConcourseURL: prevOK.URL},
				Params: concourse.OutParams{AlertType: "fixed"},
			},
			want: alerted("fixed", "", "false"),
			env:  env,
		},
		"broke after success": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, ConcourseURL: prevOK.URL},
				Params: concourse.OutParams{AlertType: "broke"},
			},
			want: alerted("broke", "", "true"),
			env:  env,
		},
		"broke after failure is suppressed": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL, ConcourseURL: prevFailed.URL},
				Params: concourse.OutParams{AlertType: "broke"},
			},
			want: alerted("broke", "", "false"),
			env:  env,
		},
		"fixed on first build is suppressed": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "fixed"},
			},
			want: alerted("fixed", "", "false"),
			env:  firstBuild,
		},
		"broke on first build is suppressed": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{URL: ok.URL},
				Params: concourse.OutParams{AlertType: "broke"},
			},
			want: alerted("broke", "", "false"),
			env:  firstBuild,
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
				t.Setenv(k, v)
			}

			got, err := out(c.outRequest, "")
			if err != nil && !c.err {
				t.Fatalf("unexpected error from out:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from out:\n\t(GOT): nil")
			} else if !cmp.Equal(got, c.want) {
				t.Fatalf("unexpected concourse.OutResponse value from out:\n\t(GOT): %#v\n\t(WNT): %#v\n\t(DIFF): %v", got, c.want, cmp.Diff(got, c.want))
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	cases := map[string]struct {
		alert Alert
		want  *slack.Message
	}{
		"empty channel": {
			alert: Alert{
				Type:    "default",
				Color:   "#ffffff",
				IconURL: "",
				Message: "Testing",
			},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback:   "Testing: demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Color:      "#ffffff",
						AuthorName: "Testing",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
				Channel: ""},
		},
		"channel and url set": {
			alert: Alert{
				Type:    "default",
				Channel: "general",
				Color:   "#ffffff",
				IconURL: "",
				Message: "Testing",
			},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback:   "Testing: demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Color:      "#ffffff",
						AuthorName: "Testing",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
				Channel: "general"},
		},
		"message file": {
			alert: Alert{
				Type:        "default",
				Message:     "Testing",
				MessageFile: "test_file",
			},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback:   "filecontents: demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						AuthorName: "filecontents",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
			},
		},
		"message file failure": {
			alert: Alert{
				Type:        "default",
				Message:     "Testing",
				MessageFile: "missing file",
			},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback:   "Testing: demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						AuthorName: "Testing",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
			},
		},
		"channel file": {
			alert: Alert{
				Type:        "default",
				Channel:     "testchannel",
				ChannelFile: "test_file",
			},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback: ": demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
				Channel: "filecontents",
			},
		},
		"channel file failure": {
			alert: Alert{
				Type:        "default",
				Channel:     "testchannel",
				ChannelFile: "missing file",
			},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback: ": demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1", FooterIcon: ""},
				},
				Channel: "testchannel",
			},
		},
		"text file success": {
			alert: Alert{Type: "default", TextFile: "test_file"},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback: ": demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Text:   "filecontents",
					},
				},
			},
		},
		"text file failure": {
			alert: Alert{Type: "default", Text: "fallback text", TextFile: "missing file"},
			want: &slack.Message{
				Attachments: []slack.Attachment{
					{
						Fallback: ": demo/test/1 -- https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Fields: []slack.Field{
							{Title: "Job", Value: "demo/test", Short: true},
							{Title: "Build", Value: "1", Short: true},
						},
						Footer: "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
						Text:   "fallback text",
					},
				},
			},
		},
	}

	metadata := concourse.BuildMetadata{
		Host:         "https://ci.example.com",
		TeamName:     "main",
		PipelineName: "demo",
		JobName:      "test",
		BuildName:    "1",
		URL:          "https://ci.example.com/teams/main/pipelines/demo/jobs/test/builds/1",
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			path := ""
			if c.alert.MessageFile != "" || c.alert.ChannelFile != "" || c.alert.TextFile != "" {
				path = t.TempDir()

				if err := os.WriteFile(filepath.Join(path, "test_file"), []byte("filecontents"), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got := buildMessage(c.alert, metadata, path)
			if !cmp.Equal(got, c.want) {
				t.Fatalf("unexpected slack.Message value from buildMessage:\n\t(GOT): %#v\n\t(WNT): %#v\n\t(DIFF): %v", got, c.want, cmp.Diff(got, c.want))
			}
		})
	}
}

func TestReadLocalFile(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok"), []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Outside root.
	secret := filepath.Join(base, "secret")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "link-to-file")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(base, filepath.Join(root, "link-to-dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("ok", filepath.Join(root, "link-inside")); err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		name string
		want string
		err  bool
	}{
		"file in root":         {name: "ok", want: "contents"},
		"dot-slash prefix":     {name: "./ok", want: "contents"},
		"parent escape":        {name: "../secret", err: true},
		"nested parent escape": {name: "a/../../secret", err: true},
		"bare parent":          {name: "..", err: true},
		"absolute path":        {name: secret, err: true},
		"symlink to file":      {name: "link-to-file", err: true},
		"symlink via dir":      {name: "link-to-dir/secret", err: true},
		"symlink inside root":  {name: "link-inside", want: "contents"},
		"missing file":         {name: "nope", err: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := readLocalFile(root, c.name)
			if (err != nil) != c.err {
				t.Fatalf("unexpected result from readLocalFile:\n\t(ERR): %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("unexpected contents from readLocalFile:\n\t(GOT): %q\n\t(WNT): %q", got, c.want)
			}
		})
	}
}

func TestParamFromFile(t *testing.T) {
	root := t.TempDir()
	for name, contents := range map[string]string{"ok": " contents\n", "empty": "\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cases := map[string]struct {
		file    string
		want    string
		wantLog string
	}{
		"no file":                       {want: "value"},
		"file overrides value":          {file: "ok", want: "contents"},
		"empty file falls back":         {file: "empty", want: "value"},
		"missing file logs, falls back": {file: "nope", want: "value", wantLog: "will default to message instead"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			old := os.Stderr
			os.Stderr = w
			t.Cleanup(func() { os.Stderr = old })

			got := paramFromFile(root, "message", "value", c.file)
			_ = w.Close()
			logged, _ := io.ReadAll(r)

			if got != c.want {
				t.Fatalf("unexpected value from paramFromFile:\n\t(GOT): %q\n\t(WNT): %q", got, c.want)
			}
			if (c.wantLog == "") != (len(logged) == 0) || !strings.Contains(string(logged), c.wantLog) {
				t.Fatalf("unexpected stderr from paramFromFile:\n\t(GOT): %q\n\t(WNT): contains %q", logged, c.wantLog)
			}
		})
	}
}

func TestPreviousBuildName(t *testing.T) {
	cases := map[string]struct {
		build string
		want  string

		err bool
	}{
		"standard": {
			build: "6",
			want:  "5",
		},
		"two digit rerun 1": {
			build: "10.1",
			want:  "10",
		},
		"rerun 1": {
			build: "6.1",
			want:  "6",
		},
		"rerun x": {
			build: "6.2",
			want:  "6.1",
		},
		"error 1": {
			build: "X",
			err:   true,
		},
		"error x": {
			build: "6.X",
			err:   true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := previousBuildName(c.build)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from previousBuildName:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from previousBuildName:\n\t(GOT): nil")
			} else if err != nil {
				return
			}

			if got != c.want {
				t.Fatalf("unexpected value from previousBuildName:\n\t(GOT): %#v\n\t(WNT): %#v", got, c.want)
			}
		})
	}
}

func TestPreviousBuildStatus(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/teams/main/pipelines/demo/jobs/test/builds/1":
			_ = json.NewEncoder(w).Encode(concourse.Build{Name: "1", Status: "failed"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()

	cases := map[string]struct {
		build string
		host  string
		want  string
		err   bool
	}{
		"first build":      {build: "1", host: s.URL},
		"previous build":   {build: "2", host: s.URL, want: "failed"},
		"unknown previous": {build: "5", host: s.URL, err: true},
		"bad build name":   {build: "X", host: s.URL, err: true},
		"bad host":         {build: "2", host: "://", err: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := concourse.BuildMetadata{Host: c.host, TeamName: "main", PipelineName: "demo", JobName: "test", BuildName: c.build}
			got, err := previousBuildStatus(&concourse.OutRequest{}, m)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from previousBuildStatus:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from previousBuildStatus:\n\t(GOT): nil")
			} else if got != c.want {
				t.Fatalf("unexpected status from previousBuildStatus:\n\t(GOT): %#v\n\t(WNT): %#v", got, c.want)
			}
		})
	}
}

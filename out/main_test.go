package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
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
		"default configuration": {
			outRequest: &concourse.OutRequest{
				Source: concourse.Source{
					URL: ok.URL,
				},
			},
			want: &concourse.OutResponse{
				Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
				Metadata: []concourse.Metadata{
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
					concourse.Metadata{Name: "type", Value: "default"},
					concourse.Metadata{Name: "message", Value: "Deploying"},
					concourse.Metadata{Name: "color", Value: "#ffffff"},
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
					URL: bad.URL,
				},
				Params: concourse.OutParams{
					AlertType: "fixed",
				},
			},
			err: true,
		},
	}

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

package concourse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewClient(t *testing.T) {
	var testToken = token{
		Type:  "Bearer",
		Value: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJjc3JmIjoiMDM3YWQ5Zjk5OGMyYmViZDg1MGMzN2NkOTkwMGE2YjdmOTZjOTkwYzk4ZDk3YWQyNjliYTU2N2IyMTI5MjJkZCIsImV4cCI6MTUzMDc0NzA5NiwiaXNBZG1pbiI6dHJ1ZSwidGVhbU5hbWUiOiJtYWluIn0.YL11IVWLlkx5xV_aX7DT0f_Y_FIA8NS5jOndZFw8paJBUVv-_brvTAD9hn4t6FPw0o2gsub1jHB2E4l-VB954_iQ-SfWyzDx0idegYrLlcgsXHhPWku1hW1JBjvq3BhjkKgAuZCW4JP5UulglObaKKFFhYycMZiiiWcKM_zMpn7ebgP6giEemSRj06Bpc5EKAZeZjt0Tv3AqEKd693qI9XJp49LJwZJP_RZgCoXMduLQpm3UmIQppwzFUEyIcAfXJmYvi3utr_JjxfuVuwqZbsemf_fCxRGgkUcwRBwBnlRqvBUdErk63HYAL7t4pdk9mzb61U5OPK9XT8NS195IHw",
	}

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(testToken)
		w.Write(b)
	}))
	defer ok.Close()

	unauthoized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthoized.Close()

	cases := map[string]struct {
		username string
		password string
		host     string
		team     string
		want     *Client
		err      bool
	}{
		"public pipeline": {
			host: ok.URL,
			team: "main",
			want: &Client{
				host: fmt.Sprintf("%s/api/v1/", ok.URL),
				team: "main",
			},
		},
		"private pipeline": {
			username: "admin",
			password: "sup3rs3cret1",
			host:     ok.URL,
			team:     "main",
			want: &Client{
				host: fmt.Sprintf("%s/api/v1/", ok.URL),
				team: "main",
				auth: testToken,
			},
		},
		"unauthorized": {
			username: "admin",
			password: "sup3rs3cret1",
			host:     unauthoized.URL,
			team:     "main",
			err:      true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client, err := NewClient(c.username, c.password, c.host, c.team)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from NewClient:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from NewClient:\n\t(GOT): nil")
			} else if !reflect.DeepEqual(client, c.want) {
				t.Fatalf("unexpected Client from NewClient:\n\t(GOT): %#v\n\t(WNT): %#v", client, c.want)
			}
		})
	}
}

func TestGetBuild(t *testing.T) {
	build := &Build{
		ID:       1,
		Team:     "main",
		Name:     "1",
		Status:   "succeeded",
		Job:      "test",
		APIURL:   "/api/v1/builds/1",
		Pipeline: "demo",
	}

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(build)
		w.Write(b)
	}))
	defer ok.Close()

	unauthoized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthoized.Close()

	cases := map[string]struct {
		client   *Client
		pipeline string
		job      string
		name     string
		want     *Build
		err      bool
	}{
		"public build": {
			client: &Client{
				host: ok.URL + "/",
				team: "main",
			},
			pipeline: "demo",
			job:      "test",
			name:     "1",
			want:     build,
		},
		"unauthorized": {
			client: &Client{
				host: unauthoized.URL + "/",
				team: "main",
			},
			pipeline: "demo",
			job:      "test",
			name:     "1",
			err:      true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			build, err := c.client.GetBuild(c.pipeline, c.job, c.name)
			if err != nil && !c.err {
				t.Fatalf("unexpected error from GetBuild:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from GetBuild:\n\t(GOT): nil")
			} else if !reflect.DeepEqual(build, c.want) {
				t.Fatalf("unexpected Build from GetBuild:\n\t(GOT): %#v\n\t(WNT): %#v", build, c.want)
			}
		})
	}
}

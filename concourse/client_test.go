package concourse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

const tokenType = "Bearer"
const accessToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJjc3JmIjoiMDM3YWQ5Zjk5OGMyYmViZDg1MGMzN2NkOTkwMGE2YjdmOTZjOTkwYzk4ZDk3YWQyNjliYTU2N2IyMTI5MjJkZCIsImV4cCI6MTUzMDc0NzA5NiwiaXNBZG1pbiI6dHJ1ZSwidGVhbU5hbWUiOiJtYWluIn0.YL11IVWLlkx5xV_aX7DT0f_Y_FIA8NS5jOndZFw8paJBUVv-_brvTAD9hn4t6FPw0o2gsub1jHB2E4l-VB954_iQ-SfWyzDx0idegYrLlcgsXHhPWku1hW1JBjvq3BhjkKgAuZCW4JP5UulglObaKKFFhYycMZiiiWcKM_zMpn7ebgP6giEemSRj06Bpc5EKAZeZjt0Tv3AqEKd693qI9XJp49LJwZJP_RZgCoXMduLQpm3UmIQppwzFUEyIcAfXJmYvi3utr_JjxfuVuwqZbsemf_fCxRGgkUcwRBwBnlRqvBUdErk63HYAL7t4pdk9mzb61U5OPK9XT8NS195IHw"

func TestNewClient(t *testing.T) {
	cases := map[string]struct {
		username string
		password string

		public bool
		legacy bool
		err    bool
	}{
		"public": {
			public: true,
		},
		"legacy": {
			username: "admin",
			password: "sup3rs3cret1",

			legacy: true,
		},
		"skymarshal": {
			username: "admin",
			password: "sup3rs3cret1",
		},
		"unauthorized": {
			username: "admin",
			password: "sup3rs3cret1",

			err: true,
		},
	}

	for name, c := range cases {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.err {
				http.Error(w, "", http.StatusUnauthorized)
			}

			var version string
			if c.legacy {
				version = "3.14.2"
			} else {
				version = "4.0.0"
			}

			var resp []byte
			switch r.RequestURI {
			case "/api/v1/info":
				resp, _ = json.Marshal(Info{ATCVersion: version})
			case "/sky/token":
				resp, _ = json.Marshal(oauth2.Token{TokenType: tokenType, AccessToken: accessToken})
			default:
				resp, _ = json.Marshal(Token{Type: tokenType, Value: accessToken})
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}))

		t.Run(name, func(t *testing.T) {
			client, err := NewClient(s.URL, "main", c.username, c.password)
			// Test err conditions.
			if err != nil && !c.err {
				t.Fatalf("unexpected error from NewClient:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from NewClient:\n\t(GOT): nil")
			} else if err != nil && c.err {
				return
			}

			// Test client conditions (if no errors occurred).
			if client.atcurl.String() != s.URL {
				t.Fatalf("unexpected Client.atcurl from NewClient:\n\t(GOT): %#v\n\t(WNT): %#v", client.atcurl, s.URL)
			} else if client.team != "main" {
				t.Fatalf("unexpected Client.atcurl from NewClient:\n\t(GOT): %#v\n\t(WNT): %#v", client.team, "main")
			} else if c.public {
				return
			}

			// Test client cookie conditions (if pipeline is not public).
			cv := client.conn.Jar.Cookies(client.atcurl)[0].Value
			wnt := strings.Join([]string{tokenType, accessToken}, " ")
			if cv != wnt {
				t.Fatalf("unexpected Client.conn cookie from NewClient:\n\t(GOT): %#v\n\t(WNT): %#v", cv, wnt)
			}
		})
		s.Close()
	}
}

func TestJobBuild(t *testing.T) {
	cases := map[string]struct {
		build *Build
		err   bool
	}{
		"ok": {build: &Build{
			ID:       1,
			Team:     "main",
			Name:     "1",
			Status:   "succeeded",
			Job:      "test",
			APIURL:   "/api/v1/builds/1",
			Pipeline: "demo",
		}},
		"unauthorized": {
			build: &Build{},
			err:   true,
		},
	}

	for name, c := range cases {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.err {
				http.Error(w, "", http.StatusUnauthorized)
			}
			resp, _ := json.Marshal(c.build)
			w.Write(resp)
		}))
		u, _ := url.Parse(s.URL)

		t.Run(name, func(t *testing.T) {
			client := &Client{atcurl: u, team: c.build.Team, conn: &http.Client{}}
			build, err := client.JobBuild(c.build.Pipeline, c.build.Job, c.build.Name)

			if err != nil && !c.err {
				t.Fatalf("unexpected error from JobBuild:\n\t(ERR): %s", err)
			} else if err == nil && c.err {
				t.Fatalf("expected an error from JobBuild:\n\t(GOT): nil")
			} else if !c.err && !reflect.DeepEqual(build, c.build) {
				t.Fatalf("unexpected Build from JobBuild:\n\t(GOT): %#v\n\t(WNT): %#v", build, c.build)
			}
		})
		s.Close()
	}
}

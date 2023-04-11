package concourse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

const tokenType = "Bearer"

func TestNewClient(t *testing.T) {
	cases := map[string]struct {
		version  string
		public   bool
		username string
		password string

		token   string
		idToken string
		err     bool
	}{
		"public": {
			version: "6.5.0",
			public:  true,
		},
		"legacy auth": {
			version:  "3.14.2",
			username: "admin",
			password: "sup3rs3cret1",

			token: "legacy",
		},
		"legacy skymarshal": {
			version:  "4.0.0",
			username: "admin",
			password: "sup3rs3cret1",

			token: "access-token",
		},
		"multi cookie": {
			version:  "6.0.0",
			username: "admin",
			password: "sup3rs3cret1",

			token: "multi-cookie",
		},
		"skymarshal id token": {
			version:  "6.1.0",
			username: "admin",
			password: "sup3rs3cret1",

			token:   "new-access-token",
			idToken: "id-token",
		},
		"skymarshal access token": {
			version:  "6.5.0",
			username: "admin",
			password: "sup3rs3cret1",

			token: "new-access-token",
		},
		"missing id token": {
			version:  "6.1.0",
			username: "admin",
			password: "sup3rs3cret1",

			token: "new-access-token",
			err:   true,
		},
		"unauthorized": {
			version:  "6.5.0",
			username: "admin",
			password: "sup3rs3cret1",

			err: true,
		},
	}

	for name, c := range cases {
		info := Info{ATCVersion: c.version}
		legacy := Token{Type: tokenType, Value: c.token}
		oldsky := map[string]string{"token_type": tokenType, "access_token": c.token}
		sky := map[string]string{"token_type": tokenType, "access_token": c.token, "id_token": c.idToken}
		if c.idToken == "" {
			sky = oldsky
		}

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var resp []byte
			switch r.RequestURI {
			case "/api/v1/info":
				resp, _ = json.Marshal(info)
			case "/api/v1/teams/main/auth/token":
				resp, _ = json.Marshal(legacy)
			case "/sky/token":
				resp, _ = json.Marshal(oldsky)
			case "/sky/issuer/token":
				resp, _ = json.Marshal(sky)
			default:
				http.Error(w, "", http.StatusUnauthorized)
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
			wnt := strings.Join([]string{tokenType, c.token}, " ")
			if c.idToken != "" {
				wnt = strings.Join([]string{tokenType, c.idToken}, " ")
			}
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
		"okWithoutInstanceVars": {build: &Build{
			ID:       1,
			Team:     "main",
			Name:     "1",
			Status:   "succeeded",
			Job:      "test",
			APIURL:   "/api/v1/builds/1",
			Pipeline: "demo",
		}},
		"okWithInstanceVars": {build: &Build{
			ID:       1,
			Team:     "main",
			Name:     "1",
			Status:   "succeeded",
			Job:      "test",
			APIURL:   "/api/v1/builds/1",
			Pipeline: "demo",
			InstanceVars: map[string]interface{}{
				"image_name": "my-image",
				"pr_number":  "1234",
			},
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
			instanceVars := ""
			if c.build.InstanceVars != nil {
				query := &url.Values{}
				for key, value := range c.build.InstanceVars {
					key = fmt.Sprintf("vars.%s", key)

					switch val := value.(type) {
					case int:
						query.Set(key, fmt.Sprintf(`%v`, val))
					case float64:
						query.Set(key, fmt.Sprintf(`%v`, val))
					case string:
						query.Set(key, fmt.Sprintf(`"%v"`, val))
					default:
						panic(fmt.Sprintf("unexpected type for instance var %v: %v", key, reflect.TypeOf(val)))
					}
				}
				instanceVars = fmt.Sprintf("?%s", query.Encode())
			}
			build, err := client.JobBuild(c.build.Pipeline, c.build.Job, c.build.Name, instanceVars)

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

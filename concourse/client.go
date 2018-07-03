package concourse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	apiHost        = "%s://%s/api/v1/"
	apiAuth        = "teams/%s/auth/token"
	apiBuildStatus = "teams/%s/pipelines/%s/jobs/%s/builds/%s"
)

// Client represents a Concourse client
type Client struct {
	host string
	team string

	auth token
}

type token struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewClient logs into Concourse with Basic Auth and returns a client struct
func NewClient(username, password, host, team string) (*Client, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	c := &Client{
		host: fmt.Sprintf(apiHost, u.Scheme, u.Host),
		team: team,
	}

	// Skip auth if no username and password provided
	if username == "" && password == "" {
		return c, nil
	}

	client := http.Client{}
	authURL := c.host + fmt.Sprintf(apiAuth, c.team)
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)

	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("Could not log into Concourse: status code %d", r.StatusCode)
	}

	json.NewDecoder(r.Body).Decode(&c.auth)
	return c, nil
}

// GetBuild returns a Build
func (c *Client) GetBuild(pipeline, job, name string) (*Build, error) {
	client := http.Client{}
	statusURL := c.host + fmt.Sprintf(apiBuildStatus, c.team, pipeline, job, name)
	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return nil, err
	}

	if c.auth.Type != "" {
		cookie := &http.Cookie{
			Name:  "ATC-Authorization",
			Value: fmt.Sprintf("%s %s", c.auth.Type, c.auth.Value),
		}
		req.AddCookie(cookie)
	}

	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", r.StatusCode)
	}

	var build *Build
	json.NewDecoder(r.Body).Decode(&build)
	return build, nil
}

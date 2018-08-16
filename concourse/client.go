package concourse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// A Client is a Concourse API connection.
type Client struct {
	apiurl string
	team   string
	auth   token
}

// A token is used by Concourse for auth.
type token struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewClient returns an authorized Client (if private) for the Concourse API.
func NewClient(host, team, username, password string) (*Client, error) {
	c := &Client{
		apiurl: fmt.Sprintf("%s/api/v1", strings.TrimSuffix(host, "/")),
		team:   team,
	}

	// Return Client early if authorization is not needed.
	if username == "" && password == "" {
		return c, nil
	}

	err := c.loginLegacy(username, password)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) loginLegacy(username, password string) error {
	url := fmt.Sprintf("%s/teams/%s/auth/token", c.apiurl, c.team)

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	r, err := client.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("Could not log into Concourse: status code %d", r.StatusCode)
	}

	json.NewDecoder(r.Body).Decode(&c.auth)
	return nil
}

// GetBuild finds and returns a Build from the Concourse API  provided
// pipeline, job and build name.
func (c *Client) GetBuild(pipeline, job, name string) (*Build, error) {
	url := fmt.Sprintf(
		"%s/teams/%s/pipelines/%s/jobs/%s/builds/%s",
		c.apiurl,
		c.team,
		pipeline,
		job,
		name,
	)

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
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

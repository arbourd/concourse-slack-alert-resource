package concourse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/oauth2"
)

// requestTimeout bounds each Concourse API request.
const requestTimeout = 30 * time.Second

// A Client is a Concourse API connection.
type Client struct {
	atcurl *url.URL
	team   string

	conn *http.Client
}

// Info is version information from the Concourse API.
type Info struct {
	ATCVersion    string `json:"version"`
	WorkerVersion string `json:"worker_version"`
}

// A Token is a legacy Concourse access token.
type Token struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewClient returns an authorized Client (if private) for the Concourse API.
func NewClient(atcurl, team, username, password string) (*Client, error) {
	u, err := url.Parse(atcurl)
	if err != nil {
		return nil, err
	}
	// This cookie jar implementation never returns an error.
	jar, _ := cookiejar.New(nil)

	c := &Client{
		atcurl: u,
		team:   team,

		conn: &http.Client{Jar: jar, Timeout: requestTimeout},
	}

	// Return Client early if authorization is not needed.
	if username == "" && password == "" {
		return c, nil
	}

	info, err := c.info()
	if err != nil {
		return nil, err
	}

	legacy, err := semver.NewConstraint("< 4.0.0")
	if err != nil {
		return nil, err
	}

	v, err := semver.NewVersion(info.ATCVersion)
	if err != nil {
		return nil, err
	}

	multiCookie, err := semver.NewConstraint("5.5 - 6.4")
	if err != nil {
		return nil, err
	}

	oldsky, err := semver.NewConstraint("< 6.1.0")
	if err != nil {
		return nil, err
	}

	// Check if target Concourse is less than '4.0.0'.
	if legacy.Check(v) {
		tokenURL := fmt.Sprintf("%s/api/v1/teams/%s/auth/token", c.atcurl, c.team)
		err = c.loginLegacy(tokenURL, username, password)
		return c, err
	}

	tokenURL := fmt.Sprintf("%s/sky/issuer/token", c.atcurl)
	// Support 4.x and 5.x skymarshall calls
	if oldsky.Check(v) {
		tokenURL = fmt.Sprintf("%s/sky/token", c.atcurl)
	}

	token, err := c.login(tokenURL, username, password)
	if err != nil {
		return nil, err
	}

	// Check if the version supports single cookie access tokens.
	// Single cookie is used between versions 4.0.0 - 5.5.0 and 6.5.0 or greater.
	if !multiCookie.Check(v) {
		c.singleCookie(token.TokenType, token.AccessToken)
		return c, nil
	}

	// Check if the version is less than '6.1.0'.
	if oldsky.Check(v) {
		err = c.splitToken(token.TokenType, token.AccessToken)
		return c, err
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return c, errors.New("invalid id_token")
	}

	err = c.splitToken(token.TokenType, idToken)
	return c, err
}

// info queries Concourse for its version information.
func (c *Client) info() (Info, error) {
	u := fmt.Sprintf("%s/api/v1/info", c.atcurl)
	var info Info

	r, err := c.conn.Get(u)
	if err != nil {
		return info, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return info, fmt.Errorf("could not get info from Concourse: status code %d", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(&info)

	return info, err
}

// singleCookie add the token as a single cookie.
func (c *Client) singleCookie(tokenType, tokenValue string) {
	c.conn.Jar.SetCookies(
		c.atcurl,
		[]*http.Cookie{{
			Name:  "skymarshal_auth",
			Value: fmt.Sprintf("%s %s", tokenType, tokenValue),
		}},
	)
}

// splitToken splits the token across multiple cookies.
func (c *Client) splitToken(tokenType, tokenValue string) error {
	const numCookies = 15
	const authCookieName = "skymarshal_auth"
	const maxCookieSize = 4000

	tokenStr := fmt.Sprintf("%s %s", tokenType, tokenValue)

	for i := range numCookies {
		chunk := tokenStr
		if len(chunk) > maxCookieSize {
			chunk = chunk[:maxCookieSize]
		}
		c.conn.Jar.SetCookies(
			c.atcurl,
			[]*http.Cookie{{
				Name:  authCookieName + strconv.Itoa(i),
				Value: chunk,
			}},
		)
		tokenStr = tokenStr[len(chunk):]
		if tokenStr == "" {
			return nil
		}
	}

	return errors.New("token too large to split across cookies")
}

// login gets an access token from Concourse.
func (c *Client) login(tokenURL, username, password string) (*oauth2.Token, error) {
	config := oauth2.Config{
		ClientID:     "fly",
		ClientSecret: "Zmx5",
		Endpoint:     oauth2.Endpoint{TokenURL: tokenURL},
		Scopes:       []string{"openid", "profile", "email", "federated:id", "groups"},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.conn)
	return config.PasswordCredentialsToken(ctx, username, password)
}

// loginLegacy gets a legacy access token from Concourse.
func (c *Client) loginLegacy(tokenURL, username, password string) error {
	req, err := http.NewRequest(http.MethodGet, tokenURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	r, err := c.conn.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("could not log into Concourse: status code %d", r.StatusCode)
	}

	var t Token
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		return err
	}

	c.singleCookie(t.Type, t.Value)
	return nil
}

// JobBuild finds and returns a Build from the Concourse API by its
// pipeline name, job name and build name.
func (c *Client) JobBuild(pipeline, job, name, instanceVars string) (*Build, error) {
	u := fmt.Sprintf(
		"%s/api/v1/teams/%s/pipelines/%s/jobs/%s/builds/%s%s",
		c.atcurl,
		url.PathEscape(c.team),
		url.PathEscape(pipeline),
		url.PathEscape(job),
		url.PathEscape(name),
		instanceVars,
	)

	r, err := c.conn.Get(u)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", r.StatusCode)
	}

	var build *Build
	if err := json.NewDecoder(r.Body).Decode(&build); err != nil {
		return nil, err
	}
	if build == nil {
		return nil, errors.New("empty build response")
	}
	return build, nil
}

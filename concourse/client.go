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

	"github.com/Masterminds/semver/v3"
	"golang.org/x/oauth2"
)

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

		conn: &http.Client{Jar: jar},
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
		url := fmt.Sprintf("%s/api/v1/teams/%s/auth/token", c.atcurl, c.team)
		err = c.loginLegacy(url, username, password)
		return c, err
	}

	url := fmt.Sprintf("%s/sky/issuer/token", c.atcurl)
	// Support 4.x and 5.x skymarshall calls
	if oldsky.Check(v) {
		url = fmt.Sprintf("%s/sky/token", c.atcurl)
	}

	token, err := c.login(url, username, password)
	if err != nil {
		return nil, err
	}

	// Check if the version supports single cookie access tokens.
	// Single cookie is used between versions 4.0.0 - 5.5.0 and 6.5.0 or greater.
	if !multiCookie.Check(v) {
		err = c.singleCookie(token.TokenType, token.AccessToken)
		return c, err
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
	if r.StatusCode != 200 {
		return info, fmt.Errorf("could not get info from Concourse: status code %d", r.StatusCode)
	}
	json.NewDecoder(r.Body).Decode(&info)

	return info, nil
}

// singleCookie add the token as a single cookie.
func (c *Client) singleCookie(tokenType, tokenValue string) error {
	c.conn.Jar.SetCookies(
		c.atcurl,
		[]*http.Cookie{{
			Name:  "skymarshal_auth",
			Value: fmt.Sprintf("%s %s", tokenType, tokenValue),
		}},
	)
	return nil
}

// splitToken splits the token across multiple cookies.
func (c *Client) splitToken(tokenType, tokenValue string) error {
	const NumCookies = 15
	const authCookieName = "skymarshal_auth"
	const maxCookieSize = 4000

	tokenStr := fmt.Sprintf("%s %s", tokenType, tokenValue)

	for i := 0; i < NumCookies; i++ {
		if len(tokenStr) > maxCookieSize {
			c.conn.Jar.SetCookies(
				c.atcurl,
				[]*http.Cookie{{
					Name:  authCookieName + strconv.Itoa(i),
					Value: tokenStr[:maxCookieSize],
				}},
			)
			tokenStr = tokenStr[maxCookieSize:]
		} else {
		}
		c.conn.Jar.SetCookies(
			c.atcurl,
			[]*http.Cookie{{
				Name:  authCookieName + strconv.Itoa(i),
				Value: tokenStr,
			}},
		)
		break
	}

	return nil
}

// login gets an access token from Concourse.
func (c *Client) login(url, username, password string) (*oauth2.Token, error) {
	config := oauth2.Config{
		ClientID:     "fly",
		ClientSecret: "Zmx5",
		Endpoint:     oauth2.Endpoint{TokenURL: url},
		Scopes:       []string{"openid", "profile", "email", "federated:id", "groups"},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.conn)
	t, err := config.PasswordCredentialsToken(ctx, username, password)
	return t, err
}

// loginLegacy gets a legacy access token from Concourse.
func (c *Client) loginLegacy(url, username, password string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	r, err := c.conn.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("could not log into Concourse: status code %d", r.StatusCode)
	}

	var t Token
	json.NewDecoder(r.Body).Decode(&t)

	c.conn.Jar.SetCookies(
		c.atcurl,
		[]*http.Cookie{{
			Name:  "skymarshal_auth",
			Value: fmt.Sprintf("%s %s", t.Type, t.Value),
		}},
	)
	return nil
}

// JobBuild finds and returns a Build from the Concourse API by its
// pipeline name, job name and build name.
func (c *Client) JobBuild(pipeline, job, name, instanceVars string) (*Build, error) {
	u := fmt.Sprintf(
		"%s/api/v1/teams/%s/pipelines/%s/jobs/%s/builds/%s%s",
		c.atcurl,
		c.team,
		pipeline,
		job,
		name,
		instanceVars,
	)

	r, err := c.conn.Get(u)
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

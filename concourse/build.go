package concourse

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// A Build is a build's data from the undocumented Concourse API.
type Build struct {
	ID        int    `json:"id"`
	Team      string `json:"team_name"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Job       string `json:"job_name"`
	APIURL    string `json:"api_url"`
	Pipeline  string `json:"pipeline_name"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
}

// BuildMetadata is the current build's metadata exposed via the environment.
// https://concourse-ci.org/implementing-resources.html#resource-metadata
type BuildMetadata struct {
	Host         string
	ID           string
	TeamName     string
	PipelineName string
	JobName      string
	BuildName    string
	URL          string
}

// NewBuildMetadata returns a populated BuildMetadata.
// The default external URL can be overriden by the URL.
func NewBuildMetadata(atcurl string) BuildMetadata {
	if atcurl == "" {
		atcurl = os.Getenv("ATC_EXTERNAL_URL")
	}

	metadata := BuildMetadata{
		Host:         strings.TrimSuffix(atcurl, "/"),
		ID:           os.Getenv("BUILD_ID"),
		TeamName:     os.Getenv("BUILD_TEAM_NAME"),
		PipelineName: os.Getenv("BUILD_PIPELINE_NAME"),
		JobName:      os.Getenv("BUILD_JOB_NAME"),
		BuildName:    os.Getenv("BUILD_NAME"),
	}

	// "$HOST/teams/$BUILD_TEAM_NAME/pipelines/$BUILD_PIPELINE_NAME/jobs/$BUILD_JOB_NAME/builds/$BUILD_NAME"
	metadata.URL = fmt.Sprintf(
		"%s/teams/%s/pipelines/%s/jobs/%s/builds/%s",
		metadata.Host,
		url.PathEscape(metadata.TeamName),
		url.PathEscape(metadata.PipelineName),
		url.PathEscape(metadata.JobName),
		url.PathEscape(metadata.BuildName),
	)

	return metadata
}

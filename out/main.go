package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
	"github.com/arbourd/concourse-slack-alert-resource/slack"
)

// Alert defines the configuration of the Slack alert
type Alert struct {
	Type    string
	Channel string
	Color   string
	IconURL string
	Message string
}

func main() {
	var input *concourse.OutRequest
	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		log.Fatalln(err)
	}

	o, err := out(input)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(o)
	if err != nil {
		log.Fatalln(err)
	}
}

func out(input *concourse.OutRequest) (*concourse.OutResponse, error) {
	if input.Source.URL == "" {
		return nil, errors.New("slack webhook url cannot be blank")
	}

	metadata := &concourse.BuildMetadata{
		URL:          input.Source.ConcourseURL,
		TeamName:     os.Getenv("BUILD_TEAM_NAME"),
		PipelineName: os.Getenv("BUILD_PIPELINE_NAME"),
		JobName:      os.Getenv("BUILD_JOB_NAME"),
		BuildName:    os.Getenv("BUILD_NAME"),
	}
	if metadata.URL == "" {
		metadata.URL = os.Getenv("ATC_EXTERNAL_URL")
	}

	var alert *Alert
	switch input.Params.AlertType {
	case "success":
		alert = &Alert{
			Type:    "success",
			Color:   "#32cd32",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-succeeded.png",
			Message: "Success",
		}
	case "failed":
		alert = &Alert{
			Type:    "failed",
			Color:   "#d00000",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-failed.png",
			Message: "Failed",
		}
	case "started":
		alert = &Alert{
			Type:    "started",
			Color:   "#f7cd42",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-started.png",
			Message: "Started",
		}
	case "aborted":
		alert = &Alert{
			Type:    "aborted",
			Color:   "#8d4b32",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-aborted.png",
			Message: "Aborted",
		}
	case "fixed":
		alert = &Alert{
			Type:    "fixed",
			Color:   "#32cd32",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-succeeded.png",
			Message: "Fixed",
		}
	case "broke":
		alert = &Alert{
			Type:    "broke",
			Color:   "#d00000",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-failed.png",
			Message: "Broke",
		}
	default:
		alert = &Alert{
			Type:    "default",
			Color:   "#35495c",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-pending.png",
			Message: "",
		}
	}

	alert.Channel = input.Params.Channel
	if alert.Channel == "" {
		alert.Channel = input.Source.Channel
	}
	if input.Params.Message != "" {
		alert.Message = input.Params.Message
	}
	if input.Params.Color != "" {
		alert.Color = input.Params.Color
	}
	var send = !input.Params.Disable

	if send && (alert.Type == "fixed" || alert.Type == "broke") {
		status, err := previousBuildStatus(input, metadata)
		if err != nil {
			return nil, err
		}
		send = (alert.Type == "fixed" && status != "succeeded") || (alert.Type == "broke" && status == "succeeded")
	}

	if send {
		message := buildMessage(alert, metadata)
		err := slack.Send(input.Source.URL, message)
		if err != nil {
			return nil, err
		}
	}

	out := &concourse.OutResponse{
		Version: concourse.Version{"timestamp": time.Now().UTC().Format("201806200430")},
		Metadata: []concourse.Metadata{
			concourse.Metadata{Name: "type", Value: alert.Type},
			concourse.Metadata{Name: "channel", Value: alert.Channel},
			concourse.Metadata{Name: "alerted", Value: strconv.FormatBool(send)},
		},
	}
	return out, nil
}

func previousBuildStatus(input *concourse.OutRequest, meta *concourse.BuildMetadata) (string, error) {
	// Exit early if first build
	if meta.BuildName == "1" {
		return "", nil
	}

	c, err := concourse.NewClient(input.Source.Username, input.Source.Password, meta.URL, meta.TeamName)
	if err != nil {
		return "", fmt.Errorf("error connecting to Concourse: %s", err)
	}

	no, err := strconv.Atoi(meta.BuildName)
	if err != nil {
		return "", err
	}

	previous, err := c.GetBuild(meta.PipelineName, meta.JobName, strconv.Itoa(no-1))
	if err != nil {
		return "", fmt.Errorf("error requesting Concourse build status: %s", err)
	}

	return previous.Status, nil
}

const (
	// "$ATC_EXTERNAL_URL/teams/$BUILD_TEAM_NAME/pipelines/$BUILD_PIPELINE_NAME/jobs/$BUILD_JOB_NAME/builds/$BUILD_NAME"
	buildURLTemplate = "%s/teams/%s/pipelines/%s/jobs/%s/builds/%s"

	// "$ALERT_MESSAGE: $BUILD_PIPELINE_NAME/$BUILD_JOB_NAME/$BUILD_NAME -- $BUILD_URL"
	fallbackTemplate = "%s: %s/%s/%s"
)

func buildMessage(alert *Alert, m *concourse.BuildMetadata) *slack.Message {
	buildURL := fmt.Sprintf(buildURLTemplate, m.URL, m.TeamName, m.PipelineName, m.JobName, m.BuildName)
	attachment := slack.Attachment{
		Fallback:   fmt.Sprintf("%s -- %s", fmt.Sprintf(fallbackTemplate, alert.Message, m.PipelineName, m.JobName, m.BuildName), buildURL),
		AuthorName: alert.Message,
		Color:      alert.Color,
		Footer:     buildURL,
		FooterIcon: alert.IconURL,
		Fields: []slack.Field{
			slack.Field{
				Title: "Job",
				Value: fmt.Sprintf("%s/%s", m.PipelineName, m.JobName),
				Short: true,
			},
			slack.Field{
				Title: "Build",
				Value: m.BuildName,
				Short: true,
			},
		},
	}

	return &slack.Message{Attachments: []slack.Attachment{attachment}, Channel: alert.Channel}
}

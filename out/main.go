package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
	"github.com/arbourd/concourse-slack-alert-resource/slack"
)

func buildMessage(alert Alert, m concourse.BuildMetadata) *slack.Message {
	fallback := fmt.Sprintf("%s -- %s", fmt.Sprintf("%s: %s/%s/%s", alert.Message, m.PipelineName, m.JobName, m.BuildName), m.URL)
	attachment := slack.Attachment{
		Fallback:   fallback,
		AuthorName: alert.Message,
		Color:      alert.Color,
		Footer:     m.URL,
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

func previousBuildStatus(input *concourse.OutRequest, m concourse.BuildMetadata) (string, error) {
	// Exit early if first build
	if m.BuildName == "1" {
		return "", nil
	}

	c, err := concourse.NewClient(m.Host, m.TeamName, input.Source.Username, input.Source.Password)
	if err != nil {
		return "", fmt.Errorf("error connecting to Concourse: %s", err)
	}

	no, err := strconv.Atoi(m.BuildName)
	if err != nil {
		return "", err
	}

	previous, err := c.GetBuild(m.PipelineName, m.JobName, strconv.Itoa(no-1))
	if err != nil {
		return "", fmt.Errorf("error requesting Concourse build status: %s", err)
	}

	return previous.Status, nil
}

func out(input *concourse.OutRequest) (*concourse.OutResponse, error) {
	if input.Source.URL == "" {
		return nil, errors.New("slack webhook url cannot be blank")
	}

	alert := NewAlert(input)
	metadata := concourse.NewBuildMetadata(input.Source.ConcourseURL)
	send := !alert.Disabled

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
		Version: concourse.Version{"ver": "static"},
		Metadata: []concourse.Metadata{
			concourse.Metadata{Name: "type", Value: alert.Type},
			concourse.Metadata{Name: "channel", Value: alert.Channel},
			concourse.Metadata{Name: "alerted", Value: strconv.FormatBool(send)},
		},
	}
	return out, nil
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

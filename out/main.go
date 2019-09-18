package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
	"github.com/arbourd/concourse-slack-alert-resource/slack"
)

func buildMessage(alert Alert, m concourse.BuildMetadata, path string) *slack.Message {
	message := alert.Message

	// Open and read message file if set
	if alert.MessageFile != "" {
		file := filepath.Join(path, alert.MessageFile)
		f, err := ioutil.ReadFile(file)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message_file: %v\nwill default to message instead\n", err)
		} else {
			message = strings.TrimSpace(string(f))
		}
	}

	attachment := slack.Attachment{
		Fallback:   fmt.Sprintf("%s -- %s", fmt.Sprintf("%s: %s/%s/%s", message, m.PipelineName, m.JobName, m.BuildName), m.URL),
		AuthorName: message,
		Color:      alert.Color,
		Footer:     m.URL,
		FooterIcon: alert.IconURL,
		Fields: []slack.Field{
			{
				Title: "Job",
				Value: fmt.Sprintf("%s/%s", m.PipelineName, m.JobName),
				Short: true,
			},
			{
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

	previous, err := c.JobBuild(m.PipelineName, m.JobName, strconv.Itoa(no-1))
	if err != nil {
		return "", fmt.Errorf("error requesting Concourse build status: %s", err)
	}

	return previous.Status, nil
}

func out(input *concourse.OutRequest, path string) (*concourse.OutResponse, error) {
	if input.Source.URL == "" {
		return nil, errors.New("slack webhook url cannot be blank")
	}

	alert := NewAlert(input)
	metadata := concourse.NewBuildMetadata(input.Source.ConcourseURL)
	send := !alert.Disabled

	if send && (alert.Type == "fixed" || alert.Type == "broke") {
		status, err := previousBuildStatus(input, metadata)
		if err != nil {
			return nil, fmt.Errorf("error getting last build status: %v", err)
		}
		send = (alert.Type == "fixed" && status != "succeeded") || (alert.Type == "broke" && status == "succeeded")
	}

	if send {
		message := buildMessage(alert, metadata, path)
		err := slack.Send(input.Source.URL, message)
		if err != nil {
			return nil, fmt.Errorf("error sending slack message: %v", err)
		}
	}

	out := &concourse.OutResponse{
		Version: concourse.Version{"ver": "static"},
		Metadata: []concourse.Metadata{
			{Name: "type", Value: alert.Type},
			{Name: "channel", Value: alert.Channel},
			{Name: "alerted", Value: strconv.FormatBool(send)},
		},
	}
	return out, nil
}

func main() {
	// The first argument is the path to the build's sources.
	path := os.Args[1]

	var input *concourse.OutRequest
	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		log.Fatalln(fmt.Errorf("error reading stdin: %v", err))
	}

	o, err := out(input, path)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(o)
	if err != nil {
		log.Fatalln(fmt.Errorf("error writing stdout: %v", err))
	}
}

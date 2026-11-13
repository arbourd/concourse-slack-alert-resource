package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
	"github.com/arbourd/concourse-slack-alert-resource/slack"
)

// readLocalFile reads name relative to root, refusing paths that escape root.
func readLocalFile(root, name string) ([]byte, error) {
	// Resolve symlinks so links cannot escape root.
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	file, err := filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil {
		return nil, err
	}
	if rel, err := filepath.Rel(realRoot, file); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("%q is outside the build directory", name)
	}
	return os.ReadFile(file)
}

// paramFromFile returns the trimmed contents of file when set, else value.
// An unreadable file is logged and value is used instead.
func paramFromFile(root, param, value, file string) string {
	if file == "" {
		return value
	}
	f, err := readLocalFile(root, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s_file: %v\nwill default to %s instead\n", param, err, param)
		return value
	}
	if s := strings.TrimSpace(string(f)); s != "" {
		return s
	}
	return value
}

func buildMessage(alert Alert, m concourse.BuildMetadata, path string) *slack.Message {
	message := paramFromFile(path, "message", alert.Message, alert.MessageFile)
	channel := paramFromFile(path, "channel", alert.Channel, alert.ChannelFile)
	text := paramFromFile(path, "text", alert.Text, alert.TextFile)

	attachment := slack.Attachment{
		Fallback:   fmt.Sprintf("%s: %s/%s/%s -- %s", message, m.PipelineName, m.JobName, m.BuildName, m.URL),
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
		Text: text,
	}

	return &slack.Message{Attachments: []slack.Attachment{attachment}, Channel: channel}
}

func previousBuildStatus(input *concourse.OutRequest, m concourse.BuildMetadata) (string, error) {
	// Exit early if first build
	if m.BuildName == "1" {
		return "", nil
	}

	c, err := concourse.NewClient(m.Host, m.TeamName, input.Source.Username, input.Source.Password)
	if err != nil {
		return "", fmt.Errorf("error connecting to Concourse: %w", err)
	}

	p, err := previousBuildName(m.BuildName)
	if err != nil {
		return "", fmt.Errorf("error parsing build name: %w", err)
	}

	instanceVars := ""
	instanceVarsIndex := strings.Index(m.URL, "?")
	if instanceVarsIndex > -1 {
		instanceVars = m.URL[instanceVarsIndex:]
	}

	previous, err := c.JobBuild(m.PipelineName, m.JobName, p, instanceVars)
	if err != nil {
		return "", fmt.Errorf("error requesting Concourse build status: %w", err)
	}

	return previous.Status, nil
}

func previousBuildName(s string) (string, error) {
	strs := strings.Split(s, ".")

	if len(strs) == 1 {
		i, err := strconv.Atoi(strs[0])
		if err != nil {
			return "", err
		}

		return strconv.Itoa(i - 1), nil
	}

	i, err := strconv.Atoi(strs[1])
	if err != nil {
		return "", err
	}

	s = fmt.Sprintf("%s.%s", strs[0], strconv.Itoa(i-1))
	return strings.TrimSuffix(s, ".0"), nil
}

var maxElapsedTime = 30 * time.Second

func out(input *concourse.OutRequest, path string) (*concourse.OutResponse, error) {
	if input.Source.URL == "" {
		return nil, errors.New("slack webhook url cannot be blank")
	}

	alert := NewAlert(input)
	// Resolve the channel up front so skipped alerts report it too.
	alert.Channel = paramFromFile(path, "channel", alert.Channel, alert.ChannelFile)
	alert.ChannelFile = ""
	metadata := concourse.NewBuildMetadata(input.Source.ConcourseURL)
	if alert.Disabled {
		return buildOut(alert.Type, alert.Channel, false), nil
	}

	if alert.Type == "fixed" || alert.Type == "broke" {
		pstatus, err := previousBuildStatus(input, metadata)
		if err != nil {
			return nil, fmt.Errorf("error getting last build status: %w", err)
		}

		// An empty status means there is no previous build (build 1).
		if pstatus == "" || (alert.Type == "fixed" && pstatus == "succeeded") || (alert.Type == "broke" && pstatus != "succeeded") {
			return buildOut(alert.Type, alert.Channel, false), nil
		}
	}

	message := buildMessage(alert, metadata, path)
	err := slack.Send(input.Source.URL, message, maxElapsedTime)
	if err != nil {
		return nil, fmt.Errorf("error sending slack message: %w", err)
	}
	return buildOut(alert.Type, message.Channel, true), nil
}

func buildOut(atype string, channel string, alerted bool) *concourse.OutResponse {
	return &concourse.OutResponse{
		Version: concourse.Version{"ver": "static"},
		Metadata: []concourse.Metadata{
			{Name: "type", Value: atype},
			{Name: "channel", Value: channel},
			{Name: "alerted", Value: strconv.FormatBool(alerted)},
		},
	}
}

func main() {
	// The first argument is the path to the build's sources.
	path := os.Args[1]

	var input *concourse.OutRequest
	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		log.Fatalln(fmt.Errorf("error reading stdin: %w", err))
	}

	o, err := out(input, path)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(o)
	if err != nil {
		log.Fatalln(fmt.Errorf("error writing stdout: %w", err))
	}
}

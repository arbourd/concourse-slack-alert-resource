package main

import "github.com/arbourd/concourse-slack-alert-resource/concourse"

// An Alert defines the notification that will be sent to Slack.
type Alert struct {
	Type        string
	Channel     string
	ChannelFile string
	Color       string
	IconURL     string
	Message     string
	MessageFile string
	Text        string
	TextFile    string
	Disabled    bool
}

const iconBase = "https://ci.concourse-ci.org/public/images/"

// alertDefaults holds the color, icon and message for each alert type.
var alertDefaults = map[string]Alert{
	"success": {Type: "success", Color: "#32cd32", IconURL: iconBase + "favicon-succeeded.png", Message: "Success"},
	"failed":  {Type: "failed", Color: "#d00000", IconURL: iconBase + "favicon-failed.png", Message: "Failed"},
	"started": {Type: "started", Color: "#f7cd42", IconURL: iconBase + "favicon-started.png", Message: "Started"},
	"aborted": {Type: "aborted", Color: "#8d4b32", IconURL: iconBase + "favicon-aborted.png", Message: "Aborted"},
	"fixed":   {Type: "fixed", Color: "#32cd32", IconURL: iconBase + "favicon-succeeded.png", Message: "Fixed"},
	"broke":   {Type: "broke", Color: "#d00000", IconURL: iconBase + "favicon-failed.png", Message: "Broke"},
	"errored": {Type: "errored", Color: "#f5a623", IconURL: iconBase + "favicon-errored.png", Message: "Errored"},
	"default": {Type: "default", Color: "#35495c", IconURL: iconBase + "favicon-pending.png"},
}

// NewAlert constructs and returns an Alert.
func NewAlert(input *concourse.OutRequest) Alert {
	alert, ok := alertDefaults[input.Params.AlertType]
	if !ok {
		alert = alertDefaults["default"]
	}

	alert.Disabled = input.Params.Disable || input.Source.Disable

	alert.Channel = input.Params.Channel
	if alert.Channel == "" {
		alert.Channel = input.Source.Channel
	}
	alert.ChannelFile = input.Params.ChannelFile

	if input.Params.Message != "" {
		alert.Message = input.Params.Message
	}
	alert.MessageFile = input.Params.MessageFile

	if input.Params.Color != "" {
		alert.Color = input.Params.Color
	}

	alert.Text = input.Params.Text
	alert.TextFile = input.Params.TextFile
	return alert
}

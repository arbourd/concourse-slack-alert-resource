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

// NewAlert constructs and returns an Alert.
func NewAlert(input *concourse.OutRequest) Alert {
	var alert Alert
	switch input.Params.AlertType {
	case "success":
		alert = Alert{
			Type:    "success",
			Color:   "#32cd32",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-succeeded.png",
			Message: "Success",
		}
	case "failed":
		alert = Alert{
			Type:    "failed",
			Color:   "#d00000",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-failed.png",
			Message: "Failed",
		}
	case "started":
		alert = Alert{
			Type:    "started",
			Color:   "#f7cd42",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-started.png",
			Message: "Started",
		}
	case "aborted":
		alert = Alert{
			Type:    "aborted",
			Color:   "#8d4b32",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-aborted.png",
			Message: "Aborted",
		}
	case "fixed":
		alert = Alert{
			Type:    "fixed",
			Color:   "#32cd32",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-succeeded.png",
			Message: "Fixed",
		}
	case "broke":
		alert = Alert{
			Type:    "broke",
			Color:   "#d00000",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-failed.png",
			Message: "Broke",
		}
	case "errored":
		alert = Alert{
			Type:    "errored",
			Color:   "#f5a623",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-errored.png",
			Message: "Errored",
		}
	default:
		alert = Alert{
			Type:    "default",
			Color:   "#35495c",
			IconURL: "https://ci.concourse-ci.org/public/images/favicon-pending.png",
			Message: "",
		}
	}

	alert.Disabled = input.Params.Disable
	if alert.Disabled == false {
		alert.Disabled = input.Source.Disable
	}

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

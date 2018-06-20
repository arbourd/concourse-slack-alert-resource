package concourse

type Source struct {
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	ConcourseURL string `json:"concourse_url"`
	Channel      string `json:"channel"`
}

type Metadata struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type Version map[string]string

type OutRequest struct {
	Source Source    `json:"source"`
	Params OutParams `json:"params"`
}

type OutParams struct {
	AlertType string `json:"alert_type"`
	Message   string `json:"message"`
	Color     string `json:"color"`
	Disable   bool   `json:"disable"`
	Channel   string `json:"channel"`
}

type OutResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata"`
}

type InResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata"`
}

type CheckResponse []Version

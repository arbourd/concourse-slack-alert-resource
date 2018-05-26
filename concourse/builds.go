package concourse

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

type BuildMetadata struct {
	URL          string
	TeamName     string
	PipelineName string
	JobName      string
	BuildName    string
}

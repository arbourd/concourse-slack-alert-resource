package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/arbourd/concourse-slack-alert-resource/concourse"
)

func main() {
	err := json.NewEncoder(os.Stdout).Encode(concourse.InResponse{Version: concourse.Version{"ref": ""}})
	if err != nil {
		log.Fatalln(err)
	}
}

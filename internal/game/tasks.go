package game

import (
	"encoding/json"
	"os"
)

var tasks []string

func loadTasks(filename string) error {

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, tasks)
}

package tasks

import (
	"encoding/json"
	"os"
)

func loadTasksFromFile(filename string) ([]string, error) {

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var tasks []string

	err = json.Unmarshal(data, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

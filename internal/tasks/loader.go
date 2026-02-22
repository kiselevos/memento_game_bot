package tasks

import (
	"encoding/json"
	"math/rand"
	"os"

	"github.com/kiselevos/memento_game_bot/internal/models"
)

func LoadTasksFromFile() ([]models.Task, error) {

	data, err := os.ReadFile("./assets/tasks.json")
	if err != nil {
		return nil, err
	}

	var tasks []models.Task

	err = json.Unmarshal(data, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func ShuffleTasks(tasks []models.Task) {
	rand.Shuffle(len(tasks), func(i, j int) {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	})
}

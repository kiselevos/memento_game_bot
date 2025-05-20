package tasks

import (
	"errors"
	"math/rand"
	"sync"
)

type TasksList struct {
	allTasks []string
	mu       *sync.Mutex
}

// NewTasksList - Конструктор для структуры списка вопросов
func NewTasksList(filepath string) (*TasksList, error) {
	tasks, err := loadTasksFromFile(filepath)
	if err != nil {
		return nil, err
	}

	return &TasksList{
		allTasks: tasks,
		mu:       &sync.Mutex{},
	}, nil
}

// GetRandomTask - метод принимающий мапу использованных вопросов, возвращающий один из несипользуемых.
func (tl *TasksList) GetRandomTask(used map[string]bool) (string, error) {

	tl.mu.Lock()
	defer tl.mu.Unlock()

	var avalibalTasks []string
	for _, task := range tl.allTasks {
		if !used[task] {
			avalibalTasks = append(avalibalTasks, task)
		}
	}

	if len(avalibalTasks) == 0 {
		return "", errors.New("Все задания уже использованы")
	}

	return avalibalTasks[rand.Intn(len(avalibalTasks))], nil
}

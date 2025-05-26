package models

import "gorm.io/gorm"

type Task struct {
	gorm.Model
	Text     string `gorm:"column:text;uniqueIndex"`
	UseCount int    `gorm:"column:use_count"` // количестов раундов, который данный вопрос играли (хотя бы одна присланная фотография)
}

func NewTask(text string) *Task {
	return &Task{
		Text: text,
	}
}

func (t *Task) AddUseCount() {
	t.UseCount++
}

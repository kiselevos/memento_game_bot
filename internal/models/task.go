package models

import "gorm.io/gorm"

type Task struct {
	gorm.Model
	Text      string `gorm:"column:text;uniqueIndex"`
	UseCount  int    `gorm:"column:use_count"`  // количестов фото на данный вопрос
	SkipCount int    `gorm:"column:skip_count"` // количество раз, когда пропускали этот вопрос.
}

func NewTask(text string) *Task {
	return &Task{
		Text: text,
	}
}

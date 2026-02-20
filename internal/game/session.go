package game

import (
	"fmt"
	"math/rand"
	"sort"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/models"
)

// GameSession - Хранит данные о конкретной партии игры
type GameSession struct {

	// Постоянные
	SessionID   int64            // ID сессии для статистики
	ChatID      int64            // Номер чата, где идет игра
	Host        User             // Ведущий игры для управления
	Score       map[int64]int    // Мапа с очками юзеров
	UserNames   map[int64]string //Список участников партии
	CountRounds int              // Сыгранные раунды
	Tasks       []models.Task    // Задания для данной игры

	// Обнуляющиеся при новом раунде

	FSM              *FSM             // Машина состояний
	Votes            map[int64]int64  // Кто кому отдал свой голос в раунде
	UsersPhoto       map[int64]string // Хранение фотографий, отпрвленных юзером
	CarrentTask      models.Task      // Текущее задание
	IndexPhotoToUser map[int]int64    // Мапа для голосования(Индекс очердности фото к игроку)
	VotePhotoMsgIDs  map[int]int      // Мапа для хранения msgID для удаления кнопок
	SystemMsgIDs     []int            // Слайс для хранения msgID для удаления системных сообщений
}

var (
	ErrNoPhotosToVote = fmt.Errorf("нет фото для голосования")
	ErrFSMState       = fmt.Errorf("ошибка перехода FSM")
)

type PlayerScore struct {
	UserID   int64
	UserName string
	Value    int
}

// Проверка ведущего игры
func (s *GameSession) IsHost(userID int64) bool {
	return s.Host.ID == userID
}

// GetUserName - возвращает имя или ник пользователя
func (s *GameSession) GetUserName(userID int64) string {
	if name, ok := s.UserNames[userID]; ok {
		return name
	}
	return "Анонимный Осётр"
}

func (s *GameSession) TotalScore() []PlayerScore {
	return s.scoreFromMap(s.Score)
}

func (s *GameSession) RoundScore() []PlayerScore {
	voteCount := make(map[int64]int)
	for _, votedFor := range s.Votes {
		voteCount[votedFor]++
	}
	return s.scoreFromMap(voteCount)
}

func (s *GameSession) scoreFromMap(data map[int64]int) []PlayerScore {
	var result []PlayerScore

	for userID, val := range data {
		result = append(result, PlayerScore{
			UserID:   userID,
			UserName: s.GetUserName(userID),
			Value:    val,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Value > result[j].Value
	})

	return result
}

func (s *GameSession) TakePhoto(user *User, photoID string) bool {

	if _, ok := s.UserNames[user.ID]; !ok {
		s.UserNames[user.ID] = DisplayNameHTML(user)
	}

	// Проверяем на уже существующее фото.
	_, existed := s.UsersPhoto[user.ID]
	s.UsersPhoto[user.ID] = photoID

	return existed
}

// Начало нового раунда
func (s *GameSession) StartNewRound() (prevTaskID int64, countPhoto int, task models.Task, err error) {
	prevTaskID = s.CarrentTask.ID
	countPhoto = len(s.UsersPhoto)

	newTask := models.Task{}

	if !SafeTrigger(s.FSM, EventStartRound, "GameSession.StartNewRound") {
		return prevTaskID, 0, newTask, ErrFSMState
	}

	s.CarrentTask, err = s.NextTask()
	if err != nil {
		return prevTaskID, countPhoto, newTask, err
	}

	newTask = s.CarrentTask

	// готовимсся к новому раунду
	s.UsersPhoto = make(map[int64]string)
	// Чистим предыдущее голосование
	s.Votes = make(map[int64]int64)
	s.IndexPhotoToUser = make(map[int]int64)
	s.VotePhotoMsgIDs = make(map[int]int)

	return prevTaskID, countPhoto, newTask, nil
}

type VotePhoto struct {
	Num     int
	UserID  int64
	PhotoID string
}

func (s *GameSession) StartVoting() ([]VotePhoto, error) {
	if len(s.UsersPhoto) == 0 {
		return nil, ErrNoPhotosToVote
	}

	if !SafeTrigger(s.FSM, EventStartVote, "GameSession.StartVoting") {
		return nil, ErrFSMState
	}

	items := make([]VotePhoto, 0, len(s.UsersPhoto))
	for uid, pid := range s.UsersPhoto {
		items = append(items, VotePhoto{UserID: uid, PhotoID: pid})
	}

	rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	for i := range items {
		items[i].Num = i + 1
		s.IndexPhotoToUser[items[i].Num] = items[i].UserID
	}

	return items, nil
}

func (s *GameSession) RegisterVote(voter *User, photoNum int) (bool, string, error) {

	if s.FSM.Current() != VoteState {
		return false, messages.VotedNotActive, nil
	}

	if _, voted := s.Votes[voter.ID]; voted {
		return false, messages.VotedAlready, nil
	}

	targetUserID, ok := s.IndexPhotoToUser[photoNum]
	if !ok {
		return false, messages.ErrorMessagesForUser, fmt.Errorf("unknown photo num")
	}

	if targetUserID == voter.ID {
		return false, messages.VotedForSelf, nil
	}

	s.Votes[voter.ID] = targetUserID
	s.Score[targetUserID]++

	if _, ok := s.UserNames[voter.ID]; !ok {
		s.UserNames[voter.ID] = DisplayNameHTML(voter)
	}

	return true, fmt.Sprintf("%s проголосовал(а)", s.GetUserName(voter.ID)), nil
}

func (s *GameSession) FinishVoting() error {
	if !SafeTrigger(s.FSM, EventFinishVote, "GameSession.FinishVoting") {
		return ErrFSMState
	}
	// Считаем завершенные раунды.
	s.CountRounds++
	return nil
}

// Получение нового вопроса
func (s *GameSession) NextTask() (models.Task, error) {
	if len(s.Tasks) == 0 {
		return models.Task{}, ErrNoTasksLeft
	}

	// Вопросы уже перемешаны, берем последний и удаляем его
	lastIndex := len(s.Tasks) - 1
	task := s.Tasks[lastIndex]

	s.Tasks = s.Tasks[:lastIndex]

	return task, nil
}

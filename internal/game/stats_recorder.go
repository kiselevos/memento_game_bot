package game

type StatsRecorder interface {
	CreateSessionRecord(chatID int64)
	RegisterUserLinkedToSession(chatID int64, user User)
	IncrementPhotoSubmission(chatID int64, userID int64)
	RecordVote(voterID int64)
	IncrementTaskUsage(task string, hadPhotos bool)
	RegisterRoundTask(chatID int64, task string)
	IsFirstGame(chatID int64) (bool, error)
}

// NoopStatsRecorder — дефолт: ничего не делает.
type NoopStatsRecorder struct{}

func (NoopStatsRecorder) CreateSessionRecord(chatID int64)                    {}
func (NoopStatsRecorder) RegisterUserLinkedToSession(chatID int64, user User) {}
func (NoopStatsRecorder) IncrementPhotoSubmission(chatID int64, userID int64) {}
func (NoopStatsRecorder) RecordVote(voterID int64)                            {}
func (NoopStatsRecorder) IncrementTaskUsage(task string, hadPhotos bool)      {}
func (NoopStatsRecorder) RegisterRoundTask(chatID int64, task string)         {}
func (NoopStatsRecorder) IsFirstGame(chatID int64) (bool, error)              { return true, nil }

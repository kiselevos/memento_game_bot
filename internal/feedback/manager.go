package feedback

import (
	"context"
	"sync"
	"time"
)

type FeedbackManager struct {
	mu         sync.Mutex
	waiting    map[int64]context.CancelFunc
	timeoutDur time.Duration
}

func NewFeedbackManager(timeout time.Duration) *FeedbackManager {
	return &FeedbackManager{
		waiting:    make(map[int64]context.CancelFunc),
		timeoutDur: timeout,
		mu:         sync.Mutex{},
	}
}

func (fm *FeedbackManager) StartFeedback(userID int64) {

	fm.mu.Lock()
	defer fm.mu.Unlock()

	if cancel, ok := fm.waiting[userID]; ok {
		cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), fm.timeoutDur)
	fm.waiting[userID] = cancel

	go func() {
		<-ctx.Done()
		fm.mu.Lock()
		delete(fm.waiting, userID)
		fm.mu.Unlock()
	}()

}

func (fm *FeedbackManager) CancelFeedback(userID int64) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if cancel, ok := fm.waiting[userID]; ok {
		cancel()
		delete(fm.waiting, userID)
	}
}

func (fm *FeedbackManager) IsWaitingFeedback(userID int64) bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	_, ok := fm.waiting[userID]
	return ok
}

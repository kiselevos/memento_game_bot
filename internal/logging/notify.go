package logging

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	tb "gopkg.in/telebot.v3"
)

type Notifier struct {
	Bot      *tb.Bot
	AdminIDs []int64

	mu   sync.Mutex
	last time.Time
	min  time.Duration
}

func NewNotifier(b *tb.Bot, admins []int64) *Notifier {
	return &Notifier{
		Bot:      b,
		AdminIDs: admins,
		min:      30 * time.Second,
	}
}

func (n *Notifier) Notify(level slog.Level, msg string, attrs ...any) {
	if n == nil || n.Bot == nil || len(n.AdminIDs) == 0 {
		return
	}

	n.mu.Lock()
	if time.Since(n.last) < n.min {
		n.mu.Unlock()
		return
	}
	n.last = time.Now()
	n.mu.Unlock()

	text := fmt.Sprintf("🚨 %s: %s", level.String(), msg)
	for i := 0; i+1 < len(attrs); i += 2 {
		text += fmt.Sprintf("\n%v=%v", attrs[i], attrs[i+1])
	}

	_, _ = n.Bot.Send(&tb.User{ID: n.AdminIDs[0]}, text)
}

var (
	notifierMu sync.RWMutex
	globalN    *Notifier
)

func SetNotifier(n *Notifier) {
	notifierMu.Lock()
	globalN = n
	notifierMu.Unlock()
}

func Notify(level slog.Level, msg string, kv ...any) {
	notifierMu.RLock()
	n := globalN
	notifierMu.RUnlock()
	if n != nil {
		n.Notify(level, msg, kv...)
	}
}

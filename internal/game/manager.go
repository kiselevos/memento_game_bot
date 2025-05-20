package game

import "sync"

type GameManager struct {
	sessions map[int64]*GameSession
	mu       *sync.Mutex
}

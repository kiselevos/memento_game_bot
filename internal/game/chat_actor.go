package game

type chatActor struct {
	chatID  int64
	inbox   chan actorMsg
	session *GameSession
}

type actorMsg struct {
	fn    func(a *chatActor) error
	reply chan error
}

func newChatActor(chatID int64) *chatActor {
	a := &chatActor{
		chatID: chatID,
		inbox:  make(chan actorMsg, 64),
	}
	go func() {
		for m := range a.inbox {
			m.reply <- m.fn(a)
		}
	}()
	return a
}

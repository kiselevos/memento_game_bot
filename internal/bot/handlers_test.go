package bot

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	tb "gopkg.in/telebot.v3"
)

type MockBot struct {
	mock.Mock
}

func (m *MockBot) Send(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error) {
	args := m.Called(to, what)
	return &tb.Message{}, args.Error(1)
}

func (m *MockBot) Delete(msg tb.Editable) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockBot) Respond(c *tb.Callback, resp ...*tb.CallbackResponse) error {
	args := m.Called(c, resp)
	return args.Error(0)
}

func (m *MockBot) Handle(endpoint interface{}, handler tb.HandlerFunc, middlewear ...tb.MiddlewareFunc) {
	m.Called(endpoint, handler)
}

type mockContext struct {
	tb.Context
	chat    *tb.Chat
	message *tb.Message
	sender  *tb.User
	mockBot *MockBot
}

func (m *mockContext) Chat() *tb.Chat {
	return m.chat
}

func (m *mockContext) Message() *tb.Message {
	return m.message
}

// имитируем успешную отправку
func (m *mockContext) Send(what interface{}, _ ...interface{}) error {
	_, err := m.mockBot.Send(m.chat, what)
	return err
}

func (m *mockContext) Sender() *tb.User {
	return m.sender
}

func (m *mockContext) Callback() *tb.Callback {
	return nil
}

func SetupTestHandler() (*MockBot, *Handlers, *tb.Chat, *mockContext, *game.GameSession) {

	mockBot := new(MockBot)

	gm := game.NewGameManager()
	tl := tasks.NewTasksListForTest([]string{"test task"})

	handlers := NewHandlers(mockBot, gm, tl)

	const testChatID = 12345
	session := gm.StartNewGameSession(testChatID)

	chat := &tb.Chat{ID: testChatID}
	fakeCtx := &tb.Message{Chat: chat}
	ctx := &mockContext{chat: chat, message: fakeCtx, mockBot: mockBot}

	return mockBot, handlers, chat, ctx, session
}

func TestStartGame(t *testing.T) {

	mockBot, handlers, chat, ctx, _ := SetupTestHandler()

	mockBot.On("Send", chat, mock.Anything).Return(&tb.Message{}, nil)

	err := handlers.Start(ctx)
	if err != nil {
		t.Errorf("Command /start return error: %v", err)
	}

	err = handlers.StartGame(ctx)
	if err != nil {
		t.Errorf("Command /startgame return error: %v", err)
	}
	mockBot.AssertExpectations(t)
	mockBot.AssertCalled(t, "Send", chat, mock.Anything)
}

func TestStartRound(t *testing.T) {

	mockBot, handlers, chat, ctx, _ := SetupTestHandler()

	mockBot.On("Send", chat, mock.MatchedBy(func(msg interface{}) bool {
		text, ok := msg.(string)
		return ok && strings.Contains(text, "🎲") // начало задания
	})).Return(&tb.Message{}, nil)

	err := handlers.HandleStartRound(ctx)
	if err != nil {
		t.Errorf("HandleStartRound returned error: %v", err)
	}

	mockBot.AssertCalled(t, "Send", chat, mock.Anything)
	mockBot.AssertExpectations(t)
}

func TestTakeUserPhoto(t *testing.T) {
	mockBot, handlers, chat, ctx, session := SetupTestHandler()

	const (
		userID   int64  = 12345
		username string = "kiselevos"
		photoID  string = "test_photo"
	)

	user := &tb.User{ID: userID, Username: username}
	message := &tb.Message{
		Chat:   chat,
		Sender: user,
		Photo:  &tb.Photo{File: tb.File{FileID: photoID}},
	}

	session.FSM.ForceState(game.RoundStartState)
	session.UsersPhoto = make(map[int64]string)

	ctx.sender = user
	ctx.message = message

	// Моки
	mockBot.On("Delete", mock.MatchedBy(func(m tb.Editable) bool {
		// Любое сообщение, потому что оно из контекста
		return true
	})).Return(nil).Once()

	mockBot.On("Send", chat, mock.MatchedBy(func(val interface{}) bool {
		text, ok := val.(string)
		return ok && strings.Contains(text, messages.PhotoReceived)
	})).Return(&tb.Message{}, nil).Once()

	err := handlers.TakeUserPhoto(ctx)
	assert.NoError(t, err)

	got := session.UsersPhoto[userID]
	if got != photoID {
		t.Errorf("Expected photo %s to be stored for user %d, got %s", photoID, userID, got)
	}

	mockBot.AssertExpectations(t)
}

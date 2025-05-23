package bot

import (
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"strings"
	"testing"

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

func (m *mockContext) Callback() *tb.Callback {
	return nil
}

func TestStartGame(t *testing.T) {

	mockBot := new(MockBot)

	gm := game.NewGameManager()
	tl := &tasks.TasksList{AllTasks: []string{"test task"}}

	handlers := NewHandlers(mockBot, gm, tl)

	const testChatId = 12345
	chat := &tb.Chat{ID: testChatId}
	fakeCtx := &tb.Message{Chat: chat}
	ctx := &mockContext{chat: chat, message: fakeCtx, mockBot: mockBot}

	mockBot.On("Send", chat, mock.Anything).Return(&tb.Message{}, nil)

	err := handlers.Start(ctx)
	if err != nil {
		t.Errorf("Command /start return error: %v", err)
	}

	err = handlers.StartGame(ctx)
	if err != nil {
		t.Errorf("Command /startgame return error: %v", err)
	}

	mockBot.AssertCalled(t, "Send", chat, mock.Anything)
}

func TestStartRound(t *testing.T) {
	mockBot := new(MockBot)

	gm := game.NewGameManager()
	tl := tasks.NewTasksListForTest([]string{"test task"})

	handlers := NewHandlers(mockBot, gm, tl)

	const testChatId = 12345
	gm.StartNewGameSession(testChatId)

	chat := &tb.Chat{ID: testChatId}
	fakeCtx := &tb.Message{Chat: chat}
	ctx := &mockContext{chat: chat, message: fakeCtx, mockBot: mockBot}

	mockBot.On("Send", chat, mock.MatchedBy(func(msg interface{}) bool {
		text, ok := msg.(string)
		return ok && strings.Contains(text, "🎲") // начало задания
	})).Return(&tb.Message{}, nil)

	err := handlers.HandleStartRound(ctx)
	if err != nil {
		t.Errorf("HandleStartRound returned error: %v", err)
	}

	mockBot.AssertCalled(t, "Send", chat, mock.Anything)
}

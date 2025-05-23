package bot

import (
	"PhotoBattleBot/internal/game"
	"fmt"
	"strings"

	"gopkg.in/telebot.v3"
)

const (
	FinalScore = "🏁 Игра завершена!\n\n📊 Финальный счёт:"
	GameScore  = "🏆 Текущий результат игры:"
	RoundScore = "⭐ Результаты раунда:"
)

func RenderScore(title string, scores []game.PlayerScore) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n\n", title))
	for i, ps := range scores {
		if title == RoundScore {
			b.WriteString(fmt.Sprintf("%d. %s — %s\n", i+1, ps.UserName, strings.Repeat("🔥", ps.Value)))
		} else {
			b.WriteString(fmt.Sprintf("%d. %s — %d 🔥\n", i+1, ps.UserName, ps.Value))
		}
	}
	return b.String()
}

// Обертка для групповых команд.
func GroupOnly(handler telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		chatType := c.Chat().Type
		if chatType != telebot.ChatGroup && chatType != telebot.ChatSuperGroup {
			return c.Send("Играть в одиночестве - не интересно. Добавь меня в группу друзей.")
		}
		return handler(c)
	}
}

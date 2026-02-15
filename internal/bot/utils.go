package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"

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

// Анимация загрузки
func WaitingAnimation(c telebot.Context, bot botinterface.BotInterface, t int) {

	steps := []string{"⏳ Подготовка", " ⏳ Подготовка.", "  ⏳ Подготовка..", "  ⏳ Подготовка..."}

	msg, err := bot.Send(&telebot.Chat{ID: c.Chat().ID}, steps[0])
	if err != nil {
		return
	}

	for i := 0; i < t; i++ {
		time.Sleep(1 * time.Second)
		step := steps[i%len(steps)]
		bot.Edit(msg, step)
	}

	_ = bot.Delete(msg)
}

package bot

import (
	"PhotoBattleBot/internal/game"
	"fmt"
	"sort"
	"strings"
)

const (
	FinalScore = "🏁 Игра завершена!\n\n📊 Финальный счёт:"
	GameScore  = "🏆 Текущий результат игры:"
	RoundScore = "⭐ Результаты раунда:"
)

func RenderResults(session *game.GameSession, title string) string {

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s\n\n", title))

	type playerResult struct {
		userID int64
		score  int
	}

	userScore := make(map[int64]int)

	switch title {
	case FinalScore:
		userScore = session.Score
	case GameScore:
		userScore = session.Score
	case RoundScore:
		for _, votedFor := range session.Votes {
			userScore[votedFor]++
		}
	}

	var results []playerResult
	for userID, score := range userScore {
		results = append(results, playerResult{userID, score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	for i, res := range results {
		name := session.GetUserName(res.userID)
		result.WriteString(fmt.Sprintf("%d. %s — %d 🔥\n", i+1, name, res.score))
	}

	return result.String()
}

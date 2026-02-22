package game

import (
	"html"
	"strings"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"gopkg.in/telebot.v3"
)

type User struct {
	ID        int64
	Username  string
	FirstName string
}

// Достаем User из телеги
func GetUserFromTelebot(u *telebot.User) User {
	if u == nil {
		return User{}
	}

	return User{
		ID:        u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
	}
}

// Как обращаться к участнику
func DisplayNameHTML(u *User) string {
	if u == nil {
		return messages.UnnownPerson
	}

	// FirstName приоритет
	name := strings.TrimSpace(u.FirstName)
	if name != "" {
		return html.EscapeString(name)
	}

	// Если нет FirstName используем username
	if u.Username != "" {
		return "@" + html.EscapeString(u.Username)
	}

	// Фолбэк
	return messages.UnnownPerson
}

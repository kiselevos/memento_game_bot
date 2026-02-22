package logging

import (
	"log/slog"
	"os"

	"github.com/kiselevos/memento_game_bot/internal/config"
)

func InitSlogLogger(logCfg config.LogConfig) *slog.Logger {

	var handler slog.Handler

	if logCfg.AppEnv == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logCfg.Level,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     logCfg.Level,
			AddSource: true,
		})
	}

	logger := slog.New(handler).With(
		"service", "memento-bot",
		"env", logCfg.AppEnv,
	)

	slog.SetDefault(logger)
	return logger
}

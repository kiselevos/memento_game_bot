# 📸 Memento Game Bot
Telegram-бот для групповой фото-игры во время видеозвонков.

Memento помогает оживить разговор: бот присылает задание, участники делятся фото и историями, затем голосуют за лучший ответ.
---

## Технологический стек
- **Go 1.22+**  
- **Telebot** (Работа с Telegram Bot API)  
- **PostgreSQL + GORM** (Хранение данных и ORM)  
- **FSM (Finite State Machine)** (Управление игровыми состояниями)
- **Docker**, **Makefile** (контейнеризация и автоматизация сборки)  


## Запуск приложения

### Клонируйте репозиторий
```bash
git clone https://github.com/kiselevos/memento_game_bot.git
cd memento_game_bot
```

### Установите зависимости
```bash
go mod tidy
go mod download

# или одной командой make
make tidy
```

### Заполните настройки окружения
```bash
# Пример .env
APP_ENV=local

DB_USER=memento_user
DB_PASSWORD=memento_password
DB_NAME=memento_battle_bot
DB_PORT=5432

ADMINS_ID=your_tg_id
TG_TOKEN=your_telegram_bot_token
```
> В APP_ENV=local - бот подключается к localhost:5432, а при APP_ENV=docker - к контейнеру postgres.

### Поднимите базу данных и выполните миграции
```bash
docker compose -f docker-compose.db.yml up -d postgres
docker compose -f docker-compose.db.yml run --rm migrate

# или одной командой make 
make setup
```

### Запусстите бота локально
```bash
go run ./cmd/main.go

# или командой make 
make run
```

## Основные команды

- `/start` - приветственное сообщение  
- `/startgame` - начать новую игру (сброс текущей)  
- `/endgame` - завершить игру и показать финальный счёт  
- `/newround` - начать новый раунд  
- `/vote` - начать голосование  
- `/finishvote` - досрочно завершить голосование  
- `/score` - текущие очки игроков
- `/feedback` - обратная связь
---

### Проектная структура
```bash
memento_game_bot/
│
├── Dockerfile
├── Makefile
├── README.md
├── go.mod / go.sum
│
├── assets/                   # Статические данные
│   ├── messages.go            # Текстовые шаблоны и сообщения
│   └── tasks.json             # Задания для раундов
│
├── cmd/
│   └── main.go                # Точка входа: инициализация зависимостей, запуск бота
│
├── config/
│   └── config.go              # Загрузка и валидация переменных окружения
│
├── docker-compose.yml         # Основной compose (бот + postgres)
├── docker-compose.db.yml      # Только база и миграции
│
├── internal/                  # Внутренняя логика приложения
│   ├── bot/                   # Телеграм-хелперы и middleware
│   │   ├── middleware/
│   │   │   ├── check_bot_rights.go
│   │   │   ├── chek_admin.go
│   │   │   ├── command_filter.go
│   │   │   └── pulling.go
│   │   └── utils.go
│   │
│   ├── botinterface/          # Интерфейсы для тестирования и абстракций Telebot
│   │   └── botiface.go
│   │
│   ├── feedback/              # Обработка отзывов пользователей
│   │   └── manager.go
│   │
│   ├── game/                  # FSM и управление игровыми сессиями
│   │   ├── fsm.go
│   │   ├── fsm_test.go
│   │   ├── manager.go
│   │   ├── manager_test.go
│   │   ├── session.go
│   │   └── session_test.go
│   │
│   ├── handlers/              # Обработка команд Telegram
│   │   ├── feedback.go
│   │   ├── game.go
│   │   ├── init.go
│   │   ├── photo.go
│   │   ├── round.go
│   │   ├── score.go
│   │   └── vote.go
│   │
│   ├── logging/               # Настройка логгера
│   │   └── logger.go
│   │
│   ├── models/                # Модели БД
│   │   ├── session.go
│   │   ├── task.go
│   │   └── user.go
│   │
│   ├── repositories/          # Репозитории для работы с БД
│   │   ├── session.go
│   │   ├── task.go
│   │   └── user.go
│   │
│   └── tasks/                 # Сервис загрузки и выбора заданий
│       ├── loader.go
│       └── services.go
│
├── pkg/
│   └── db/
│       └── db.go              # Подключение к PostgreSQL
│
├── migrations/
│   └── auto.go                # Автоматические миграции GORM
│
└── logs/
    └── bot.log                # Логи приложения
```

### Осеновные команды Make
```bash
make tidy    # Установка зависимостей Go
make setup   # Поднять PostgreSQL и применить миграции
make run     # Запустить бота локально
make restart    # Перезапустит базу данных и миграции
make db-stop # Остановка контейнеров
```

## 📬 Обратная связь

Хотите предложить задание, сообщить об ошибке или просто сказать спасибо?  
Связь: [@kiselevos](https://t.me/kiselevos)

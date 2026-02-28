package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStorage реализует хранилище на базе PostgreSQL с пулом соединений (pgxpool).
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создаёт подключение к PostgreSQL, применяет миграции и возвращает *PostgresStorage.
// Принимает строку подключения (config.DatabaseURL).
func NewPostgresStorage(dbURL string) *PostgresStorage {
	if dbURL == "" {
		panic("DATABASE_URL is not set (pass config.DatabaseURL)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		panic(fmt.Sprintf("failed to parse DATABASE_URL: %v", err))
	}

	// Настройки пула
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
	}

	// Проверка подключения
	if err := pool.Ping(ctx); err != nil {
		panic(fmt.Sprintf("failed to ping postgres: %v", err))
	}

	s := &PostgresStorage{pool: pool}
	s.migrate(ctx)
	return s
}

// Close закрывает пул соединений. Вызывайте при завершении приложения.
func (s *PostgresStorage) Close() {
	s.pool.Close()
}

// Ping проверяет доступность базы данных.
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Pool возвращает *pgxpool.Pool для прямых запросов, если нужно.
func (s *PostgresStorage) Pool() *pgxpool.Pool {
	return s.pool
}

// ---------- миграции ----------

func (s *PostgresStorage) migrate(ctx context.Context) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id          BIGSERIAL PRIMARY KEY,
			chat_id     BIGINT      NOT NULL,
			name        TEXT        NOT NULL,
			date        TEXT        NOT NULL,
			description TEXT        NOT NULL DEFAULT '',
			status      TEXT        NOT NULL DEFAULT 'active',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(chat_id, name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_chat_id ON events (chat_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_name    ON events (name)`,

		`CREATE TABLE IF NOT EXISTS user_events (
			id       BIGSERIAL PRIMARY KEY,
			chat_id  BIGINT NOT NULL,
			user_id  BIGINT NOT NULL,
			event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			UNIQUE(chat_id, user_id, event_id)
		)`,
	}

	for _, q := range queries {
		if _, err := s.pool.Exec(ctx, q); err != nil {
			panic(fmt.Sprintf("migration failed: %v\nquery: %s", err, q))
		}
	}
}

// ---------- Events CRUD ----------

// CreateEvent создаёт событие. Возвращает ошибку, если событие с таким именем в чате уже существует.
func (s *PostgresStorage) CreateEvent(ctx context.Context, chatID int64, name, date, description string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO events (chat_id, name, date, description) VALUES ($1, $2, $3, $4)`,
		chatID, name, date, description,
	)
	return err
}

// GetEvent возвращает событие по chat_id + name.
func (s *PostgresStorage) GetEvent(ctx context.Context, chatID int64, name string) (*Event, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, date, description, status FROM events WHERE chat_id = $1 AND name = $2`,
		chatID, name,
	)
	var e Event
	if err := row.Scan(&e.ID, &e.ChatID, &e.Name, &e.Date, &e.Description, &e.Status); err != nil {
		return nil, err
	}
	return &e, nil
}

// ListEvents возвращает все события чата.
func (s *PostgresStorage) ListEvents(ctx context.Context, chatID int64) ([]Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, chat_id, name, date, description, status FROM events WHERE chat_id = $1 ORDER BY created_at`,
		chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.ChatID, &e.Name, &e.Date, &e.Description, &e.Status); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindEventAcrossChats ищет событие по имени во всех чатах, исключая excludeChatID.
// Возвращает событие и chat_id, в котором оно найдено.
func (s *PostgresStorage) FindEventAcrossChats(ctx context.Context, name string, excludeChatID int64) (*Event, int64, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, date, description, status FROM events WHERE name = $1 AND chat_id <> $2 LIMIT 1`,
		name, excludeChatID,
	)
	var e Event
	if err := row.Scan(&e.ID, &e.ChatID, &e.Name, &e.Date, &e.Description, &e.Status); err != nil {
		return nil, 0, err
	}
	return &e, e.ChatID, nil
}

// UpdateEventStatus обновляет статус события.
func (s *PostgresStorage) UpdateEventStatus(ctx context.Context, chatID int64, name, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE events SET status = $1 WHERE chat_id = $2 AND name = $3`,
		status, chatID, name,
	)
	return err
}

// DeleteEvent удаляет событие.
func (s *PostgresStorage) DeleteEvent(ctx context.Context, chatID int64, name string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM events WHERE chat_id = $1 AND name = $2`,
		chatID, name,
	)
	return err
}

// ---------- User-Events ----------

// AddEventToUser привязывает событие к пользователю.
func (s *PostgresStorage) AddEventToUser(ctx context.Context, chatID, userID, eventID int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_events (chat_id, user_id, event_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		chatID, userID, eventID,
	)
	return err
}

// ---------- модель (локальная, пока нет пакета models) ----------

// Event — строка таблицы events.
type Event struct {
	ID          int64
	ChatID      int64
	Name        string
	Date        string
	Description string
	Status      string
}

// cfg := config.LoadConfig()
// store := storage.NewPostgresStorage(cfg.DatabaseURL)

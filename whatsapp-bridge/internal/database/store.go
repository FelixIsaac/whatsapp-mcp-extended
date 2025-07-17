package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// MessageStore handles database operations for storing message history and webhook configurations
type MessageStore struct {
	db *sql.DB
}

// NewMessageStore initializes a new message store with SQLite database
func NewMessageStore() (*MessageStore, error) {
	// Create directory for database if it doesn't exist
	if err := os.MkdirAll("store", 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	// Open SQLite database for messages
	db, err := sql.Open("sqlite3", "file:store/messages.db?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open message database: %v", err)
	}

	// Create tables if they don't exist
	err = createTables(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &MessageStore{db: db}, nil
}

// createTables creates all necessary database tables
func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			jid TEXT PRIMARY KEY,
			name TEXT,
			last_message_time TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT,
			chat_jid TEXT,
			sender TEXT,
			content TEXT,
			timestamp TIMESTAMP,
			is_from_me BOOLEAN,
			media_type TEXT,
			filename TEXT,
			url TEXT,
			media_key BLOB,
			file_sha256 BLOB,
			file_enc_sha256 BLOB,
			file_length INTEGER,
			PRIMARY KEY (id, chat_jid),
			FOREIGN KEY (chat_jid) REFERENCES chats(jid)
		);

		CREATE TABLE IF NOT EXISTS webhook_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			webhook_url TEXT NOT NULL,
			secret_token TEXT,
			enabled BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS webhook_triggers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			webhook_config_id INTEGER REFERENCES webhook_configs(id),
			trigger_type TEXT NOT NULL,
			trigger_value TEXT,
			match_type TEXT DEFAULT 'exact',
			enabled BOOLEAN DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS webhook_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			webhook_config_id INTEGER REFERENCES webhook_configs(id),
			message_id TEXT,
			chat_jid TEXT,
			trigger_type TEXT,
			trigger_value TEXT,
			payload TEXT,
			response_status INTEGER,
			response_body TEXT,
			attempt_count INTEGER DEFAULT 1,
			delivered_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// Close the database connection
func (store *MessageStore) Close() error {
	return store.db.Close()
}

// GetDB returns the underlying database connection for direct access
func (store *MessageStore) GetDB() *sql.DB {
	return store.db
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

// WhitelistedJID represents a whitelisted JID
type WhitelistedJID struct {
	JID    string `json:"jid"`
	Label  string `json:"label"`
	Created string `json:"created_at"`
}

// getWhitelistedJIDs returns all whitelisted JIDs
func getWhitelistedJIDs(db *sql.DB) ([]WhitelistedJID, error) {
	rows, err := db.Query("SELECT jid, label, created_at FROM allowed_jids")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var whitelistedJIDs []WhitelistedJID
	for rows.Next() {
		var jid, label, created string
		err := rows.Scan(&jid, &label, &created)
		if err != nil {
			return nil, err
		}
		whitelistedJIDs = append(whitelistedJIDs, WhitelistedJID{JID: jid, Label: label, Created: created})
	}
	return whitelistedJIDs, nil
}

// addWhitelistedJID adds a new whitelisted JID
func addWhitelistedJID(db *sql.DB, jid, label string) error {
	_, err := db.Exec("INSERT INTO allowed_jids (jid, label) VALUES (?, ?)", jid, label)
	return err
}

// removeWhitelistedJID removes a whitelisted JID
func removeWhitelistedJID(db *sql.DB, jid string) error {
	_, err := db.Exec("DELETE FROM allowed_jids WHERE jid = ?", jid)
	return err
}

// isJIDWhitelisted checks if a JID is whitelisted
func isJIDWhitelisted(db *sql.DB, jid string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM allowed_jids WHERE jid = ?", jid).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func whitelistHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./whatsapp.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	switch r.Method {
	case http.MethodGet:
		whitelistedJIDs, err := getWhitelistedJIDs(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(whitelistedJIDs)
	case http.MethodPost:
		var data struct {
			JID    string `json:"jid"`
			Label  string `json:"label"`
		}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = addWhitelistedJID(db, data.JID, data.Label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		jid := r.URL.Path[len("/api/whitelist/"):]
		err := removeWhitelistedJID(db, jid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func checkWhitelistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWhitelistEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		db, err := sql.Open("sqlite3", "./whatsapp.db")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close()

		jid := r.Header.Get("X-JID")
		if jid == "" {
			http.Error(w, "JID not provided", http.StatusBadRequest)
			return
		}

		whitelisted, err := isJIDWhitelisted(db, jid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !whitelisted {
			http.Error(w, "JID not whitelisted", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isWhitelistEnabled() bool {
	return os.Getenv("WHATSAPP_ENABLE_JID_WHITELIST") == "true"
}

func main() {
	http.HandleFunc("/api/whitelist", whitelistHandler)
	http.HandleFunc("/api/whitelist/", whitelistHandler)
	http.Handle("/send_message", checkWhitelistMiddleware(http.HandlerFunc(sendMessageHandler)))
	http.Handle("/send_file", checkWhitelistMiddleware(http.HandlerFunc(sendFileHandler)))
	http.Handle("/send_audio_message", checkWhitelistMiddleware(http.HandlerFunc(sendAudioMessageHandler)))
	http.Handle("/download_media", checkWhitelistMiddleware(http.HandlerFunc(downloadMediaHandler)))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
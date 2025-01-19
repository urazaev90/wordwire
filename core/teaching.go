// функции, связанные с обучением

package core

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

// страница обучения
func TeachingPageHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/teaching.html")
	if err != nil {
		log.Println("Error parsing template:", err)
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

// генератор json данных для страницы обучения (слова, транскрипции, перевод)
func WordsAPIHandler(w http.ResponseWriter, r *http.Request) {
	var userID int

	// проверка авторизации
	if isAuthorized(r) {
		userID = getUserIDFromSession(r) // получение userID из сессии
	} else {
		userID = 1 // значение по умолчанию для незарегистрированных пользователей
	}

	rows, err := Database.Query(`
		SELECT ew.word, ew.transcription, ew.translation
		FROM english_words ew
		INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
		WHERE uwl.user_id = $1 AND uwl.label = 2
	`, userID)
	if err != nil {
		log.Println("Error querying database:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Word struct {
		Word          string `json:"word"`
		Transcription string `json:"transcription"`
		Translation   string `json:"translation"`
	}

	var words []Word
	for rows.Next() {
		var word Word
		if err := rows.Scan(&word.Word, &word.Transcription, &word.Translation); err != nil {
			log.Println("Error scanning row:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		log.Println("Error with rows:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(words)
}

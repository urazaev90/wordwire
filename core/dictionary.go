// обработчики для работы со словами

package core

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
)

// количество слов в списках
const wordsPerPage = 10

// список всех слов
func DictionaryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		updateWordLabel(w, r)
		return
	}

	userID := getUserIDFromSession(r)

	countsChan := make(chan map[string]int)
	go getWordCountsAsync(userID, countsChan)

	var totalWords int
	dbErr := Database.QueryRow(`
		SELECT COUNT(*)
		FROM user_word_labels
		WHERE user_id = $1 AND label IN (1, 2)
	`, userID).Scan(&totalWords)
	if dbErr != nil {
		http.Error(w, "Server error: unable to count words", http.StatusInternalServerError)
		return
	}

	maxPages := (totalWords + wordsPerPage - 1) / wordsPerPage

	var page int
	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		page, dbErr = strconv.Atoi(pageParam)
		if dbErr != nil || page < 0 {
			page = 0
		}
	}

	if page >= maxPages-1 {
		_ = loadNextPageWordsForUser(userID)
		totalWords += 10
		maxPages = (totalWords + wordsPerPage - 1) / wordsPerPage
	}

	if page >= maxPages {
		page = maxPages - 1
	}
	if page < 0 {
		page = 0
	}

	offset := page * wordsPerPage

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	wordsChan := make(chan []Word)

	go func() {
		rows, queryErr := Database.Query(`
			SELECT ew.id, ew.word, uwl.label
			FROM english_words ew
			INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
			WHERE uwl.user_id = $1 AND uwl.label IN (1, 2)
			ORDER BY ew.usage_per_billion DESC
			LIMIT $2 OFFSET $3
		`, userID, wordsPerPage, offset)
		if queryErr != nil {
			log.Println("Error querying words: ", queryErr)
			wordsChan <- nil
			return
		}
		defer rows.Close()

		words := make([]Word, 0, wordsPerPage)
		for rows.Next() {
			var word Word
			if scanErr := rows.Scan(&word.ID, &word.Word, &word.Label); scanErr != nil {
				log.Println("Error scanning word: ", scanErr)
				wordsChan <- nil
				return
			}
			words = append(words, word)
		}
		wordsChan <- words
	}()

	// Wait for data to arrive
	counts := <-countsChan
	words := <-wordsChan
	if words == nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	tmpl, tmplErr := template.New("dictionary.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}).ParseFiles("templates/header.html", "templates/dictionary.html", "templates/footer.html")
	if tmplErr != nil {
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Words":         words,
		"WordCount":     totalWords,
		"Page":          page,
		"LastPage":      page == maxPages-1,
		"HasNext":       page < maxPages-1,
		"HasPrev":       page > 0,
		"FirstNumber":   page*wordsPerPage + 1,
		"CurrentURL":    r.URL.Path,
		"SelectedCount": counts["selected"],
		"ArchivedCount": counts["archived"],
	}

	if execErr := tmpl.Execute(w, data); execErr != nil {
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
}

// список избранных слов для обучения (label 2)
func SelectedHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		updateWordLabel(w, r)
		return
	}

	userID := getUserIDFromSession(r)

	countsChan := make(chan map[string]int)
	go getWordCountsAsync(userID, countsChan)

	var totalWords int
	dbErr := Database.QueryRow(`
        SELECT COUNT(*)
        FROM user_word_labels
        WHERE user_id = $1 AND label = 2
	`, userID).Scan(&totalWords)
	if dbErr != nil {
		http.Error(w, "Server error: unable to count words", http.StatusInternalServerError)
		return
	}

	maxPages := (totalWords + wordsPerPage - 1) / wordsPerPage

	var page int
	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		page, dbErr = strconv.Atoi(pageParam)
		if dbErr != nil || page < 0 {
			page = 0
		}
	}

	if page >= maxPages {
		page = maxPages - 1
	}
	if page < 0 {
		page = 0
	}

	offset := page * wordsPerPage

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	wordsChan := make(chan []Word)

	go func() {
		rows, queryErr := Database.Query(`
			SELECT ew.id, ew.word, uwl.label
			FROM english_words ew
			INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
			WHERE uwl.user_id = $1 AND uwl.label = 2
			ORDER BY ew.usage_per_billion DESC
			LIMIT $2 OFFSET $3
		`, userID, wordsPerPage, offset)
		if queryErr != nil {
			log.Println("Error querying words: ", queryErr)
			wordsChan <- nil
			return
		}
		defer rows.Close()

		words := make([]Word, 0, wordsPerPage)
		for rows.Next() {
			var word Word
			if scanErr := rows.Scan(&word.ID, &word.Word, &word.Label); scanErr != nil {
				log.Println("Error scanning word: ", scanErr)
				wordsChan <- nil
				return
			}
			words = append(words, word)
		}
		wordsChan <- words
	}()

	// Wait for data to arrive
	counts := <-countsChan
	words := <-wordsChan
	if words == nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	tmpl, tmplErr := template.New("selected.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}).ParseFiles("templates/header.html", "templates/selected.html", "templates/footer.html")
	if tmplErr != nil {
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Words":         words,
		"WordCount":     totalWords,
		"Page":          page,
		"LastPage":      page == maxPages-1,
		"HasNext":       page < maxPages-1,
		"HasPrev":       page > 0,
		"FirstNumber":   page*wordsPerPage + 1,
		"CurrentURL":    r.URL.Path,
		"SelectedCount": counts["selected"],
		"ArchivedCount": counts["archived"],
	}

	if execErr := tmpl.Execute(w, data); execErr != nil {
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
}

// список архивированных слов (label 3)
func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		updateWordLabel(w, r)
		return
	}

	userID := getUserIDFromSession(r)

	countsChan := make(chan map[string]int)
	go getWordCountsAsync(userID, countsChan)

	var totalWords int
	dbErr := Database.QueryRow(`
        SELECT COUNT(*)
        FROM user_word_labels
        WHERE user_id = $1 AND label = 3
	`, userID).Scan(&totalWords)
	if dbErr != nil {
		http.Error(w, "Server error: unable to count words", http.StatusInternalServerError)
		return
	}

	maxPages := (totalWords + wordsPerPage - 1) / wordsPerPage

	var page int
	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		page, dbErr = strconv.Atoi(pageParam)
		if dbErr != nil || page < 0 {
			page = 0
		}
	}

	if page >= maxPages {
		page = maxPages - 1
	}
	if page < 0 {
		page = 0
	}

	offset := page * wordsPerPage

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	wordsChan := make(chan []Word)

	go func() {
		rows, queryErr := Database.Query(`
			SELECT ew.id, ew.word, uwl.label
			FROM english_words ew
			INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
			WHERE uwl.user_id = $1 AND uwl.label = 3
			ORDER BY ew.usage_per_billion DESC
			LIMIT $2 OFFSET $3
		`, userID, wordsPerPage, offset)
		if queryErr != nil {
			log.Println("Error querying words: ", queryErr)
			wordsChan <- nil
			return
		}
		defer rows.Close()

		words := make([]Word, 0, wordsPerPage)
		for rows.Next() {
			var word Word
			if scanErr := rows.Scan(&word.ID, &word.Word, &word.Label); scanErr != nil {
				log.Println("Error scanning word: ", scanErr)
				wordsChan <- nil
				return
			}
			words = append(words, word)
		}
		wordsChan <- words
	}()

	// Wait for data to arrive
	counts := <-countsChan
	words := <-wordsChan
	if words == nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	tmpl, tmplErr := template.New("archive.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}).ParseFiles("templates/header.html", "templates/archive.html", "templates/footer.html")
	if tmplErr != nil {
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Words":         words,
		"WordCount":     totalWords,
		"Page":          page,
		"LastPage":      page == maxPages-1,
		"HasNext":       page < maxPages-1,
		"HasPrev":       page > 0,
		"FirstNumber":   page*wordsPerPage + 1,
		"CurrentURL":    r.URL.Path,
		"SelectedCount": counts["selected"],
		"ArchivedCount": counts["archived"],
	}

	if execErr := tmpl.Execute(w, data); execErr != nil {
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
}

// добавить в архив слово (присвоение label 3)
func AddToArchiveHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userID := getUserIDFromSession(r)

	if r.Method == http.MethodPost {
		// Пытаемся подгрузить новое слово, если это необходимо
		_ = loadNextWordForUser(userID)

		if r.Method == http.MethodPost {
			// Это часть для обновления метки на 3
			wordID, err := strconv.Atoi(r.FormValue("archive_word_id"))
			if err != nil {
				log.Println("Invalid word ID:", err)
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			_, err = Database.Exec(`
			UPDATE user_word_labels SET label = 3
			WHERE user_id = $1 AND word_id = $2
		`, userID, wordID)
			if err != nil {
				log.Println("Error updating label to archive:", err)
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}
		}
	}
}

// убрать из архива слово (присвоение label 1)
func RemoveFromArchiveHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userID := getUserIDFromSession(r)

	if r.Method == http.MethodPost {
		// Это часть для обновления метки на 3
		wordID, err := strconv.Atoi(r.FormValue("archive_word_id"))
		if err != nil {
			log.Println("Invalid word ID:", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		_, err = Database.Exec(`
			UPDATE user_word_labels SET label = 1
			WHERE user_id = $1 AND word_id = $2
		`, userID, wordID)
		if err != nil {
			log.Println("Error updating label to archive:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
	}

}

// изменения label на 1 или 2 (при удалении и установки галочки в чекбоксах списков)
func updateWordLabel(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromSession(r)

	wordID, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		log.Println("Invalid word ID:", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	label, err := strconv.Atoi(r.FormValue("label"))
	if err != nil || (label != 1 && label != 2) {
		log.Println("Invalid label value:", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err = Database.Exec(`
		UPDATE user_word_labels SET label = $1
		WHERE user_id = $2 AND word_id = $3
	`, label, userID, wordID)
	if err != nil {
		log.Println("Error updating word label:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusNoContent)
}

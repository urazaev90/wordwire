package main

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	Database *sql.DB
	store    = sessions.NewCookieStore([]byte("your-secret-key"))
)

const wordsPerPage = 10 // Количество слов в списках

func main() {
	connStr := "user=urazaev90 password=Grr(-87He dbname=app_database sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	Database = db

	router := mux.NewRouter()

	router.HandleFunc("/", RegisterHandler).Methods("GET", "POST")
	router.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", LogoutHandler).Methods("POST")
	router.HandleFunc("/setting", SettingHandler).Methods("GET", "POST")
	router.HandleFunc("/selected", SelectedHandler).Methods("GET", "POST")
	router.HandleFunc("/archive", ArchiveHandler).Methods("GET", "POST")
	router.HandleFunc("/remove_from_archive", RemoveFromArchiveHandler).Methods("GET", "POST")
	router.HandleFunc("/add_to_archive", AddToArchiveHandler).Methods("GET", "POST")
	router.HandleFunc("/teaching", TeachingPageHandler).Methods("GET")
	router.HandleFunc("/api/words", WordsAPIHandler).Methods("GET")

	log.Println("Server started at :8080")
	http.ListenAndServe(":8080", router)
}

func isAuthorized(r *http.Request) bool {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int)
	if ok {
		updateLastVisitDate(userID)
	}
	return ok
} //сообщает статус посетителя на авторизацию

func updateLastVisitDate(userID int) { //записывает дату последнего посещения пользователя в SQL
	_, err := Database.Exec(`
		UPDATE user_accounts 
		SET last_visit_date = $1 
		WHERE id = $2`,
		time.Now().Format("2006-01-02"), userID)
	if err != nil {
		log.Println("Error updating last visit date:", err)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		// Вставляем код из renderRegisterTemplate
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Cannot parse template", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Println("Register: Error parsing form:", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		login := r.FormValue("login")
		password := r.FormValue("password")

		var existingUser string
		err = Database.QueryRow("SELECT login FROM user_accounts WHERE login=$1", login).Scan(&existingUser)
		if err == nil {
			log.Println("Register: User already exists")
			// Вставляем код из renderRegisterTemplate
			tmpl, err := template.ParseFiles("templates/index.html")
			if err != nil {
				http.Error(w, "Cannot parse template", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, map[string]string{"Error": "Извините, такой логин занят, придумайте другой"})
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Register: Error hashing password:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		var userID int
		err = Database.QueryRow(` 
			INSERT INTO user_accounts (login, password_hash, registration_date) 
			VALUES ($1, $2, $3) RETURNING id`,
			login, passwordHash, time.Now().Format("2006-01-02")).Scan(&userID)
		if err != nil {
			log.Println("Register: Error inserting user to database:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		_, err = Database.Exec(`
			INSERT INTO user_word_labels (user_id, word_id, label)
			SELECT $1, id, CASE 
				WHEN id IN (31, 32, 33, 34, 35, 36, 37, 38, 39, 40) THEN 2
				ELSE 1
			END
			FROM english_words
		`, userID)
		if err != nil {
			log.Println("Register: Error inserting labels:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		session, _ := store.Get(r, "session-name")
		session.Values["user_id"] = userID
		err = session.Save(r, w)
		if err != nil {
			log.Println("Register: Error saving session:", err)
			http.Error(w, "Cannot save session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
	}
} // 1 Регистрация нового пользователя (главная стр.)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
		return
	}

	// Функция для рендеринга шаблона встроена
	renderTemplate := func(data interface{}) {
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			http.Error(w, "Cannot parse template", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
	}

	if r.Method == http.MethodGet {
		renderTemplate(nil)
	} else if r.Method == http.MethodPost {
		r.ParseForm()
		login := r.FormValue("login")
		password := r.FormValue("password")

		var dbPasswordHash string
		var userID int
		err := Database.QueryRow("SELECT id, password_hash FROM user_accounts WHERE login=$1", login).Scan(&userID, &dbPasswordHash)
		if err != nil {
			renderTemplate(map[string]string{"Error": "Такого логина не существует"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(password))
		if err != nil {
			renderTemplate(map[string]string{"Error": "Неверный пароль"})
			return
		}

		session, _ := store.Get(r, "session-name")
		session.Values["user_id"] = userID
		session.Save(r, w)

		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
	}
} // 1 Вход в аккаунт

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	delete(session.Values, "user_id")
	session.Save(r, w)

	// Изменяем перенаправление на "/"
	http.Redirect(w, r, "/", http.StatusSeeOther)
} //1 Выйти из аккаунта

func getUserIDFromSession(r *http.Request) int {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return 0
	}
	return userID
} //вещание id авторизированного пользователя

func SettingHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		updateWordLabel(w, r)
		return
	}

	userID := getUserIDFromSession(r)

	selectedCount, archivedCount, err := getWordCounts(userID)
	if err != nil {
		http.Error(w, "Server error: unable to count words", http.StatusInternalServerError)
		return
	}

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

	if page >= maxPages {
		page = maxPages - 1
	}
	if page < 0 {
		page = 0
	}

	offset := page * wordsPerPage

	rows, queryErr := Database.Query(`
        SELECT ew.id, ew.word, uwl.label
        FROM english_words ew
        INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
        WHERE uwl.user_id = $1 AND uwl.label IN (1, 2)
        ORDER BY ew.usage_per_billion DESC
        LIMIT $2 OFFSET $3
    `, userID, wordsPerPage, offset)
	if queryErr != nil {
		http.Error(w, "Server error: unable to fetch words", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	words := make([]Word, 0, wordsPerPage)
	for rows.Next() {
		var word Word
		if scanErr := rows.Scan(&word.ID, &word.Word, &word.Label); scanErr != nil {
			http.Error(w, "Server error: unable to scan words", http.StatusInternalServerError)
			return
		}
		words = append(words, word)
	}

	if rowErr := rows.Err(); rowErr != nil {
		http.Error(w, "Server error: problems with rows", http.StatusInternalServerError)
		return
	}

	tmpl, tmplErr := template.New("setting.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}).ParseFiles("templates/header.html", "templates/setting.html", "templates/footer.html")
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
		"SelectedCount": selectedCount,
		"ArchivedCount": archivedCount,
	}

	if execErr := tmpl.Execute(w, data); execErr != nil {
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
} //1 Список всех слов

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

	selectedCount, archivedCount, err := getWordCounts(userID)
	if err != nil {
		http.Error(w, "Server error: unable to count words", http.StatusInternalServerError)
		return
	}

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

	rows, queryErr := Database.Query(`
        SELECT ew.id, ew.word, uwl.label
        FROM english_words ew
        INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
        WHERE uwl.user_id = $1 AND uwl.label = 2
        ORDER BY ew.usage_per_billion DESC
        LIMIT $2 OFFSET $3
    `, userID, wordsPerPage, offset)
	if queryErr != nil {
		http.Error(w, "Server error: unable to fetch words", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	words := make([]Word, 0, wordsPerPage)
	for rows.Next() {
		var word Word
		if scanErr := rows.Scan(&word.ID, &word.Word, &word.Label); scanErr != nil {
			http.Error(w, "Server error: unable to scan words", http.StatusInternalServerError)
			return
		}
		words = append(words, word)
	}

	if rowErr := rows.Err(); rowErr != nil {
		http.Error(w, "Server error: problems with rows", http.StatusInternalServerError)
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
		"SelectedCount": selectedCount,
		"ArchivedCount": archivedCount,
	}

	if execErr := tmpl.Execute(w, data); execErr != nil {
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
} //1 Список избранных слов для обучения (label 2)

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

	selectedCount, archivedCount, err := getWordCounts(userID)
	if err != nil {
		http.Error(w, "Server error: unable to count words", http.StatusInternalServerError)
		return
	}

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

	rows, queryErr := Database.Query(`
        SELECT ew.id, ew.word, uwl.label
        FROM english_words ew
        INNER JOIN user_word_labels uwl ON ew.id = uwl.word_id
        WHERE uwl.user_id = $1 AND uwl.label = 3
        ORDER BY ew.usage_per_billion DESC
        LIMIT $2 OFFSET $3
    `, userID, wordsPerPage, offset)
	if queryErr != nil {
		http.Error(w, "Server error: unable to fetch words", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	words := make([]Word, 0, wordsPerPage)
	for rows.Next() {
		var word Word
		if scanErr := rows.Scan(&word.ID, &word.Word, &word.Label); scanErr != nil {
			http.Error(w, "Server error: unable to scan words", http.StatusInternalServerError)
			return
		}
		words = append(words, word)
	}

	if rowErr := rows.Err(); rowErr != nil {
		http.Error(w, "Server error: problems with rows", http.StatusInternalServerError)
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
		"SelectedCount": selectedCount,
		"ArchivedCount": archivedCount,
	}

	if execErr := tmpl.Execute(w, data); execErr != nil {
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
} //1 Список архивированных слов (label 3)

func getWordCounts(userID int) (int, int, error) {
	var selectedCount, archivedCount int

	err := Database.QueryRow(`
		SELECT COUNT(*)
		FROM user_word_labels
		WHERE user_id = $1 AND label = 2
	`, userID).Scan(&selectedCount)
	if err != nil {
		return 0, 0, err
	}

	err = Database.QueryRow(`
		SELECT COUNT(*)
		FROM user_word_labels
		WHERE user_id = $1 AND label = 3
	`, userID).Scan(&archivedCount)
	if err != nil {
		return 0, 0, err
	}

	return selectedCount, archivedCount, nil
} //считает сколько у пользователя слов с label 2, с label 3

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
} // Изменения label на 1 или 2 (при удалении и установки галочки в чекбоксах списков)

func AddToArchiveHandler(w http.ResponseWriter, r *http.Request) {

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
			UPDATE user_word_labels SET label = 3
			WHERE user_id = $1 AND word_id = $2
		`, userID, wordID)
		if err != nil {
			log.Println("Error updating label to archive:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

	}
} //2 Добавить в архив слово (присвоение label 3)

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

} //2 Убрать из архива слово (присвоение label 1)

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
} //1 Страница обучения

func WordsAPIHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := getUserIDFromSession(r)

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
} //генератор json данных для страницы обучения (слова, транскрипции, перевод)

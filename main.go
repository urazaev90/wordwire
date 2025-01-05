package main

import (
	"database/sql"
	"encoding/json"
	"github.com/dchest/captcha"
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
	router.HandleFunc("/demonstration", DemonstrationTeachingPageHandler).Methods("GET")
	router.HandleFunc("/api/words", WordsAPIHandler).Methods("GET")
	router.PathPrefix("/images/").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir("images/"))))
	router.PathPrefix("/sounds/").Handler(http.StripPrefix("/sounds/", http.FileServer(http.Dir("sounds/"))))
	router.HandleFunc("/api/get_user_login", GetUserLoginHandler).Methods("GET")
	router.HandleFunc("/check-login", CheckLoginHandler).Methods("GET", "POST")
	router.Handle("/captcha/{captchaID}.png", captcha.Server(captcha.StdWidth, captcha.StdHeight))
	router.HandleFunc("/generate-captcha", GenerateCaptchaHandler).Methods("GET")

	log.Println("Server started at :8080")
	http.ListenAndServe(":8080", router)
}

func CheckLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Login string `json:"login"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var existingUser string
	err = Database.QueryRow("SELECT login FROM user_accounts WHERE login=$1", data.Login).Scan(&existingUser)
	if err == nil {
		// Логин занят
		json.NewEncoder(w).Encode(map[string]bool{"isTaken": true})
		return
	}

	if err == sql.ErrNoRows {
		// Логин свободен
		json.NewEncoder(w).Encode(map[string]bool{"isTaken": false})
		return
	}

	http.Error(w, "Database error", http.StatusInternalServerError)
} //проверка при регистрации не занят ли логин во всплывающем окне в демонстрационной странице

func RestartSoundHandler(w http.ResponseWriter, r *http.Request) {
	// Просто возвращаем статус 200 для подтверждения нажатия
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func isAuthorized(r *http.Request) bool {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int)
	if ok {
		updateLastVisitDate(userID)
	}
	return ok
} //сообщает статус посетителя на авторизацию

func updateLastVisitDate(userID int) {
	_, err := Database.Exec(`
		UPDATE user_accounts 
		SET last_visit_date = $1 
		WHERE id = $2`,
		time.Now().Format("2006-01-02"), userID)
	if err != nil {
		log.Println("Error updating last visit date:", err)
	}
} //записывает дату последнего посещения пользователя (в SQL)

func GenerateCaptchaHandler(w http.ResponseWriter, r *http.Request) {
	captchaID := captcha.New() // Создаем новую капчу
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"captchaID":  captchaID,                        // Отправляем клиенту ID капчи
		"captchaURL": "/captcha/" + captchaID + ".png", // Ссылка на изображение капчи
	})
}

func VerifyCaptcha(captchaID, captchaValue string) bool {
	return captcha.VerifyString(captchaID, captchaValue) // Проверяем правильность введенного значения
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, авторизован ли пользователь
	session, _ := store.Get(r, "session-name")

	if _, ok := session.Values["username"].(string); ok {
		// Если пользователь уже авторизован, перенаправляем его в профиль
		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet { // Здесь мы ему говорим: "Если запрос GET"
		tmpl, err := template.ParseFiles("templates/index.html") // Загружаем шаблон
		if err != nil {
			log.Printf("Error parsing template: %v", err)                          // Если ошибка, пишем в лог
			http.Error(w, "Internal Server Error", http.StatusInternalServerError) // Отправляем клиенту сообщение об ошибке
			return
		}
		tmpl.Execute(w, nil) // Отправляем шаблон клиенту
		return
	}

	// Если запрос POST
	username := r.FormValue("username") // Получаем значение поля "username"
	password := r.FormValue("password") // Получаем значение поля "password"
	captchaID := r.FormValue("captchaID")
	captchaValue := r.FormValue("captchaValue")

	// Проверяем капчу
	if !VerifyCaptcha(captchaID, captchaValue) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"error": "Число с картинки введено не правильно!"})
		return
	}

	// Проверяем, существует ли пользователь
	var exists bool
	err := Database.QueryRow("SELECT EXISTS (SELECT 1 FROM user_accounts WHERE login = $1)", username).Scan(&exists)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)                   // Если ошибка, пишем в лог
		http.Error(w, "Internal Server Error", http.StatusInternalServerError) // Сообщаем об ошибке
		return
	}

	if exists {
		// Возвращаем ошибку в формате JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"error": "Извините, такой логин занят, придумайте другой"})
		return
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)                          // Если ошибка, пишем в лог
		http.Error(w, "Internal Server Error", http.StatusInternalServerError) // Сообщаем об ошибке
		return
	}

	// Сохраняем нового пользователя в базе
	var userID int
	err = Database.QueryRow(` 
			INSERT INTO user_accounts (login, password_hash, registration_date) 
			VALUES ($1, $2, $3) RETURNING id`,
		username, hashedPassword, time.Now().Format("2006-01-02")).Scan(&userID)
	if err != nil {
		log.Println("Register: Error inserting user to database:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = Database.Exec(`
    		WITH ranked_words AS (
        	SELECT id,
               ROW_NUMBER() OVER (ORDER BY usage_per_billion DESC) as rank
        	FROM english_words
    		)
    		INSERT INTO user_word_labels (user_id, word_id, label)
    		SELECT $1, id, CASE WHEN rank <= 5 THEN 2 ELSE 1 END
    		FROM ranked_words
    		WHERE rank <= 10
			`, userID)
	if err != nil {
		log.Println("Register: Error inserting labels:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = userID
	err = session.Save(r, w)
	if err != nil {
		log.Println("Register: Error saving session:", err)
		http.Error(w, "Cannot save session", http.StatusInternalServerError)
		return
	}

	// Как только регистрация прошла успешно, возвращаем успешный JSON-ответ, чтобы там javascript обработал эту команду
	// о успешной регистрации и делал что ему там дальше велено
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
} // 1 Регистрация нового пользователя (главная стр.)

func loadNextWordForUser(userID int) error {
	// Выполняем запрос для добавления следующего слова
	_, err := Database.Exec(`
        INSERT INTO user_word_labels (user_id, word_id, label)
        SELECT $1, id, 1
        FROM english_words
        WHERE id NOT IN (SELECT word_id FROM user_word_labels WHERE user_id = $1)
        ORDER BY usage_per_billion DESC
        LIMIT 1
    `, userID)

	return err
}

func loadNextPageWordsForUser(userID int) error {
	// Выполняем запрос для добавления 10 следующих слов
	_, err := Database.Exec(`
        INSERT INTO user_word_labels (user_id, word_id, label)
        SELECT $1, id, 1
        FROM english_words
        WHERE id NOT IN (SELECT word_id FROM user_word_labels WHERE user_id = $1)
        ORDER BY usage_per_billion DESC
        LIMIT 10
    `, userID)

	return err
}

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

	if page >= maxPages-1 {
		err = loadNextPageWordsForUser(userID)
		if err != nil {
			log.Println("Error loading next page words:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		// Обновляем общее количество слов после подгрузки
		totalWords += 10
		// Обновляем количество страниц
		maxPages = (totalWords + wordsPerPage - 1) / wordsPerPage
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

func DemonstrationTeachingPageHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
		return
	}

	// Статические данные для демонстрации
	demonstrationWords := []map[string]string{
		{"word": "good", "transcription": "[gud]", "translation": "хороший; добро"},
		{"word": "long", "transcription": "[lɔŋ]", "translation": "длинный; долго"},
		{"word": "night", "transcription": "[nait]", "translation": "ночь; вечер"},
		{"word": "room", "transcription": "[ru:m]", "translation": "комната"},
		{"word": "place", "transcription": "[pleis]", "translation": "место; помещать"},
	}

	// Рендеринг HTML страницы
	tmpl, err := template.ParseFiles("templates/demonstration.html")
	if err != nil {
		log.Println("Error parsing template:", err)
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}

	// Отправляем HTML и встраиваем статические данные в JS
	tmpl.Execute(w, struct {
		Words []map[string]string
	}{
		Words: demonstrationWords,
	})
} //1 Демонстративная страница обучения

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

func GetUserLoginHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := getUserIDFromSession(r)

	var login string
	err := Database.QueryRow("SELECT login FROM user_accounts WHERE id=$1", userID).Scan(&login)
	if err != nil {
		log.Println("Error fetching user login:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"login": login}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

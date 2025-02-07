// функции, связанные с авторизацией

package core

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"time"
)

// карта для отслеживания попыток авторизации
var loginAttempts = make(map[string]int)

// вещание id авторизированного пользователя
func getUserIDFromSession(r *http.Request) int {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return 0
	}
	return userID
}

// обработчик проверяющий авторизирован ли посетитель или нет
func isAuthorized(r *http.Request) bool {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int)
	if ok {
		updateLastVisitDate(userID)
	}
	return ok
}

// регистрация нового пользователя
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, авторизован ли пользователь
	session, _ := store.Get(r, "session-name")

	if _, ok := session.Values["user_id"].(int); ok {
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

	// Проверка, чтобы пароль был длинной не менее 4 символа
	if len(password) < 4 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"error": "Минимальная длинна пароля - 4 символа!"})
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
}

// авторизация
func LoginHandler(w http.ResponseWriter, r *http.Request) {

	session, _ := store.Get(r, "session-name")

	if _, ok := session.Values["user_id"].(int); ok {
		// Если пользователь уже авторизован, перенаправляем его в профиль
		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		clientIP := r.RemoteAddr
		showCaptcha := false

		if loginAttempts[clientIP] >= 3 {
			showCaptcha = true
		}

		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			log.Printf("Error parsing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]bool{"ShowCaptcha": showCaptcha})
		return
	}

	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			log.Printf("Error parsing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		captchaID := r.FormValue("captchaID")
		captchaValue := r.FormValue("captchaValue")
		clientIP := r.RemoteAddr

		// Проверяем капчу, если нужно
		if loginAttempts[clientIP] >= 3 {
			if !VerifyCaptcha(captchaID, captchaValue) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"error": "Число с картинки введено не правильно!", "showCaptcha": "true"})
				return
			}
		}

		var storedHashedPassword string
		var userID int
		err := Database.QueryRow("SELECT id, password_hash FROM user_accounts WHERE login = $1", username).Scan(&userID, &storedHashedPassword)
		if err != nil {
			log.Printf("Login: Error fetching user: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			showCaptcha := "false"
			if loginAttempts[clientIP] >= 2 {
				showCaptcha = "true"
			}
			json.NewEncoder(w).Encode(map[string]string{"error": "Неверный логин или пароль", "showCaptcha": showCaptcha})
			loginAttempts[clientIP]++
			return
		}

		// Проверяем пароль
		if bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(password)) != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			showCaptcha := "false"
			if loginAttempts[clientIP] >= 2 {
				showCaptcha = "true"
			}
			json.NewEncoder(w).Encode(map[string]string{"error": "Неверный логин или пароль", "showCaptcha": showCaptcha})
			loginAttempts[clientIP]++
			return
		}

		// Сбрасываем счетчик попыток при успешном входе
		loginAttempts[clientIP] = 0

		// Создаем сессию

		session.Values["user_id"] = userID
		session.Save(r, w)

		// Успешный ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"success": "true"})
	}
}

// деавторизация
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	delete(session.Values, "user_id")
	session.Save(r, w)

	// Изменяем перенаправление на "/"
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

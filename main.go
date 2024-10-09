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
)

var (
	Database *sql.DB
	store    = sessions.NewCookieStore([]byte("your-secret-key"))
)

func main() {
	connStr := "user=urazaev90 password=Grr(-87He dbname=app_database sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	Database = db

	router := mux.NewRouter()
	router.HandleFunc("/", RedirectHandler).Methods("GET")
	router.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	router.HandleFunc("/register", RegisterHandler).Methods("GET", "POST")
	router.HandleFunc("/profile", ProfileHandler).Methods("GET", "POST")
	router.HandleFunc("/teaching", TeachingPageHandler).Methods("GET") // Render HTML
	router.HandleFunc("/api/words", WordsAPIHandler).Methods("GET")    // Return JSON data
	router.HandleFunc("/logout", LogoutHandler).Methods("POST")

	log.Println("Server started at :8080")
	http.ListenAndServe(":8080", router)
}

func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		renderLoginTemplate(w, nil)
	} else if r.Method == http.MethodPost {
		r.ParseForm()
		login := r.FormValue("login")
		password := r.FormValue("password")

		var dbPasswordHash string
		var userID int
		err := Database.QueryRow("SELECT id, password_hash FROM user_accounts WHERE login=$1", login).Scan(&userID, &dbPasswordHash)
		if err != nil {
			renderLoginTemplate(w, map[string]string{"Error": "Такого логина не существует"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(password))
		if err != nil {
			renderLoginTemplate(w, map[string]string{"Error": "Неверный пароль"})
			return
		}

		session, _ := store.Get(r, "session-name")
		session.Values["user_id"] = userID
		session.Save(r, w)

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}

func renderLoginTemplate(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderRegisterTemplate(w, nil)
	} else if r.Method == http.MethodPost {
		r.ParseForm()
		login := r.FormValue("login")
		password := r.FormValue("password")

		var existingUser string
		err := Database.QueryRow("SELECT login FROM user_accounts WHERE login=$1", login).Scan(&existingUser)
		if err == nil {
			renderRegisterTemplate(w, map[string]string{"Error": "Извините, такой логин занят, придумайте другой"})
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		var userID int
		err = Database.QueryRow("INSERT INTO user_accounts (login, password_hash) VALUES ($1, $2) RETURNING id", login, passwordHash).Scan(&userID)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		_, err = Database.Exec(`
			INSERT INTO user_word_labels (user_id, word_id, label)
			SELECT $1, id, 1 FROM english_words
		`, userID)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func renderRegisterTemplate(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		userID := getUserIDFromSession(r)

		tx, err := Database.Begin()
		if err != nil {
			log.Println("Error starting transaction:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		for key, values := range r.Form {
			if len(values) == 0 {
				continue
			}

			label, err := strconv.Atoi(values[0])
			if err != nil {
				log.Println("Error converting label:", err)
				continue
			}

			_, err = tx.Exec(`UPDATE user_word_labels SET label = $1 WHERE user_id = $2 AND word_id = $3`, label, userID, key)
			if err != nil {
				tx.Rollback()
				log.Println("Error updating label:", err)
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			log.Println("Error committing transaction:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	userID := getUserIDFromSession(r)
	rows, err := Database.Query(`
		SELECT ew.id, ew.word, uwl.label
		FROM english_words ew
		LEFT JOIN user_word_labels uwl ON ew.id = uwl.word_id AND uwl.user_id = $1
		ORDER BY ew.usage_per_billion DESC
	`, userID)
	if err != nil {
		log.Println("Error querying database:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Word struct {
		ID    int
		Word  string
		Label int
	}

	var words []Word
	for rows.Next() {
		var word Word
		var label sql.NullInt64
		if err := rows.Scan(&word.ID, &word.Word, &label); err != nil {
			log.Println("Error scanning row:", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		if label.Valid {
			word.Label = int(label.Int64)
		} else {
			word.Label = 1 // Use 1 if NULL
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		log.Println("Error with rows:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		log.Println("Error parsing template:", err)
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, words)
	if err != nil {
		log.Println("Error executing template:", err)
		http.Error(w, "Cannot execute template", http.StatusInternalServerError)
	}
}

func getUserIDFromSession(r *http.Request) int {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return 0
	}
	return userID
}

func isAuthorized(r *http.Request) bool {
	session, _ := store.Get(r, "session-name")
	_, ok := session.Values["user_id"].(int)
	return ok
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	delete(session.Values, "user_id")
	session.Save(r, w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func TeachingPageHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
}

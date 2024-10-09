package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

var Database *sql.DB

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
		err := Database.QueryRow("SELECT password_hash FROM user_accounts WHERE login=$1", login).Scan(&dbPasswordHash)
		if err != nil {
			renderLoginTemplate(w, map[string]string{"Error": "Такого логина не существует"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(password))
		if err != nil {
			renderLoginTemplate(w, map[string]string{"Error": "Неверный пароль"})
			return
		}

		// Установить куку сессии
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "example-session-token",
			Expires: time.Now().Add(1 * time.Hour),
		})

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
		tmpl, err := template.ParseFiles("templates/register.html")
		if err != nil {
			http.Error(w, "Cannot parse template", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	} else if r.Method == http.MethodPost {
		r.ParseForm()
		login := r.FormValue("login")
		password := r.FormValue("password")

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		var userID int
		err = Database.QueryRow("INSERT INTO user_accounts (login, password_hash) VALUES ($1, $2) RETURNING id", login, passwordHash).Scan(&userID)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		_, err = Database.Exec(`
            INSERT INTO user_word_labels (user_id, word_id, label)
            SELECT $1, id, 1 FROM english_words
        `, userID)
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
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

			wordID, err := strconv.Atoi(key)
			if err != nil {
				log.Println("Error converting wordID:", err)
				continue
			}
			label, err := strconv.Atoi(values[0])
			if err != nil {
				log.Println("Error converting label:", err)
				continue
			}

			_, err = tx.Exec(`UPDATE user_word_labels SET label = $1 WHERE user_id = $2 AND word_id = $3`, label, userID, wordID)
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
			word.Label = 1 // Временно устанавливаем 1, если метка NULL
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
	// В реальном приложении здесь будет логика получения userID на основе куки сессии
	return 1 // Для примера возвращаем user_id = 1
}

func isAuthorized(r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value != "example-session-token" {
		return false
	}
	return true
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

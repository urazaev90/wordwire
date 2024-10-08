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
	router.HandleFunc("/profile", ProfileHandler).Methods("GET")
	router.HandleFunc("/logout", LogoutHandler).Methods("POST")

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

		_, err = Database.Exec("INSERT INTO user_accounts (login, password_hash) VALUES ($1, $2)", login, passwordHash)
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

	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		http.Error(w, "Cannot parse template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func isAuthorized(r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value != "example-session-token" {
		return false
	}
	return true
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Удалить куку сессии
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

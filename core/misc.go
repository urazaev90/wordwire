// обработчики для вспомогательных функций

package core

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// обработчик при переходах на несуществующие ссылки
func CustomNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		http.Redirect(w, r, "/teaching", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// проверка при регистрации не занят ли логин во всплывающем окне в демонстрационной странице
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
}

// узнать свой логин клиенту
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

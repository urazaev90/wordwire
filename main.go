package main

import (
	"database/sql"
	"github.com/dchest/captcha"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"projectWordwire/core"
)

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

	core.Database = db

	router := mux.NewRouter()

	router.PathPrefix("/static/css/").Handler(http.StripPrefix("/static/css/", http.FileServer(http.Dir("static/css/"))))
	router.PathPrefix("/static/js/").Handler(http.StripPrefix("/static/js/", http.FileServer(http.Dir("static/js/"))))
	router.PathPrefix("/static/images/").Handler(http.StripPrefix("/static/images/", http.FileServer(http.Dir("static/images/"))))
	router.PathPrefix("/static/sounds/").Handler(http.StripPrefix("/static/sounds/", http.FileServer(http.Dir("static/sounds/"))))

	router.NotFoundHandler = http.HandlerFunc(core.CustomNotFoundHandler)

	router.Handle("/captcha/{captchaID}.png", captcha.Server(captcha.StdWidth, captcha.StdHeight))

	router.HandleFunc("/", core.RegisterHandler).Methods("GET", "POST")
	router.HandleFunc("/login", core.LoginHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", core.LogoutHandler).Methods("POST")
	router.HandleFunc("/dictionary", core.DictionaryHandler).Methods("GET", "POST")
	router.HandleFunc("/selected", core.SelectedHandler).Methods("GET", "POST")
	router.HandleFunc("/archive", core.ArchiveHandler).Methods("GET", "POST")
	router.HandleFunc("/remove_from_archive", core.RemoveFromArchiveHandler).Methods("GET", "POST")
	router.HandleFunc("/add_to_archive", core.AddToArchiveHandler).Methods("GET", "POST")
	router.HandleFunc("/teaching", core.TeachingPageHandler).Methods("GET")
	router.HandleFunc("/api/words", core.WordsAPIHandler).Methods("GET")
	router.HandleFunc("/api/get_user_login", core.GetUserLoginHandler).Methods("GET")
	router.HandleFunc("/check-login", core.CheckLoginHandler).Methods("GET", "POST")
	router.HandleFunc("/generate-captcha", core.GenerateCaptchaHandler).Methods("GET")

	router.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "robots.txt")
	}).Methods("GET")

	router.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "sitemap.xml")
	}).Methods("GET")

	router.HandleFunc("/demonstration", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/demonstration.html"))
		tmpl.Execute(w, nil)
	})

	router.HandleFunc("/developer", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/developer.html"))
		tmpl.Execute(w, nil)
	})

	log.Println("Server started at :8081")
	http.ListenAndServe(":8081", router)
}

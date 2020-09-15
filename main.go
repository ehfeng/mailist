package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
)

const RecaptchaSiteVerifyUrl = "https://www.google.com/recaptcha/api/siteverify"

var forbiddenNames = map[string]bool{"lists": true, "login": true, "static": true}

func (s *server) isAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("t")
	if err != nil {
		return false
	}
	return cookie.Value == s.config.adminPassword
}

func favicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "logo.png")
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == s.config.adminPassword {
		http.SetCookie(w, &http.Cookie{Name: "t", Value: token, MaxAge: 365 * 24 * 60 * 60, Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	return
}

var indexTemplate = template.Must(template.ParseFiles("index.tmpl"))

type IndexTemplateArgs struct {
	ListNames []string
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	rows, err := s.conn.Query("select name from lists")
	if err != nil {
		panic(err)
	}

	listNames := []string{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			panic(err)
		}
		listNames = append(listNames, name)
	}
	args := IndexTemplateArgs{listNames}
	indexTemplate.Execute(w, args)
	return
}

func (s *server) lists(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		name := r.PostForm.Get("name")
		if name == "" || forbiddenNames[name] {
			log.Println("Invalid list name", name)
			return
		}
		_, err := s.conn.Exec("INSERT INTO lists (name) values ($1)", name)
		if err != nil {
			panic(err)
		}
		log.Println("Created list", name)

		http.Redirect(w, r, "/", http.StatusFound)
		return

	} else if r.Method == "DELETE" {
		name := r.URL.Query().Get("name")
		log.Println("Preparing to delete list", name)
		_, err := s.conn.Exec("DELETE FROM lists WHERE name = $1", name)
		if err != nil {
			panic(err)
		}
		log.Printf("Deleted list %s", name)
		w.WriteHeader(http.StatusOK)
		return
	}
	return
}

var listTemplate = template.Must(template.ParseFiles("list.tmpl"))
var recaptchaTemplate = template.Must(template.ParseFiles("recaptcha.tmpl"))

type listTemplateArgs struct {
	Listname         string
	SubscriberEmails []string
}

type recaptchaArgs struct {
	Listname         string
	Email            string
	Next             string
	RecaptchaSiteKey string
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	listName := mux.Vars(r)["listname"]
	var listExists bool
	err := s.conn.QueryRow("SELECT true FROM lists WHERE name = $1", listName).Scan(&listExists)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		panic(err)
	}

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", s.config.corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Request-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == "POST" {
		email := r.FormValue("email")
		if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next := r.FormValue("next")
		if next == "" {
			next = r.URL.Query().Get("next")
		}
		if next == "" {
			next = "/" + listName
		}
		args := recaptchaArgs{
			Listname:         listName,
			Email:            email,
			Next:             next,
			RecaptchaSiteKey: s.config.recaptchaSiteKey,
		}
		recaptchaTemplate.Execute(w, args)
		return
	}
	if !s.isAdmin(r) {
		w.WriteHeader(403)
		return
	}

	// TODO use for create list input validation
	if r.Method == "HEAD" {
		w.WriteHeader(http.StatusFound)
		return
	}

	if r.Method == "DELETE" {
		emails := r.URL.Query()["email"]
		log.Println("Preparing to delete subscribers", emails)
		tx, err := s.conn.Begin()
		if err != nil {
			panic(err)
		}
		defer tx.Rollback()
		for _, email := range emails {
			cmdTag, err := tx.Exec("delete from subscribers where list = $1 and email = $2", listName, email)
			if err != nil {
				panic(err)
			}
			if cmdTag.RowsAffected() != 1 {
				log.Println("Subscriber does not exist", email)
			}
		}
		tx.Commit()
		log.Println("Deleted subscribers", emails)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if r.Method == "GET" && strings.Contains(r.Header.Get("Accept"), "text/html") {
		rows, err := s.conn.Query(`select email from subscribers where list = $1`, listName)
		if err != nil {
			panic(err)
		}
		subscriberEmails := []string{}
		var email string
		for rows.Next() {
			if err = rows.Scan(&email); err != nil {
				panic(err)
			}
			subscriberEmails = append(subscriberEmails, email)
		}
		args := listTemplateArgs{listName, subscriberEmails}
		listTemplate.Execute(w, args)

	}
	return
}

type RecaptchaVerifyResponse struct {
	Success bool    `json:"success"`
	Score   float32 `json:"score"`
}

func (s *server) recaptcha(w http.ResponseWriter, r *http.Request) {
	listName := mux.Vars(r)["listname"]
	email := r.URL.Query().Get("email")
	redirectUrl := r.URL.Query().Get("next")

	resp, err := http.PostForm(RecaptchaSiteVerifyUrl, url.Values{"secret": {s.config.recaptchaSecret}, "response": {r.URL.Query().Get("token")}})
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var verifyResp RecaptchaVerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		panic(err)
	}
	if verifyResp.Success {
		cmdTag, err := s.conn.Exec("insert into subscribers(list, email) values ($1, $2) on conflict do nothing", listName, email)
		if err != nil {
			panic(err)
		}
		if cmdTag.RowsAffected() != 1 {
			log.Println("Duplicate email subscriber", email)
		}
	}
	if redirectUrl == "" {
		redirectUrl = "/" + listName
	}
	http.Redirect(w, r, redirectUrl, http.StatusFound)
	return
}

type serverConfig struct {
	adminPassword    string
	corsOrigin       string
	recaptchaSiteKey string
	recaptchaSecret  string
}

type server struct {
	conn   *pgx.Conn
	config serverConfig
}

func main() {
	fmt.Println("Starting server...")
	corsOriginPtr := flag.String("cors-origin", "*", "CORS preflight origin header")
	flag.Parse()

	config := pgx.ConnConfig{Database: "mailist"}
	conn, err := pgx.Connect(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	s := server{conn, serverConfig{os.Getenv("AdminPassword"), *corsOriginPtr, os.Getenv("RecaptchaSiteKey"), os.Getenv("RecaptchaSecret")}}
	router := mux.NewRouter()

	// router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	router.HandleFunc("/", s.index)

	router.HandleFunc("/login/{token}", s.login)

	router.HandleFunc("/lists", s.lists)
	router.HandleFunc("/{listname}", s.list)
	router.HandleFunc("/{listname}/recaptcha", s.recaptcha)

	fmt.Println("Server available at localhost:9990")
	log.Fatal(http.ListenAndServe(":9990", router))
}

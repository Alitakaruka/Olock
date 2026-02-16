package server

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const sessionCookieName = "session"
const sessionDuration = 7 * 24 * time.Hour

const (
	testRoom = "123"
	testPass = "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
)

type Server struct {
	db *sql.DB
}

type sessionUser struct {
	ID    int64
	Login string
}

func (S *Server) Serve() {

	var addr = flag.String("addr", ":8080", "http service address")
	flag.Parse()

	err := http.ListenAndServe(*addr, nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (S *Server) Init() {
	S.initDB()
	hub := NewHub() //
	go hub.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "Pages/Room.html")
	})
	http.HandleFunc("/TestRoom", func(w http.ResponseWriter, r *http.Request) {
		if S.getSession(r) == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.ServeFile(w, r, "Pages/home.html")
	})
	http.HandleFunc("/ConnectRoom", S.ConnectRoom)

	http.HandleFunc("/CreateRoom", S.CreateRoom)
	http.HandleFunc("/CreateUser", S.CreateUser)
	http.HandleFunc("/Login", S.Login)
	http.HandleFunc("/Logout", S.Logout)
	http.HandleFunc("/Me", S.Me)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if S.getSession(r) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ServeWs(hub, w, r)
	})
}

func (S *Server) ConnectRoom(w http.ResponseWriter, r *http.Request) {
	if S.getSession(r) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	room := struct {
		Room     string `json:"room"`
		Password string `json:"password"`
	}{}

	buff, err := io.ReadAll(r.Body)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(buff, &room); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}

	// if err != nil {
	// 	fmt.Println(err)
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	queryRes := struct {
		name string
		id   int
	}{}

	tx, err := S.db.Begin()

	err = tx.QueryRow(`SELECT id,name 
	FROM rooms 
	WHERE 
	name = ? AND
	password =?`,
		room.Room,
		room.Password).Scan(&queryRes.name, &queryRes.id)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		tx.Rollback()
		return
	}

	tx.Commit()
	fmt.Println("redirect")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
	// http.Redirect(w, r, "/TestRoom", http.StatusSeeOther)
}

func (S *Server) CreateRoom(w http.ResponseWriter, r *http.Request) {
	if S.getSession(r) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	log.Println("New room creating!")

	requestParams := struct {
		RoomName     string `json:"roomname"`
		RoomPassword string `json:"roompassword"`
	}{}

	buffer, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buffer, &requestParams); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := S.db.Begin()

	_, err = tx.Exec(`INSERT INTO rooms (name,password) VALUES (?,?)`,
		requestParams.RoomName,
		requestParams.RoomPassword)

	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tx.Commit()

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Room created"))
}

func (S *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buf, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	login := strings.TrimSpace(req.Login)
	if login == "" {
		http.Error(w, "login required", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, "password required", http.StatusBadRequest)
		return
	}

	// userUUID := uuid.New().String()

	_, err = S.db.Exec(
		`INSERT INTO users (login, name, password) VALUES (?, ?, ?)`,
		login, login, req.Password,
	)
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "login already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("New user created! Name:%v \n", login)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"ok"}`))
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint") ||
		strings.Contains(err.Error(), "unique constraint")
}

func (S *Server) initDB() {
	var err error

	S.db, err = sql.Open("sqlite", "data/test.db?_busy_timeout=5000")

	if err != nil {
		panic(err)
	}

	_, err = S.db.Exec(`CREATE TABLE IF NOT EXISTS users(
	id INTEGER PRIMARY KEY,
	login TEXT UNIQUE,
	name TEXT NOT NULL,
	password varchar(255) NOT NULL)`)
	if err != nil {
		panic(err)
	}

	_, err = S.db.Exec(`
        CREATE TABLE IF NOT EXISTS rooms (
            id INTEGER PRIMARY KEY,
            name TEXT UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL
        )
    `)
	if err != nil {
		panic(err)
	}

	_, err = S.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		panic(err)
	}

	log.Println("Database init sucses!")
}

func (S *Server) getSession(r *http.Request) *sessionUser {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil
	}

	var userID int64
	var login string
	var expiresAt time.Time
	err = S.db.QueryRow(`
		SELECT u.id, u.login, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > ?
	`, cookie.Value, time.Now()).Scan(&userID, &login, &expiresAt)
	if err != nil {
		return nil
	}
	return &sessionUser{ID: userID, Login: login}
}

func (S *Server) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buf, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	login := strings.TrimSpace(req.Login)
	if login == "" || req.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	var userID int64
	var dbPassword string
	err = S.db.QueryRow(`SELECT id, password FROM users WHERE login = ?`, login).Scan(&userID, &dbPassword)
	if err != nil || dbPassword != req.Password {
		http.Error(w, "invalid login or password", http.StatusUnauthorized)
		return
	}

	token := uuid.New().String()
	expiresAt := time.Now().Add(sessionDuration)
	_, err = S.db.Exec(`INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)`,
		userID, token, expiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (S *Server) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, _ := r.Cookie(sessionCookieName)
	if cookie != nil && cookie.Value != "" {
		S.db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (S *Server) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := S.getSession(r)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"login": u.Login, "id": u.ID})
}

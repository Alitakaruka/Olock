package server

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

const (
	testRoom = "123"
	testPass = "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
)

type Server struct {
	db *sql.DB
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
	hub := NewHub() //
	go hub.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(r.)
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "Room.html")
	})
	http.HandleFunc("/TestRoom", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "home.html")
	})
	http.HandleFunc("/ConnectRoom", func(w http.ResponseWriter, r *http.Request) {

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

		if room.Password == testPass && room.Room == testRoom {
			log.Println("redirect")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			// http.Redirect(w, r, "/TestRoom", http.StatusSeeOther)
		} else {
			http.Error(w, "Cant find room", http.StatusUnauthorized)
		}

	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})
}

func (S *Server) initDB() {
	var err error

	S.db, err = sql.Open("sqlite", "test.db")

	if err != nil {
		panic(err)
	}

	_, err = S.db.Exec(`CREATE TABLE IF NOT EXIST users
	id INTEGER PRIMATY KEY,
	login TEXT UNIQUE KEY,
	name TEXT NOT NULL,
	password varchar(255) NOT NULL`)
	if err != nil {
		panic(err)
	}

	_, err = S.db.Exec(`
        CREATE TABLE IF NOT EXISTS rooms (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
			password VARCHAR(255) NOT NULL,
			user INTEGER NOT NULL

			FOREIGN KEY (user) REFERENCES users(id)
        )
    `)

	if err != nil {
		panic(err)
	}

	log.Println()
}

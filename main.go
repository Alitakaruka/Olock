// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	server "OBlocking/Server"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "room.html")
}

func main() {
	flag.Parse()
	hub := server.NewHub() //
	go hub.Run()
	http.HandleFunc("/", serveHome)
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

		fmt.Printf("room: %v\n", room)
		fmt.Printf("buff: %v\n", string(buff))
		log.Println(r.URL)
		http.ServeFile(w, r, "home.html")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.ServeWs(hub, w, r)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

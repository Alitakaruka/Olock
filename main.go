// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	server "OBlocking/Server"
	"io"
	"log"
	"os"
)

func main() {
	startLoger("data/Logs.log")
	server := server.Server{}
	server.Init()
	server.Serve()
}

func startLoger(filePath string) {
	file, ex := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if ex != nil {
		log.Fatal(ex)
	}
	muliWriter := io.MultiWriter(os.Stdout, file)
	log.SetFlags(log.Ltime | log.Ldate | log.Llongfile)
	log.SetOutput(muliWriter)
}

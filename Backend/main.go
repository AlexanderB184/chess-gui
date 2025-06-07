package main

import (
	"fmt"
	"log"
	http "net/http"
	"os"
)

var Bot *BotInterface = nil

func main() {
	cmdLineArgs := os.Args

	if len(cmdLineArgs) <= 1 {
		fmt.Println("Usage: ", os.Args[0], " [bot path]")
		return
	}

	botIface, err := NewBot(cmdLineArgs[1])
	Bot = botIface
	if err != nil {
		log.Print(err)
		return
	}

	http.HandleFunc("/socket/", runChess)
	http.Handle("/", http.FileServer(http.Dir("Public/")))
	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Print(err)
	}
}

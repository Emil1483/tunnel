package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsConnection *websocket.Conn
	wsResponse   string
	responseLock sync.RWMutex
)

type Message struct {
	Method        string              `json:"method"`
	TargetedRoute string              `json:"targeted_route"`
	Headers       map[string][]string `json:"headers"`
	Params        map[string][]string `json:"params"`
	Body          string              `json:"body"`
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Websocket upgrade failed:", err)
		return
	}

	wsConnection = conn

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Websocket read error:", err)
			return
		}

		responseLock.Lock()
		wsResponse = string(message)
		responseLock.Unlock()

		log.Println("Received message from websocket:", wsResponse)
	}
}

func tunnelHandler(w http.ResponseWriter, r *http.Request) {
	if wsConnection == nil {
		log.Println("No active websocket connection")
		return
	}

	// Build the message with request information
	message := Message{
		Method:        r.Method,
		TargetedRoute: r.URL.Path,
		Headers:       r.Header,
		Params:        r.URL.Query(),
	}

	err := r.ParseForm()
	if err != nil {
		log.Println("Error parsing form data:", err)
		return
	}

	message.Body = r.Form.Encode()

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message as JSON:", err)
		return
	}

	err = wsConnection.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		log.Println("Websocket write error:", err)
		return
	}

	responseLock.RLock()
	currentResponse := wsResponse
	responseLock.RUnlock()

	fmt.Fprint(w, currentResponse)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", tunnelHandler)

	log.Println("Starting server on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}

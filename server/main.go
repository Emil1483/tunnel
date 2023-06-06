package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var wsConnection *websocket.Conn

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Websocket upgrade failed:", err)
		return
	}

	wsConnection = conn

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Websocket read error:", err)
			return
		}
	}
}

func tunnelHandler(w http.ResponseWriter, r *http.Request) {
	if wsConnection == nil {
		log.Println("No active websocket connection")
		return
	}

	// Build the message with request information
	message := fmt.Sprintf("HTTP Method: %s\n\n", r.Method)
	message += fmt.Sprintf("Targeted Route: %s\n\n", r.URL.Path)

	// Request headers
	message += "Headers:\n"
	for key, values := range r.Header {
		for _, value := range values {
			message += fmt.Sprintf("%s: %s\n", key, value)
		}
	}
	message += "\n"

	// Request params
	message += "Params:\n"
	queryParams := r.URL.Query()
	for key, values := range queryParams {
		for _, value := range values {
			message += fmt.Sprintf("%s: %s\n", key, value)
		}
	}
	message += "\n"

	// Request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading request body:", err)
		return
	}
	defer r.Body.Close()

	message += "Body:\n" + string(body)

	err = wsConnection.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("Websocket write error:", err)
		return
	}

	fmt.Fprint(w, "Message sent through websocket")
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

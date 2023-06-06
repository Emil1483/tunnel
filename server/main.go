package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	wsConnection     *websocket.Conn
	wsResponse       []byte
	prevResponseTime time.Time
)

type Message struct {
	Method        string              `json:"method"`
	TargetedRoute string              `json:"targeted_route"`
	Headers       map[string][]string `json:"headers"`
	Params        map[string][]string `json:"params"`
	Body          string              `json:"body"`
}

type ResponseData struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Websocket upgrade failed:", err)
		return
	}

	wsConnection = conn
	log.Println("Established ws connection:", r.RemoteAddr)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Websocket read error:", err)
			return
		}

		wsResponse = message
		prevResponseTime = time.Now()

		log.Println("Received message from websocket:", string(message))
	}
}

func tunnelHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if wsConnection == nil {
		http.Error(w, "No active tunnel ws connections", http.StatusInternalServerError)
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

	end := time.Now().Add(time.Minute)
	for time.Now().Before(end) {
		if startTime.Before(prevResponseTime) {
			break
		}
		time.Sleep(time.Millisecond)
	}

	if time.Now().After(end) {
		http.Error(w, "Timeout Error: could not get a response after 60s", http.StatusInternalServerError)
		return
	}

	currentResponse := wsResponse

	parsedResponse := ResponseData{}
	err = json.Unmarshal(currentResponse, &parsedResponse)
	if err != nil {
		log.Println("Error parsing response:", err)
		log.Println("Responding with currentResponse")
		http.Error(w, string(currentResponse), http.StatusInternalServerError)
		return
	}

	// Set the status code and headers
	for key, values := range parsedResponse.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(parsedResponse.StatusCode)

	// Set the response body
	fmt.Fprint(w, parsedResponse.Body)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", tunnelHandler)

	log.Println("Starting server on localhost")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var (
	wsConnection     *websocket.Conn
	wsResponse       []byte
	prevResponseTime time.Time
	isRequestBusy    bool
	accessTokenHash  string
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

	if wsConnection != nil {
		log.Println("Refused ws connection. Connection already active:", r.RemoteAddr)
		conn.WriteMessage(websocket.TextMessage, []byte("WebSocket connection already active"))
		conn.Close()
		return
	}

	token := r.URL.Query().Get("token")
	if !compareAccessToken(token, accessTokenHash) {
		log.Println("Refused ws connection. Bad access token:", r.RemoteAddr)
		conn.WriteMessage(websocket.TextMessage, []byte("Unauthorized: Bad access token"))
		conn.Close()
		return
	}

	wsConnection = conn
	log.Println("Established ws connection:", r.RemoteAddr)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Websocket read error:", err)
			break
		}

		wsResponse = message
		prevResponseTime = time.Now()

		log.Println("Received message from websocket:", string(message))
	}

	wsConnection = nil
}

func tunnelHandler(w http.ResponseWriter, r *http.Request) {
	if isRequestBusy {
		http.Error(w, "Previous request is still being processed", http.StatusForbidden)
		return
	}

	isRequestBusy = true

	startTime := time.Now()

	if wsConnection == nil {
		http.Error(w, "No active tunnel ws connections", http.StatusInternalServerError)
		isRequestBusy = false
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read body", http.StatusInternalServerError)
		isRequestBusy = false
		return
	}

	bodyString := string(bodyBytes)

	// Build the message with request information
	message := Message{
		Method:        r.Method,
		TargetedRoute: r.URL.Path,
		Headers:       r.Header,
		Params:        r.URL.Query(),
		Body:          bodyString,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message as JSON:", err)
		isRequestBusy = false
		return
	}

	err = wsConnection.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		log.Println("Websocket write error:", err)
		isRequestBusy = false
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
		isRequestBusy = false
		return
	}

	currentResponse := wsResponse

	parsedResponse := ResponseData{}
	err = json.Unmarshal(currentResponse, &parsedResponse)
	if err != nil {
		log.Println("Error parsing response:", err)
		log.Println("Responding with currentResponse")
		http.Error(w, string(currentResponse), http.StatusInternalServerError)
		isRequestBusy = false
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

	isRequestBusy = false
}

func compareAccessToken(accessToken, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(accessToken))
	return err == nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file:", err)
	}

	accessToken := os.Getenv("ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("ACCESS_TOKEN environment variable is not set")
	}

	// Hash the accessToken
	hash, err := bcrypt.GenerateFromPassword([]byte(accessToken), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Error hashing accessToken:", err)
	}
	accessTokenHash = string(hash)

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", tunnelHandler)

	log.Println("Starting server on localhost")
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}

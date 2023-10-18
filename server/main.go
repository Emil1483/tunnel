package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var (
	wsConnection    *websocket.Conn
	accessTokenHash string
	responses       = map[string]ResponseData{}
)

type Message struct {
	Method        string              `json:"method"`
	TargetedRoute string              `json:"targeted_route"`
	Headers       map[string][]string `json:"headers"`
	Params        map[string][]string `json:"params"`
	Body          string              `json:"body"`
	Id            string              `json:"id"`
}

type ResponseData struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	Id         string              `json:"id"`
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Websocket upgrade failed:", err)
		return
	}

	if wsConnection != nil {
		log.Println("Connection established while interrupting another")
		wsConnection.WriteMessage(websocket.TextMessage, []byte("Connection interrupted by another client"))
		wsConnection.Close()
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

		parsedResponse := ResponseData{}
		err = json.Unmarshal(message, &parsedResponse)

		if err != nil {
			log.Println("Error parsing response. Request will timeout:", err)
			return
		}

		responses[parsedResponse.Id] = parsedResponse

		log.Println("Received message from websocket:", string(message))
	}

	wsConnection = nil
}

func tunnelHandler(w http.ResponseWriter, r *http.Request) {
	if wsConnection == nil {
		http.Error(w, "No active tunnel ws connections", http.StatusInternalServerError)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read body", http.StatusInternalServerError)
		return
	}

	bodyString := string(bodyBytes)

	currentId := uuid.New().String()

	// Build the message with request information
	message := Message{
		Method:        r.Method,
		TargetedRoute: r.URL.Path,
		Headers:       r.Header,
		Params:        r.URL.Query(),
		Body:          bodyString,
		Id:            currentId,
	}

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
		_, exists := responses[currentId]

		if exists {
			break
		}

		time.Sleep(time.Millisecond)
	}

	if time.Now().After(end) {
		http.Error(w, "Timeout Error: could not get a response after 60s", http.StatusInternalServerError)
		return
	}

	currentResponse, _ := responses[currentId]

	// Set the status code and headers
	for key, values := range currentResponse.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(currentResponse.StatusCode)

	// Set the response body
	fmt.Fprint(w, currentResponse.Body)

	delete(responses, currentId)
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

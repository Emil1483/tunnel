package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type Message struct {
	Method        string              `json:"method"`
	TargetedRoute string              `json:"targeted_route"`
	Headers       map[string][]string `json:"headers"`
	Params        map[string][]string `json:"params"`
	Body          string              `json:"body"`
}

type Response struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
}

func main() {
	// Connect to the WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("WebSocket connection error:", err)
	}
	defer conn.Close()

	log.Println("Connected to WS tunnel")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// Parse the received message
		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		// Create the HTTP request
		req, err := http.NewRequest(msg.Method, "http://localhost:3000"+msg.TargetedRoute, strings.NewReader(msg.Body))
		if err != nil {
			log.Println("Error creating request:", err)
			continue
		}

		// Set request headers
		for key, values := range msg.Headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Set query parameters
		query := req.URL.Query()
		for key, values := range msg.Params {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		req.URL.RawQuery = query.Encode()

		// Send the HTTP request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error sending request:", err)
			continue
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			continue
		}

		// Create the response message
		response := Response{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       string(body),
		}
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Println("Error encoding response message:", err)
			continue
		}

		// Write the response back to the WebSocket server
		err = conn.WriteMessage(websocket.TextMessage, responseJSON)
		if err != nil {
			log.Println("Error writing response to WebSocket:", err)
			break
		}

		log.Printf("Response Message: %s", responseJSON)
	}
}

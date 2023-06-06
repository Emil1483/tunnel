package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type Message struct {
	Method  string            `json:"method"`
	Route   string            `json:"route"`
	Headers map[string]string `json:"headers"`
	Params  map[string]string `json:"params"`
	Body    map[string]string `json:"body"`
}

func main() {
	// Connect to the WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		fmt.Printf("Failed to connect to WebSocket: %v\n", err)
		return
	}
	defer conn.Close()

	// Continuously receive and process messages
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Failed to read message: %v\n", err)
			return
		}

		var message Message
		err = json.Unmarshal(msgBytes, &message)
		if err != nil {
			fmt.Printf("Failed to parse message: %v\n", err)
			continue
		}

		// Create and send the corresponding HTTP request
		reqURL := fmt.Sprintf("http://localhost:3000%s", message.Route)
		req, err := http.NewRequest(strings.ToUpper(message.Method), reqURL, nil)
		if err != nil {
			fmt.Printf("Failed to create HTTP request: %v\n", err)
			continue
		}

		// Set headers
		for key, value := range message.Headers {
			req.Header.Set(key, value)
		}

		// Set query parameters
		query := req.URL.Query()
		for key, value := range message.Params {
			query.Add(key, value)
		}
		req.URL.RawQuery = query.Encode()

		// Set request body
		body, err := json.Marshal(message.Body)
		if err != nil {
			fmt.Printf("Failed to marshal request body: %v\n", err)
			continue
		}
		req.Body = http.NoBody
		req.ContentLength = int64(len(body))
		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(body)), nil
		}

		// Send the HTTP request
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Failed to send HTTP request: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// Handle the HTTP response
		// ... (do something with the response)
	}
}

package main

import (
	_ "bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type ReceivedMessage struct {
	Type    string `json:"type"`
	Message struct {
		Text   string `json:"text"`
		Author struct {
			Username string `json:"username"`
		} `json:"author"`
		Streamer struct {
			Username string `json:"username"`
		} `json:"streamer"`
		ChannelId string `json:"channelId"`
	} `json:"message"`
}

var magicWord = "tacos"
var connected = false

// function to call when message is received via Websocket
func onMessage(c *websocket.Conn, message []byte) {
	var msg ReceivedMessage

	err := json.Unmarshal(message, &msg)
	if err != nil {
		log.Println("error unmarshaling JSON:", err)
		return
	}

	if !connected {
		if strings.Contains(string(msg.Type), "reject_subscription") {
			log.Fatalf("nope... no connection for you")
		} else if strings.Contains(string(msg.Type), "confirm_subscription") {
			connected = true
			fmt.Println("confirmed subscription")
		}
		return
	}

	fmt.Println("Received message: ", msg)

	if msg.Type == "ping" {
		return
	}

	if msg.Message.Text != "" {
		channelId := msg.Message.ChannelId
		if strings.ToLower(msg.Message.Text) == "hello bot" {
			response := map[string]interface{}{
				"command":    "message",
				"identifier": "{\"channel\":\"GatewayChannel\"}",
				"data": fmt.Sprintf(`{
					"action": "send_message",
					"text": "Hello, @%s",
					"channelId":"%s"
				}`, msg.Message.Author.Username, channelId),
			}
			//sendreturn
			jsonStr, err := json.Marshal(response)
			if err != nil {
				log.Fatalf("Critical Error while Marshaling the JSON string: %v", err)
			}
			sendMessage(c, jsonStr)
			return
		}

		if strings.Contains(msg.Message.Text, magicWord) {
			response := map[string]interface{}{
				"command":    "message",
				"identifier": "{\"channel\":\"GatewayChannel\"}",
				"data": fmt.Sprintf(`{
					"action": "send_message",
					"text": "You said @%s's magic word!",
					"channelId":"%s"
				}`, msg.Message.Streamer.Username, channelId),
			}
			jsonStr, err := json.Marshal(response)
			if err != nil {
				log.Fatalf("Critical Error while Marshaling the JSON string: %v", err)
			}

			sendMessage(c, jsonStr)
			return
		}
	}
}

// function to call when sending a message to the Websocket
func sendMessage(c *websocket.Conn, msg []byte) {
	if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Fatalf("Critical Error sending message: %v", err)
	}
}

func main() {
	//load environment variables from .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	HOST := os.Getenv("JOYSTICKTV_HOST")
	_ = HOST //null ref pointer to keep IDE from deleting future use variable
	CLIENT_ID := os.Getenv("JOYSTICKTV_CLIENT_ID")
	CLIENT_SECRET := os.Getenv("JOYSTICKTV_CLIENT_SECRET")
	WSS_HOST := os.Getenv("JOYSTICKTV_API_HOST")

	//concatenate and base64 encode clientId and clientSecret
	auth := base64.StdEncoding.EncodeToString([]byte(CLIENT_ID + ":" + CLIENT_SECRET))
	//construct the WSS Endpoint Uri
	wsEndpoint := fmt.Sprintf("%s?token=%s", WSS_HOST, auth)

	//Construct websocket
	ws, _, err := websocket.DefaultDialer.Dial(wsEndpoint, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	log.Println("connection has opened")

	defer ws.Close()

	//start listening
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Read Error: %v :: %v", err, message)
			break
		}
		if !connected {
			//subscribe on open
			subscribe := map[string]interface{}{
				"command":    "subscribe",
				"identifier": "{\"channel\":\"GatewayChannel\"}",
			}

			jsonStr, err := json.Marshal(subscribe)
			if err != nil {
				log.Fatalf("Critical Error Marshaling JSON string: %v", err)
			}
			sendMessage(ws, jsonStr)
		}

		go onMessage(ws, message)
	}
}

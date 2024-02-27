package main

import (
	_ "bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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

// global vars
var magicWord = "tacos"
var connected = false
var authorized = false

// global connection variables
var HOST = ""
var CLIENT_ID = ""
var CLIENT_SECRET = ""
var WSS_HOST = ""
var WSS_ENDPOINT = ""
var BASIC_AUTH = ""

// function to call when message is received via Websocket
func onMessage(c *websocket.Conn, message []byte) {
	var msg ReceivedMessage

	err := json.Unmarshal(message, &msg)
	if err != nil { //basically returns on pings because structure doesn't match, and I didn't want to put the entire json structure up. see https://support.joystick.tv/ and go to dev section to see full json structure
		//log.Println("error unmarshaling JSON:", err)
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

// authorization
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`Visit <a href="/install">INSTALL</a> to install Bot`))
}

func handleInstall(w http.ResponseWriter, r *http.Request) {
	const state = "adachiehehe123"
	clientId := CLIENT_ID
	// Make sure host is correctly set
	host := HOST

	params := url.Values{}
	params.Add("client_id", clientId)
	params.Add("scope", "bot")
	params.Add("state", state)

	authorizeUri := host + "/api/oauth/authorize?" + params.Encode()
	http.Redirect(w, r, authorizeUri, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	log.Println("STATE:", state)
	log.Println("CODE:", code)

	accessToken := BASIC_AUTH
	host := HOST

	clientRequestParams := url.Values{}
	clientRequestParams.Add("redirect_uri", "/unused")
	clientRequestParams.Add("code", code)
	clientRequestParams.Add("grant_type", "authorization_code")

	tokenUri := host + "/api/oauth/token"

	req, _ := http.NewRequest("POST", tokenUri, nil)
	req.Header.Add("Authorization", "Basic "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	log.Println(result["access_token"])
	w.Write([]byte("Bot has been activated"))
	authorized = true
}

//end auth

func main() {
	//load environment variables from .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	HOST = os.Getenv("JOYSTICKTV_HOST")
	_ = HOST //null ref pointer to keep IDE from deleting future use variable
	CLIENT_ID = os.Getenv("JOYSTICKTV_CLIENT_ID")
	CLIENT_SECRET = os.Getenv("JOYSTICKTV_CLIENT_SECRET")
	WSS_HOST = os.Getenv("JOYSTICKTV_API_HOST")

	//concatenate and base64 encode clientId and clientSecret
	BASIC_AUTH = base64.StdEncoding.EncodeToString([]byte(CLIENT_ID + ":" + CLIENT_SECRET))
	//construct the WSS Endpoint Uri
	WSS_ENDPOINT = fmt.Sprintf("%s?token=%s", WSS_HOST, BASIC_AUTH)

	//setup routes
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/install", handleInstall)
	http.HandleFunc("/callback", handleCallback)

	go func() {
		fmt.Println("Go to: http://localhost:8080/install to install the bot.")
		log.Println("Listening on :8080...")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	//wait for authorization
	for {
		if !authorized {
			time.Sleep(500 * time.Millisecond)
			continue
		} else {
			break
		}
	}

	//Construct websocket
	ws, _, err := websocket.DefaultDialer.Dial(WSS_ENDPOINT, nil)
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

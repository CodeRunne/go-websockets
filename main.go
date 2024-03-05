package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
)

type Country struct {
	CountryID   string  `json:"country_id"`
	Probability float64 `json:"probability"`
}

type Feedback struct {
	Count   uint      `json:"count"`
	Name    string    `json:"name"`
	Country []Country `json:"country"`
}

type Body struct {
	Name string
}

type WebSocketFeedback struct {
	UserID uint `json:"user_id"`
	Point  int64 `json:"point"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var feedback Feedback
var mypoints int64

func main() {
	// define routes
	http.HandleFunc("/echo", RespondToWebSocketHandler)
	http.HandleFunc("/country-name-delegator", NameCountryDelegatorHandler)
	fmt.Println("Started server at port :3000")
	http.ListenAndServe(":3000", nil)
}

func RespondToWebSocketHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	for {
		// Reading message from the client
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		var websocketfeedback WebSocketFeedback
		json.NewDecoder(bytes.NewReader(msg)).Decode(&websocketfeedback)

		// validate user ( valid user_id => 123 )
		if websocketfeedback.UserID != 123 {
			err = conn.WriteMessage(msgType, []byte("User invalid!"))
			if err != nil {
				fmt.Println(err)
				return
			}

			conn.SetCloseHandler(func(code int, text string) error {
				fmt.Printf("Client disconnected with error code %d and text %s", code, text)
				return nil
			})
		}

		// increment point
		mypoints += websocketfeedback.Point

		fmt.Printf("Received from %s: %v", conn.RemoteAddr(), websocketfeedback)

		// convert point to byte
		byte_point, _ := json.Marshal(mypoints)

		// sending a message to the client
		err = conn.WriteMessage(msgType, byte_point)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func NameCountryDelegatorHandler(res http.ResponseWriter, req *http.Request) {

	// add websocket
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	for {
		// Reading message from the client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		var requestName Body
		json.NewDecoder(bytes.NewReader(msg)).Decode(&requestName)

		// fetch data from api
		feedback, err := fetchFromApi(requestName.Name)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(err.Error()))
			return
		}

		// sending a message to the client
		err = conn.WriteJSON(feedback)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func fetchFromApi(name string) (*Feedback, error) {
	// get http client to handle requests
	client := &http.Client{}

	// feedback struct
	var feedback *Feedback

	// url format
	format := fmt.Sprintf("https://api.nationalize.io/?name=%s", name)
	url, err := url.ParseRequestURI(format)
	if err != nil {
		return feedback, err
	}

	// make new http request
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return feedback, err
	}

	// handle request
	res, err := client.Do(req)
	if err != nil {
		return feedback, err
	}

	// close response body
	defer res.Body.Close()

	// read request body
	_ = json.NewDecoder(res.Body).Decode(&feedback)
	return feedback, nil
}

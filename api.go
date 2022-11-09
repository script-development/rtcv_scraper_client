package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type serverConn struct {
	authHeaderValue string
	serverLocation  string
}

// API holds information used to communicate with the RT-CV api
type API struct {
	primaryConnection int
	connections       []serverConn

	MockMode  bool
	mockCache map[string]time.Time

	CancelPreviouseCommunicationChan chan struct{}
	WebsocketReq                     chan []byte
	WebsocketRespLock                sync.Mutex
	WebsocketResp                    []chan []byte

	Cache map[string]time.Time
}

// NewAPI creates a new instance of the API
func NewAPI() *API {
	return &API{
		CancelPreviouseCommunicationChan: make(chan struct{}),
		WebsocketReq:                     make(chan []byte),
		WebsocketResp:                    []chan []byte{},

		Cache: map[string]time.Time{},
	}
}

// SetMockMode enables mock mode, which can be used for testing
func (a *API) SetMockMode() {
	a.connections = nil

	a.MockMode = true
	a.mockCache = map[string]time.Time{}
}

// SetCredentialsArg contains the credentials used to authenticate with RT-CV
type SetCredentialsArg struct {
	ServerLocation string `json:"server_location"`
	APIKeyID       string `json:"api_key_id"`
	APIKey         string `json:"api_key"`
	Primary        bool   `json:"primary"`
}

// SetCredentials sets the api credentials so we can make fetch requests to RT-CV
func (a *API) SetCredentials(credentialsList []SetCredentialsArg) error {
	a.MockMode = false
	a.primaryConnection = -1

	a.connections = []serverConn{}
	for idx, credentials := range credentialsList {
		conn := serverConn{}

		if credentials.ServerLocation == "" {
			return errors.New("server_location cannot be empty")
		}
		if !strings.HasPrefix(credentials.ServerLocation, "http://") && !strings.HasPrefix(credentials.ServerLocation, "https://") {
			return errors.New("server_location must start with a supported protocol like: http:// or https://")
		}
		conn.serverLocation = credentials.ServerLocation

		if credentials.APIKeyID == "" {
			return errors.New("api_key_id cannot be empty")
		}
		if credentials.APIKey == "" {
			return errors.New("api_key cannot be empty")
		}
		hashedAPIKey := sha512.Sum512([]byte(credentials.APIKey))
		hashedAPIKeyStr := hex.EncodeToString(hashedAPIKey[:])
		conn.authHeaderValue = "Basic " + credentials.APIKeyID + ":" + hashedAPIKeyStr

		a.connections = append(a.connections, conn)

		if credentials.Primary {
			if a.primaryConnection == -1 {
				a.primaryConnection = idx
			} else {
				return errors.New("can only have one primary connection")
			}
		}
	}

	switch len(credentialsList) {
	case 0:
		return errors.New("you cannot define no connections")
	case 1:
		a.primaryConnection = 0
	case 2:
		if a.primaryConnection == -1 {
			return errors.New("when defineing multiple connections, one must be set as primary")
		}
	}

	return nil
}

// Get makes a get request to RT-CV
func (c *serverConn) Get(path string, unmarshalResInto interface{}) error {
	return c.DoRequest("GET", path, nil, unmarshalResInto)
}

// Post makes a post request to RT-CV
func (c *serverConn) Post(path string, body interface{}, unmarshalResInto interface{}) error {
	return c.DoRequest("POST", path, body, unmarshalResInto)
}

// DoRequest makes a http request to RT-CV
func (c *serverConn) DoRequest(method, path string, body, unmarshalResInto interface{}) error {
	var reqBody io.ReadCloser
	if body != nil {
		reqBodyBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = ioutil.NopCloser(bytes.NewBuffer(reqBodyBytes))
	}
	req, err := http.NewRequest(method, c.serverLocation+path, reqBody)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.authHeaderValue != "" {
		req.Header.Add("Authorization", c.authHeaderValue)
	}

	attempt := 0
	for {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			attempt++
			if attempt > 3 {
				return fmt.Errorf("%s retried 4 times", err.Error())
			}
			time.Sleep(time.Second * time.Duration(attempt) * 2)
			continue
		}

		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		if res.StatusCode >= 400 && res.StatusCode < 600 {
			errorRes := struct {
				Error string `json:"error"`
			}{}
			err = json.Unmarshal(resBody, &errorRes)
			if err != nil {
				return fmt.Errorf("server returned %d code, with body %s", res.StatusCode, string(resBody))
			}
			return errors.New(errorRes.Error)
		}

		if unmarshalResInto != nil {
			return json.Unmarshal(resBody, unmarshalResInto)
		}
		return nil
	}
}

func (c *serverConn) tryConnectToWS() *websocket.Conn {
	url := c.serverLocation
	url = strings.Replace(url, "http://", "ws://", 1)
	url = strings.Replace(url, "https://", "wss://", 1)
	url += "/api/v1/scraper/ws"

	attempt := 0
	for {
		conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Authorization": []string{c.authHeaderValue}})
		if err == nil {
			if attempt > 0 {
				fmt.Println("connected to web socket")
			}
			return conn
		}

		attempt++
		retryInSeconds := time.Second
		if attempt == 1 {
			// retry in 1 second
		} else if attempt <= 2 {
			retryInSeconds *= 2
		} else if attempt <= 4 {
			retryInSeconds *= 4
		} else if attempt <= 6 {
			retryInSeconds *= 10
		} else {
			retryInSeconds *= 15
		}

		fmt.Printf("unable to connect to web socket, error: %s, retrying in %s\n", err, retryInSeconds)
		time.Sleep(retryInSeconds)
	}
}

// NoCredentials returns true if the SetCredentials method was not yet called and we aren't in mock mode
func (a *API) NoCredentials() bool {
	return len(a.connections) == 0 && !a.MockMode
}

// ErrMissingCredentials is returned when the SetCredentials method was not yet called
// while we're trying to execute an action that requires them
var ErrMissingCredentials = errors.New("missing credentials, call set_credentials before this method")

// SetCacheEntry sets a cache entry for the reference number that expires after the duration
func (a *API) SetCacheEntry(referenceNr string, duration time.Duration) {
	a.Cache[referenceNr] = time.Now().Add(duration)
}

// CacheEntryExists returns true if the cache entry exists and is not expired
func (a *API) CacheEntryExists(referenceNr string) bool {
	cacheEntryInsertionTime, cacheEntryExists := a.Cache[referenceNr]
	if !cacheEntryExists {
		return false
	}

	expired := time.Now().After(cacheEntryInsertionTime)
	if expired {
		delete(a.Cache, referenceNr)
	}
	return !expired
}

// WSMsg is a message recived and send to the websocket
type WSMsg[T any] struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Data T      `json:"data"`
}

// HandleWebsocketResponse handles a websocket response
// This decodes the payload and checks to which connected websocket it should be sent
func (a *API) HandleWebsocketResponse(payload []byte) {
	if a.MockMode {
		return
	}

	data := WSMsg[json.RawMessage]{}
	err := json.Unmarshal(payload, &data)
	if err != nil {
		fmt.Println("error un-marshaling websocket response, error:", err)
		return
	}

	idParts := strings.SplitN(data.ID, "-", 2)
	if len(idParts) != 2 {
		fmt.Println("error invalid id in websocket response, expected 2 parts but got 1")
		return
	}

	idx, err := strconv.Atoi(idParts[0])
	if err != nil {
		fmt.Println("error invalid id connection index in websocket response, error:", err)
		return
	}

	data.ID = idParts[1]

	// Re-encode data with the new ID
	payload, err = json.Marshal(data)
	if err != nil {
		fmt.Println("error marshaling websocket response, error:", err)
		return
	}

	a.WebsocketRespLock.Lock()
	// Data sending of the channel is thread safe but fething the array index is not hence why we lock WebsocketRespLock
	a.WebsocketResp[idx] <- payload
	a.WebsocketRespLock.Unlock()
}

// ConnectToAllWebsockets connects to all the conenctions their websocket
func (a *API) ConnectToAllWebsockets() {
	if a.MockMode {
		return
	}

	a.WebsocketResp = make([]chan []byte, len(a.connections))
	for idx := 0; idx < len(a.connections); idx++ {
		go a.connectToWS(idx)
	}
}

// connectToWS connects to the rtcv websocket
func (a *API) connectToWS(idx int) {
	server := a.connections[idx]

	url := server.serverLocation
	url = strings.Replace(url, "http://", "ws://", 1)
	url = strings.Replace(url, "https://", "wss://", 1)
	url += "/api/v1/scraper/ws"

	var c *websocket.Conn
	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	a.WebsocketRespLock.Lock()
	a.WebsocketResp[idx] = make(chan []byte)
	listenChan := &a.WebsocketResp[idx]
	a.WebsocketRespLock.Unlock()

	go func(ws *chan []byte) {
		for {
			// TODO: if the response fails to send data might get lost.
			//   It would be nice if the response is retried when WriteMessage fails
			resp := <-a.WebsocketResp[idx]
			err := c.WriteMessage(1, resp)
			if err != nil {
				fmt.Println("unable to write ws response:", err)
			}
		}
	}(listenChan)

	firstMessage := true
	var aMessageWasHandled atomic.Bool
	for {
		c = a.connections[0].tryConnectToWS()

		for {
			msgType, msgBytes, err := c.ReadMessage()
			if err != nil {
				fmt.Println("error reading from web socket:", err)
				break
			}

			switch msgType {
			case 1, 2:
				// 1 - text message
				// 2 - binary message
				// Ok continue
			default:
				// Ignore other message types
				continue
			}

			msg := WSMsg[json.RawMessage]{}
			err = json.Unmarshal(msgBytes, &msg)
			if err != nil {
				fmt.Println("error un-marshaling web socket message:", err)
				continue
			}

			// We inject the index of the server connection into the message id so we know where to send the response to later
			// See the /server_response for how we handle the response
			msg.ID = fmt.Sprintf("%d-%s", idx, msg.ID)

			msgBytes, err = json.Marshal(msg)
			if err != nil {
				fmt.Println("error marshaling web socket message:", err)
				continue
			}

			timeout := time.Second
			if aMessageWasHandled.Load() || firstMessage {
				// It might be this scraper does not listen to the /server_request url thus we will try to send something over a channel that will never read
				// That's a lot of waisted time
				timeout = time.Second * 30
			}
			firstMessage = false

			go func(msgBytes []byte, timeout time.Duration) {
				select {
				case a.WebsocketReq <- msgBytes:
					// Ok message was send
					aMessageWasHandled.Store(true)
				case <-time.After(timeout):
					errMsg := "Unable to handle request by RT-CV server"
					if !aMessageWasHandled.Load() {
						errMsg += ", probably becuase there is no one waiting for a response"
					}
					fmt.Println(errMsg)
				}
			}(msgBytes, timeout)
		}

		c.Close()
	}

}

// CancelPreviouseCommunication cancels the previous communication if it's still running
func (a *API) CancelPreviouseCommunication() {
	select {
	case a.CancelPreviouseCommunicationChan <- struct{}{}:
		// A previous communication was canceled
	default:
		// There are no more previous communications to cancel
	}
}

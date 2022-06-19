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
	"time"
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

	Cache map[string]time.Time
}

// NewAPI creates a new instance of the API
func NewAPI() *API {
	return &API{
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

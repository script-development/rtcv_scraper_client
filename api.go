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

// API holds information used to communicate with the RT-CV api
type API struct {
	authHeaderValue string
	serverLocation  string
	Cache           map[string]time.Time
	MockMode        bool
	MockOptions     MockOptions
}

// MockOptions represends options for the RT-CV mocking mode
type MockOptions struct {
	Secrets map[string]json.RawMessage `json:"secrets"`
}

// NewAPI creates a new instance of the API
func NewAPI() *API {
	return &API{
		Cache: map[string]time.Time{},
	}
}

// SetCredentials sets the api credentials so we can make fetch requests to RT-CV
func (a *API) SetCredentials(serverLocation, apiKeyID, apiKey string, runAsMockWithOpts *MockOptions) error {
	a.MockMode = runAsMockWithOpts != nil
	if a.MockMode {
		a.MockOptions = *runAsMockWithOpts
		a.MockMode = true
		return nil
	}

	if serverLocation == "" {
		return errors.New("server_location cannot be empty")
	}
	a.serverLocation = serverLocation

	if apiKeyID == "" {
		return errors.New("api_key_id cannot be empty")
	}
	if apiKey == "" {
		return errors.New("api_key cannot be empty")
	}
	hashedAPIKey := sha512.Sum512([]byte(apiKey))
	hashedAPIKeyStr := hex.EncodeToString(hashedAPIKey[:])
	a.authHeaderValue = "Basic " + apiKeyID + ":" + hashedAPIKeyStr

	return nil
}

// Get makes a get request to RT-CV
func (a *API) Get(path string, unmarshalResInto interface{}) error {
	return a.DoRequest("GET", path, nil, unmarshalResInto)
}

// Post makes a post request to RT-CV
func (a *API) Post(path string, body interface{}, unmarshalResInto interface{}) error {
	return a.DoRequest("POST", path, body, unmarshalResInto)
}

// DoRequest makes a http request to RT-CV
func (a *API) DoRequest(method, path string, body, unmarshalResInto interface{}) error {
	var reqBody io.ReadCloser
	if body != nil {
		reqBodyBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = ioutil.NopCloser(bytes.NewBuffer(reqBodyBytes))
	}
	req, err := http.NewRequest(method, a.serverLocation+path, reqBody)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	if a.authHeaderValue != "" {
		req.Header.Add("Authorization", a.authHeaderValue)
	}

	attempt := 0
	for {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			attempt++
			if attempt > 3 {
				return fmt.Errorf("%s, retried 4 times", err.Error())
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
	return a.authHeaderValue == "" && !a.MockMode
}

// ErrMissingCredentials is returned when the SetCredentials method was not yet called
// while we're trying to execute an action that requires them
var ErrMissingCredentials = errors.New("missing credentials, call set_credentials before this method")

// GetSecret returns a secret from RT-CV
func (a *API) GetSecret(key, encryptionKey string, result interface{}) error {
	if a.MockMode {
		if a.MockOptions.Secrets == nil {
			return json.Unmarshal([]byte("null"), result)
		}
		secret, ok := a.MockOptions.Secrets[key]
		if !ok || secret == nil {
			return json.Unmarshal([]byte("null"), result)
		}
		return json.Unmarshal(secret, result)
	}
	if a.NoCredentials() {
		return ErrMissingCredentials
	}
	return a.Get(fmt.Sprintf("/api/v1/secrets/myKey/%s/%s", key, encryptionKey), result)
}

// GetUsersSecret returns strictly defined users secret
func (a *API) GetUsersSecret(key, encryptionKey string) ([]UserSecret, error) {
	if a.NoCredentials() {
		return []UserSecret{}, ErrMissingCredentials
	}
	if key == "" {
		key = "users"
	}

	result := []UserSecret{}
	err := a.GetSecret(key, encryptionKey, &result)
	return result, err
}

// GetUserSecret returns strictly defined user secret
func (a *API) GetUserSecret(key, encryptionKey string) (UserSecret, error) {
	if a.NoCredentials() {
		return UserSecret{}, ErrMissingCredentials
	}
	if key == "" {
		key = "user"
	}

	result := UserSecret{}
	err := a.GetSecret(key, encryptionKey, &result)
	return result, err
}

// CacheEntryExists returns true if the cache entry exists and is not expired
func (a *API) CacheEntryExists(referenceNr string) bool {
	cacheEntryInsertionTime, cacheEntryExists := a.Cache[referenceNr]
	if cacheEntryExists {
		expired := time.Now().After(
			cacheEntryInsertionTime.Add(time.Hour * 72), // 3 days
		)
		if expired {
			delete(a.Cache, referenceNr)
			cacheEntryExists = false
		}
	}
	return cacheEntryExists
}

// UserSecret represents the json layout of an user secret
type UserSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

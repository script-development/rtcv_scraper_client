package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

func main() {
	PrintMessage(MessageTypeReady, "waiting for credentials")

	api := NewAPI()

	for {
		text := ""
		_, err := fmt.Scanln(&text)
		if err != nil {
			PrintMessage(MessageTypeError, err.Error())
			break
		}

		PrintMessage(LoopAction(api, text))
	}

	// If the loop stops there is a critical error
	// Thus the program should exit with a error code
	os.Exit(1)
}

func LoopAction(api *API, inputJSON string) (msgType MessageType, msgContent interface{}) {
	returnErr := func(err error) (msgType MessageType, msgContent interface{}) {
		return MessageTypeError, err.Error()
	}

	input := InMessage{}
	err := json.Unmarshal([]byte(inputJSON), &input)
	if err != nil {
		return returnErr(err)
	}

	switch input.Type {
	case "set_credentials":
		credentialsArgs := struct {
			ServerLocation string `json:"server_location"`
			ApiKeyID       string `json:"api_key_id"`
			ApiKey         string `json:"api_key"`
		}{}
		err = json.Unmarshal(input.Content, &credentialsArgs)
		if err != nil {
			return returnErr(err)
		}

		err = api.SetCredentials(
			credentialsArgs.ServerLocation,
			credentialsArgs.ApiKeyID,
			credentialsArgs.ApiKey,
		)
		if err != nil {
			return returnErr(err)
		}

		err = api.Get("/api/v1/health", nil)
		if err != nil {
			return returnErr(err)
		}

		apiKeyInfo := struct {
			Roles []struct {
				Role uint64 `json:"role"`
			} `json:"roles"`
		}{}
		err := api.Get("/api/v1/auth/keyinfo", &apiKeyInfo)
		if err != nil {
			return returnErr(err)
		}

		hasScraperRole := false
		for _, role := range apiKeyInfo.Roles {
			if role.Role == 1 {
				hasScraperRole = true
				break
			}
		}
		if !hasScraperRole {
			return returnErr(errors.New("provided key does not have scraper role (nr 1)"))
		}

		referenceNrs := []string{}
		err = api.Get("/api/v1/scraper/scannedReferenceNrs/since/days/3", &referenceNrs)
		if err != nil {
			return returnErr(err)
		}

		now := time.Now()
		for _, nr := range referenceNrs {
			api.Cache[nr] = now
		}

		return MessageTypeOk, nil
	case "send_cv":
		cvContent := map[string]interface{}{}
		err := json.Unmarshal(input.Content, &cvContent)
		if err != nil {
			return returnErr(errors.New("cv expected to be an object"))
		}

		referenceNrInterf, ok := cvContent["referenceNumber"]
		if !ok {
			return returnErr(errors.New("referenceNumber field does not exists"))
		}

		referenceNr, ok := referenceNrInterf.(string)
		if !ok {
			return returnErr(errors.New("referenceNumber is expected to be a string"))
		}

		cacheEntryExists := api.CacheEntryExists(referenceNr)
		if cacheEntryExists {
			// Cannot send the same cv twice
			return MessageTypeOk, nil
		}

		api.Cache[referenceNr] = time.Now()

		scanCVBody := json.RawMessage(`{"cv":` + string(input.Content) + `}`)

		err = api.Post("/api/v1/scraper/scanCV", scanCVBody, nil)
		if err != nil {
			return returnErr(err)
		}

		return MessageTypeOk, nil
	case "get_secret", "get_users_secret", "get_user_secret":
		getSecretArgs := struct {
			Key           string `json:"key"`
			EncryptionKey string `json:"encryption_key"`
		}{}
		err = json.Unmarshal(input.Content, &getSecretArgs)
		if err != nil {
			return returnErr(err)
		}

		key := getSecretArgs.Key
		encryptionKey := getSecretArgs.EncryptionKey

		switch input.Type {
		case "get_secret":
			res := json.RawMessage{}
			err = api.GetSecret(key, encryptionKey, &res)
			if err != nil {
				return returnErr(err)
			}
			return MessageTypeOk, res
		case "get_users_secret":
			users, err := api.GetUsersSecret(key, encryptionKey)
			if err != nil {
				return returnErr(err)
			}
			return MessageTypeOk, users
		case "get_user_secret":
			user, err := api.GetUserSecret(key, encryptionKey)
			if err != nil {
				return returnErr(err)
			}
			return MessageTypeOk, user
		default:
			return returnErr(errors.New("unknown secret"))
		}
	case "set_cached_reference", "has_cached_reference":
		referenceNr := ""
		err = json.Unmarshal(input.Content, &referenceNr)
		if err != nil {
			return returnErr(err)
		}

		if referenceNr == "" {
			return returnErr(errors.New("reference number cannot be an empty string"))
		}

		if input.Type == "set_cached_reference" {
			api.Cache[referenceNr] = time.Now()
			return MessageTypeOk, nil
		}

		return MessageTypeOk, api.CacheEntryExists(referenceNr)
	case "ping":
		return MessageTypePong, nil
	default:
		return returnErr(fmt.Errorf("unknown message type %s", input.Type))
	}
}

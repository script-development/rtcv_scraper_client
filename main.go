package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	if strings.ToLower(os.Getenv("SCRAPER_CLIENT_V2")) == "true" {
		newMain()
		return
	}

	replayFile := ""
	replaySkipCommands := ""
	repeatReplay := uint(1)
	flag.StringVar(&replayFile, "replay", "", "replay file, file can be generated using LOG_SCRAPER_CLIENT_INPUT=true")
	flag.StringVar(&replaySkipCommands, "replaySkipCommands", "", "in a replay skip sending specific commands, for multiple commands add comma's in between")
	flag.UintVar(&repeatReplay, "repeatReplay", 1, "repeat replay this many times, handy for performance profiling the RT-CV matcher")
	flag.Parse()

	if replayFile != "" {
		if repeatReplay == 0 {
			log.Fatal("repeatReplay must be greater than 0")
		}

		out, err := ioutil.ReadFile(replayFile)
		if err != nil {
			log.Fatal(err)
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		fmt.Println("replaying", len(lines), "commands")

		commandsToSkip := map[string]bool{}
		for _, skipCommand := range strings.Split(replaySkipCommands, ",") {
			skipCommand = strings.TrimSpace(skipCommand)
			if skipCommand != "" {
				commandsToSkip[skipCommand] = true
			}
		}

		for i := uint(0); i < repeatReplay; i++ {
			api := NewAPI()

			for _, line := range lines {
				input := InMessage{}
				err := json.Unmarshal([]byte(line), &input)
				if err != nil {
					fmt.Println("error:", err.Error())
					continue
				}

				if commandsToSkip[input.Type] {
					continue
				}

				startTime := time.Now()
				msgType, msgContents := LoopAction(api, line)
				endTime := time.Now()
				if jsonContent, ok := msgContents.(json.RawMessage); ok {
					msgContents = string(jsonContent)
				}

				durationMs := fmt.Sprintf("%dms", endTime.Sub(startTime).Milliseconds())
				durationPaddingLen := 5 - len(durationMs)
				if durationPaddingLen < 0 {
					durationPaddingLen = 0
				}
				durationPadding := strings.Repeat(" ", durationPaddingLen)

				inputPaddingLen := 20 - len(input.Type)
				if inputPaddingLen < 0 {
					inputPaddingLen = 0
				}
				inputTypePadding := strings.Repeat(" ", inputPaddingLen)

				fmt.Printf("%s%s IN: %s%s OUT: %s: %+v\n", durationMs, durationPadding, input.Type, inputTypePadding, msgType.String(), msgContents)
			}
		}

		os.Exit(0)
	}

	MessageTypeReady.Print("waiting for credentials")

	api := NewAPI()

	logInput := truthyStringValues[strings.ToLower(os.Getenv("LOG_SCRAPER_CLIENT_INPUT"))]

	var logInputFile *os.File
	var err error
	if logInput {
		logInputFile, err = os.OpenFile("scraper_client_input.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			MessageTypeError.Print(err.Error())
			os.Exit(1)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	err = scanner.Err()
	if err != nil {
		MessageTypeError.Print(err.Error())
		if logInput {
			logInputFile.Close()
		}
		os.Exit(1)
	}

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		err = scanner.Err()
		if err != nil {
			MessageTypeError.Print(err.Error())
			break
		}

		if logInput {
			fmt.Fprintln(logInputFile, text)
			logInputFile.Sync()
		}

		mt, msgContent := LoopAction(api, text)
		mt.Print(msgContent)
	}

	// If the loop stops there is a critical error
	// Thus the program should exit with a error code
	if logInput {
		logInputFile.Close()
	}
	os.Exit(1)
}

// LoopAction handles one line of input and returns the response
// Note that no-where inside this function we should print to the screen
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
	case "set_credentials", "set_multiple_credentials", "set_mock":
		switch input.Type {
		case "set_credentials":
			credentialsArgs := SetCredentialsArg{}
			err = json.Unmarshal(input.Content, &credentialsArgs)
			if err != nil {
				return returnErr(err)
			}

			err = api.SetCredentials([]SetCredentialsArg{credentialsArgs})
			if err != nil {
				return returnErr(err)
			}
		case "set_multiple_credentials":
			credentialsArgs := []SetCredentialsArg{}
			err = json.Unmarshal(input.Content, &credentialsArgs)
			if err != nil {
				return returnErr(err)
			}

			err = api.SetCredentials(credentialsArgs)
			if err != nil {
				return returnErr(err)
			}
		case "set_mock":
			options := MockOptions{}
			err = json.Unmarshal(input.Content, &options)
			if err != nil {
				return returnErr(err)
			}

			api.SetMockMode(options)
		}

		if api.MockMode {
			return MessageTypeOk, nil
		}

		for _, conn := range api.connections {
			err = conn.Get("/api/v1/health", nil)
			if err != nil {
				return returnErr(err)
			}

			apiKeyInfo := struct {
				Roles []struct {
					Role uint64 `json:"role"`
				} `json:"roles"`
			}{}
			err = conn.Get("/api/v1/auth/keyinfo", &apiKeyInfo)
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
		}

		referenceNrs := []string{}
		err = api.connections[api.primaryConnection].Get("/api/v1/scraper/scannedReferenceNrs/since/days/30", &referenceNrs)
		if err != nil {
			return returnErr(err)
		}

		DebugToLogfile(referenceNrs)

		for _, nr := range referenceNrs {
			api.SetCacheEntry(nr, time.Hour*72) // 3 days
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

		hasMatch := false
		if api.MockMode {
			api.SetCacheEntry(referenceNr, time.Hour*72)
			hasMatch = true
		} else {
			scanCVBody := json.RawMessage(`{"cv":` + string(input.Content) + `}`)

			for idx, conn := range api.connections {
				var response struct {
					HasMatches bool `json:"hasMatches"`
				}

				err = conn.Post("/api/v1/scraper/scanCV", scanCVBody, &response)
				if err != nil {
					return returnErr(err)
				}

				if idx == api.primaryConnection {
					hasMatch = response.HasMatches
					if hasMatch {
						// Only cache the CVs that where matched to something
						api.SetCacheEntry(referenceNr, time.Hour*72) // 3 days
					}
				}
			}
		}

		return MessageTypeOk, hasMatch
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
	case "set_cached_reference", "set_short_cached_reference", "has_cached_reference":
		referenceNr := ""
		err = json.Unmarshal(input.Content, &referenceNr)
		if err != nil {
			return returnErr(err)
		}

		if referenceNr == "" {
			return returnErr(errors.New("reference number cannot be an empty string"))
		}

		switch input.Type {
		case "set_cached_reference":
			api.SetCacheEntry(referenceNr, time.Hour*72) // 3 days
			return MessageTypeOk, nil
		case "set_short_cached_reference":
			api.SetCacheEntry(referenceNr, time.Hour*12) // 0.5 days
			return MessageTypeOk, nil
		}

		// has_cached_reference
		hasCachedReference := api.CacheEntryExists(referenceNr)
		DebugToLogfile("has_cached_reference", referenceNr, ">", hasCachedReference)

		return MessageTypeOk, hasCachedReference
	case "ping":
		return MessageTypePong, nil
	default:
		return returnErr(fmt.Errorf("unknown message type %s", input.Type))
	}
}

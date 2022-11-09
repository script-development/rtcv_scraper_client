package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/script-development/rtcv_scraper_client/v2/crypto"
)

func main() {
	envFilename := "env.json"
	alternativeFileName := os.Getenv("RTCV_SCRAPER_CLIENT_ENV_FILE")
	if alternativeFileName != "" {
		envFilename = alternativeFileName
	}

	envEnvName := "RTCV_SCRAPER_CLIENT_ENV"
	envFile, err := ioutil.ReadFile(envFilename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal("unable to read env file, error: " + err.Error())
		}
		envFile = []byte(os.Getenv(envEnvName))
		if len(envFile) == 0 {
			log.Fatalf("no %s file or %s environment variable found, cannot continue", envFilename, envEnvName)
		}
	}

	env := Env{}
	err = json.Unmarshal(envFile, &env)
	if err != nil {
		log.Fatal("unable to parse env file, error: " + err.Error())
	}

	err = env.validate()
	if err != nil {
		log.Fatal("validating env failed, error: " + err.Error())
	}

	api := NewAPI()

	credentials := []SetCredentialsArg{env.PrimaryServer.toCredArg(true)}
	for _, server := range env.AlternativeServers {
		credentials = append(credentials, server.toCredArg(false))
	}

	var loginUsers []EnvUser
	if !env.MockMode {
		err = api.SetCredentials(credentials)
		if err != nil {
			log.Fatal(err)
		}
		decryptionKey := crypto.LoadAndVerivyKeys(env.PublicKey, env.PrivateKey)

		fmt.Println("credentials set")
		fmt.Println("testing connections..")
		loginUsers = testServerConnections(api, credentials[0].APIKeyID, decryptionKey)
		fmt.Println("connected to RTCV")
	} else {
		api.SetMockMode()
		loginUsers = env.MockUsers
		fmt.Println("In mock mode")
		fmt.Println("You can turn this off in `env.json` by setting `mock_mode` to false")
	}

	api.ConnectToAllWebsockets()
	useAddress := startWebserver(env, api, loginUsers)

	healthCheckPort := os.Getenv("RTCV_SCRAPER_CLIENT_HEALTH_CHECK_PORT")
	if healthCheckPort != "" {
		go startHealthCheckServer(healthCheckPort)
	}

	fmt.Println("running scraper..")

	if len(os.Args) <= 1 {
		log.Fatal("must provide a command to run, for example: rtcv_scraper_client npm run scraper")
	}

	scraper := exec.Command(os.Args[1], os.Args[2:]...)
	scraper.Env = append(os.Environ(), "SCRAPER_ADDRESS="+useAddress)

	// Piple output of scraper to stdout
	scraper.Stdin = os.Stdin
	scraper.Stdout = os.Stdout
	scraper.Stderr = os.Stderr

	err = scraper.Run()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		log.Fatal(err)
	}
}

func testServerConnections(api *API, apiKeyID string, decryptionKey *crypto.Key) []EnvUser {
	var wg sync.WaitGroup

	for _, conn := range api.connections {
		wg.Add(1)
		go func(conn serverConn) {
			err := conn.Get("/api/v1/health", nil)
			if err != nil {
				log.Fatal(err)
			}

			apiKeyInfo := struct {
				Roles []struct {
					Role uint64 `json:"role"`
				} `json:"roles"`
			}{}
			err = conn.Get("/api/v1/auth/keyinfo", &apiKeyInfo)
			if err != nil {
				log.Fatal(err)
			}

			hasScraperRole := false
			for _, role := range apiKeyInfo.Roles {
				if role.Role == 1 {
					hasScraperRole = true
					break
				}
			}
			if !hasScraperRole {
				log.Fatal("provided key does not have scraper role (nr 1)")
			}
			wg.Done()
		}(conn)
	}

	scraperUsers := struct {
		ScraperPublicKey string    `json:"scraperPubKey"`
		Users            []EnvUser `json:"users"`
	}{}
	err := api.connections[0].Get("/api/v1/scraperUsers/"+apiKeyID, &scraperUsers)
	if err != nil {
		// Wait for the connections above to complete checking before we do this error check but do the request already so we don't have to wait for that
		// If one of the connections has an error they will throw
		wg.Wait()
		log.Fatal(err)
	}

	if scraperUsers.ScraperPublicKey != "" && scraperUsers.ScraperPublicKey != decryptionKey.PublicBase64 {
		log.Fatal("the env.json provided contains a diffrent public key than registered in RTCV, scraper users won't be able to be decrypted")
	}

	loginUsers := []EnvUser{}
	for _, user := range scraperUsers.Users {
		if user.EncryptedPassword != "" {
			user.Password, err = decryptionKey.DecryptScraperPassword(user.EncryptedPassword)
			if err != nil {
				log.Fatal("unable to decrypt password for user " + user.Username + ", error: " + err.Error())
			}
			loginUsers = append(loginUsers, user)
		} else if user.Password != "" {
			loginUsers = append(loginUsers, user)
		} else {
			fmt.Println("WARN: unusable login user", user.Username)
		}
	}

	wg.Wait()

	return loginUsers
}

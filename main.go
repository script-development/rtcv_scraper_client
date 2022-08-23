package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/script-development/rtcv_scraper_client/v2/crypto"
)

// Env contains the structure of the .env file
type Env struct {
	PrivateKey         string      `json:"private_key"`
	PublicKey          string      `json:"public_key"`
	PrimaryServer      EnvServer   `json:"primary_server"`
	AlternativeServers []EnvServer `json:"alternative_servers"`
	MockMode           bool        `json:"mock_mode"`
	MockUsers          []EnvUser   `json:"mock_users"`
}

func (e *Env) validate() error {
	if e.MockMode {
		if len(e.MockUsers) == 0 {
			fmt.Println(`"mock_users" is empty in env.json, most scrapers require at least one user to login.`)
			fmt.Println(`For documentation about mocking see https://github.com/script-development/rtcv_scraper_client`)
			if e.MockUsers == nil {
				e.MockUsers = []EnvUser{}
			}
		}
		return nil
	}

	err := e.PrimaryServer.validate()
	if err != nil {
		return fmt.Errorf("primary_server.%s", err.Error())
	}

	for idx, server := range e.AlternativeServers {
		err := server.validate()
		if err != nil {
			return fmt.Errorf("%s[%d].%s", "alternative_servers", idx, err.Error())
		}
	}

	keyPairHelpMsg := `, use the go program inside the "gen_key" folder to generate a key pair`
	if e.PrivateKey == "" && e.PublicKey == "" {
		return errors.New(`"public_key" and "private_key" are required` + keyPairHelpMsg)
	} else if e.PrivateKey == "" {
		return errors.New(`"private_key" required` + keyPairHelpMsg)
	} else if e.PublicKey == "" {
		return errors.New(`"public_key" required` + keyPairHelpMsg)
	}

	return nil
}

// EnvServer contains the structure of the primary_server and alternative_servers inside the .env file
type EnvServer struct {
	ServerLocation string `json:"server_location"`
	APIKeyID       string `json:"api_key_id"`
	APIKey         string `json:"api_key"`
}

func (e *EnvServer) validate() error {
	if e.ServerLocation == "" {
		return errors.New("server_location is required")
	}
	if e.APIKeyID == "" {
		return errors.New("api_key_id is required")
	}
	if e.APIKey == "" {
		return errors.New("api_key is required")
	}

	return nil
}

func (e *EnvServer) toCredArg(isPrimary bool) SetCredentialsArg {
	return SetCredentialsArg{
		ServerLocation: e.ServerLocation,
		APIKeyID:       e.APIKeyID,
		APIKey:         e.APIKey,
		Primary:        isPrimary,
	}
}

// EnvUser contains the structure of the login_users inside the .env file
type EnvUser struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	EncryptedPassword string `json:"encryptedPassword,omitempty"`
}

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

	useAddress := startWebserver(env, api, loginUsers)

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

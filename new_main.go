package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Env contains the structure of the .env file
type Env struct {
	PrimaryServer      EnvServer   `json:"primary_server"`
	AlternativeServers []EnvServer `json:"alternative_servers"`
	LoginUsers         []EnvUser   `json:"login_users"`
}

func (e *Env) validate() error {
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
	Username string `json:"username"`
	Password string `json:"password"`
}

func newMain() {
	envFile, err := ioutil.ReadFile("env.json")
	if err != nil {
		log.Fatal("unable to read env file, error: " + err.Error())
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
	go startWebserver(env, api)

	credentials := []SetCredentialsArg{env.PrimaryServer.toCredArg(true)}
	for _, server := range env.AlternativeServers {
		credentials = append(credentials, server.toCredArg(false))
	}

	err = api.SetCredentials(credentials)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("credentials set")
	fmt.Println("testing connections..")
	testServerConnections(api)
	fmt.Println("connected to RTCV")
	fmt.Println("running scraper..")

	if len(os.Args) <= 1 {
		log.Fatal("must provide a command to run, for example: rtcv_scraper_client npm run scraper")
	}

	scraper := exec.Command(os.Args[1], os.Args[2:]...)
	scraper.Env = append(os.Environ(), "SCRAPER_ADDRESS=http://localhost:4400")

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

func testServerConnections(api *API) {
	for _, conn := range api.connections {
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
	}

	referenceNrs := []string{}
	err := api.connections[api.primaryConnection].Get("/api/v1/scraper/scannedReferenceNrs/since/days/30", &referenceNrs)
	if err != nil {
		log.Fatal(err)
	}

	for _, nr := range referenceNrs {
		api.SetCacheEntry(nr, time.Hour*72) // 3 days
	}
}

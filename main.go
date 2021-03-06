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
)

// Env contains the structure of the .env file
type Env struct {
	PrimaryServer      EnvServer   `json:"primary_server"`
	AlternativeServers []EnvServer `json:"alternative_servers"`
	LoginUsers         []EnvUser   `json:"login_users"`
	MockMode           bool        `json:"mock_mode"`
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

func main() {
	envFilename := "env.json"
	envEnvName := "RTCV_SCRAPER_CLIENT_ENV"
	envFile, err := ioutil.ReadFile(envFilename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal("unable to read env file, error: " + err.Error())
		}
		envFile = []byte(os.Getenv(envEnvName))
		if len(envFile) == 0 {
			log.Fatalf("no %s file or %s envourment variable found, cannot continue", envFilename, envEnvName)
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
	useAddress := startWebserver(env, api)

	credentials := []SetCredentialsArg{env.PrimaryServer.toCredArg(true)}
	for _, server := range env.AlternativeServers {
		credentials = append(credentials, server.toCredArg(false))
	}

	// Turn on mock mode by default
	api.SetMockMode()

	if !env.MockMode {
		err = api.SetCredentials(credentials)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("credentials set")
		fmt.Println("testing connections..")
		testServerConnections(api)
		fmt.Println("connected to RTCV")
	} else {
		fmt.Println("In mock mode")
		fmt.Println("You can turn this off in by setting mock_mode to false in your env.json")
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
}

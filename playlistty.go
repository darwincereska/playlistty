package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Spotify struct {
		APIKey       string `yaml:"api_key"`
		UserID       string `yaml:"user_id"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	} `yaml:"spotify"`
	YouTube struct {
		APIKey       string `yaml:"api_key"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	} `yaml:"youtube"`
}
type Flags struct {
	Service    string
	ConfigPath string
}

func ParseFlags() (*Flags, error) {
	flags := &Flags{}

	// Define flags
	flag.StringVar(&flags.Service, "service", "", "Service to use (spotify/yt)")
	flag.StringVar(&flags.ConfigPath, "config", "./config/config.yml", "Path to config file")

	// Parse flags
	flag.Parse()

	// Validate service flag
	if flags.Service != "spotify" && flags.Service != "yt" {
		return nil, fmt.Errorf("invalid service: must be 'spotify' or 'yt'")
	}

	return flags, nil
}

func ParseConfig(configPath string) (*Config, error) {
	// read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parseing config file: %v", err)
	}
	return &config, nil
}

func getSpotifyAPIKey(client string, secret string) {
	// Create the request URL and data
	url := "https://accounts.spotify.com/api/token"
	body := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", client, secret)

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Parse the response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	// Read existing config
	configData, err := os.ReadFile("config/config.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		return
	}

	// Parse existing config
	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n", err)
		return
	}

	// Update API key
	config.Spotify.APIKey = tokenResp.AccessToken

	// Write updated config back to file
	newConfigData, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		return
	}

	if err := os.WriteFile("config/config.yml", newConfigData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		return
	}

	fmt.Println("Successfully updated Spotify API key in config.yml")
}

func main() {
	// parse flags
	flags, err := ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// parse config
	config, err := ParseConfig(flags.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	ListPlaylists := func(service string) {
		switch service {
		case "spotify":
			url := fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists", config.Spotify.UserID)
			client := &http.Client{}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
				return
			}

			// Add Authorization header
			req.Header.Add("Authorization", "Bearer "+config.Spotify.APIKey)

			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			// Print raw response for debugging
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
				fmt.Fprintf(os.Stderr, "Raw response: %+v\n", resp)
				return
			}

			items, ok := result["items"].([]interface{})
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: unexpected response format\n")
				fmt.Fprintf(os.Stderr, "Raw response data: %+v\n", result)
				return
			}

			fmt.Println("\nSpotify Playlists:")
			fmt.Println("------------------")
			for _, item := range items {
				playlist, ok := item.(map[string]interface{})
				if !ok {
					fmt.Fprintf(os.Stderr, "Error: unexpected playlist format\n")
					continue
				}
				fmt.Printf("Name: %s\n", playlist["name"])
				fmt.Printf("ID: %s\n", playlist["id"])
				if tracks, ok := playlist["tracks"].(map[string]interface{}); ok {
					fmt.Printf("Tracks: %v\n", tracks["total"])
				}
				fmt.Println("------------------")
			}
		case "youtube":
			config := &oauth2.Config{
				ClientID:     config.YouTube.ClientID,
				ClientSecret: config.YouTube.ClientSecret,
				Scopes: []string{
					"https://www.googleapis.com/auth/youtube.readonly",
				},
				Endpoint:    google.Endpoint,
				RedirectURL: "http://localhost:3000/callback",
			}

			// Start OAuth flow
			authURL := config.AuthCodeURL("state")
			fmt.Printf("Go to the following URL for authorization: \n%v\n", authURL)

			// Start local server to receive callback
			var authCode string
			http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
				authCode = r.URL.Query().Get("code")
				fmt.Fprintf(w, "Authorization successful! You can close this window.")
			})
			go http.ListenAndServe(":3000", nil)

			// Wait for auth code
			fmt.Print("Waiting for authorization...")
			for authCode == "" {
				time.Sleep(time.Second)
			}

			// Exchange auth code for token
			token, err := config.Exchange(context.Background(), authCode)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error exchanging auth code: %v\n", err)
				return
			}

			client := config.Client(context.Background(), token)
			url := "https://youtube.googleapis.com/youtube/v3/playlists?part=snippet&mine=true"

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
				return
			}

			items, ok := result["items"].([]interface{})
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: unexpected response format\n")
				return
			}

			fmt.Println("\nYouTube Playlists:")
			fmt.Println("------------------")
			for _, item := range items {
				playlist, ok := item.(map[string]interface{})
				if !ok {
					fmt.Fprintf(os.Stderr, "Error: unexpected playlist format\n")
					continue
				}
				snippet, ok := playlist["snippet"].(map[string]interface{})
				if !ok {
					continue
				}
				fmt.Printf("Title: %s\n", snippet["title"])
				fmt.Printf("ID: %s\n", playlist["id"])
				fmt.Printf("Description: %s\n", snippet["description"])
				fmt.Println("------------------")
			}
		}
	}

	// Get api keys based on service
	var hostKey string
	switch flags.Service {
	case "spotify":
		getSpotifyAPIKey(config.Spotify.ClientID, config.Spotify.ClientSecret)
		hostKey = config.Spotify.APIKey
		ListPlaylists("spotify")
	case "yt":
		hostKey = config.YouTube.APIKey
		ListPlaylists("youtube")
	}
	fmt.Printf("using %s API KEY: %s\n", flags.Service, hostKey)

}

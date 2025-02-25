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
	// "time"
)
const configFile = "config/config.yml"

type Config struct {
	Spotify struct {
		APIKey       string `yaml:"api_key"`
		UserID       string `yaml:"user_id"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	} `yaml:"spotify"`
	YouTube struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		Token string `yaml:"token"`
	} `yaml:"youtube"`
}
type Flags struct {
	Service    string
	ConfigPath string
	OAuthService string
}

func ParseFlags() (*Flags, error) {
	flags := &Flags{}
	services := []string{"spotify", "yt"}
	// Define flags
	flag.StringVar(&flags.Service, "service", "", "Service to use $(services)")
	flag.StringVar(&flags.ConfigPath, "config", configFile, "Path to config file")
	flag.StringVar(&flags.OAuthService, "oauth", "", "Generate OAuth Token for service")
	// Parse flags
	flag.Parse()

	// Validate service flag
	found := false
	for _, service := range services {
		if flags.Service == service {
			found = true
			break
		}
		if flags.OAuthService == service {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("invalid service: must be one of %v", services)
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

func GetSpotifyAPIKey(client string, secret string) {
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
	configData, err := os.ReadFile(configFile)
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

	if err := os.WriteFile(configFile, newConfigData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		return
	}

	fmt.Println("Successfully updated Spotify API key in config.yml")
}
func GenerateOAuthToken(service string) (*Config, error) {
	// First read config to get client credentials
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %v", err) 
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("error parsing config: %v", err)
	}

	switch service {
	case "youtube":
		// OAuth2 config
		gConfig := &oauth2.Config{
			ClientID:     config.YouTube.ClientID,
			ClientSecret: config.YouTube.ClientSecret,
			RedirectURL: "http://localhost:3000/callback",
			Scopes: []string{"https://www.googleapis.com/auth/youtube"},
			Endpoint: google.Endpoint,
		}

		authURL := gConfig.AuthCodeURL("state")
		fmt.Printf("Visit this URL to get an authorization code:\n%v\n", authURL)

		var code string
		fmt.Print("Enter the authorization code: ")
		fmt.Scan(&code)

		ctx := context.Background()
		token, err := gConfig.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("error exchanging code: %v", err)
		}

		// Update token
		config.YouTube.Token = token.AccessToken

		// Write updated config
		newConfigData, err := yaml.Marshal(&config)
		if err != nil {
			return nil, fmt.Errorf("error marshaling config: %v", err)
		}

		if err := os.WriteFile(configFile, newConfigData, 0644); err != nil {
			return nil, fmt.Errorf("error writing config: %v", err)
		}

		fmt.Println("Successfully updated YouTube token in config.yml")
		return &config, nil

	default:
		return nil, fmt.Errorf("unsupported service: %s", service)
	}
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
	
	switch flags.OAuthService {
	case "yt":
		GenerateOAuthToken("youtube")
	}
	
	
	// Get api keys based on service
	var hostKey string
	switch flags.Service {
	case "spotify":
		GetSpotifyAPIKey(config.Spotify.ClientID, config.Spotify.ClientSecret)
		hostKey = config.Spotify.APIKey
		// ListPlaylists("spotify")
		fmt.Printf("using %s API KEY: %s\n", flags.Service, hostKey)
	case "yt":
		hostKey = config.YouTube.Token
		fmt.Printf("using %s API KEY: %s\n", flags.Service, hostKey)
	}
	

}

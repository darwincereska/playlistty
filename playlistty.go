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
		Token        string `yaml:"token"`
	} `yaml:"youtube"`
}
type Flags struct {
	Service      string
	ConfigPath   string
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
			RedirectURL:  "http://localhost:3000/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/youtube"},
			Endpoint:     google.Endpoint,
		}

		// Start HTTP server to handle the OAuth callback
		codeChan := make(chan string)
		srv := &http.Server{Addr: ":3000"}

		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			codeChan <- code
			fmt.Fprintf(w, "Authorization successful! You can close this window.")
			go func() {
				srv.Shutdown(context.Background())
			}()
		})

		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				fmt.Printf("HTTP server error: %v\n", err)
			}
		}()

		authURL := gConfig.AuthCodeURL("state")
		fmt.Printf("Opening browser for authorization...\n")
		fmt.Printf("Please visit this URL to authorize: %v\n", authURL)

		// Wait for the authorization code
		code := <-codeChan

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

		fmt.Printf("Sucessfully added YouTube token in: %s\n", configFile)
		return &config, nil

	default:
		return nil, fmt.Errorf("unsupported service: %s", service)
	}
}

func ListPlaylists(service string) (config *Config) {
	switch service {
	case "spotify":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return nil
		}

		// Create request URL for Spotify playlists endpoint
		url := fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists", config.Spotify.UserID)

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return nil
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.Spotify.APIKey)

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return nil
		}
		defer resp.Body.Close()

		// Parse response
		var playlists struct {
			Items []struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&playlists); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return nil
		}

		// Print playlists
		fmt.Println("Your Spotify playlists:")
		for _, playlist := range playlists.Items {
			fmt.Printf("- %s (ID: %s)\n", playlist.Name, playlist.ID)
		}

		return config

	case "youtube":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return nil
		}

		// Create request URL for YouTube playlists endpoint
		url := "https://www.googleapis.com/youtube/v3/playlists?part=snippet&mine=true"

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return nil
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.YouTube.Token)

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return nil
		}
		defer resp.Body.Close()

		// Parse response
		var playlists struct {
			Items []struct {
				Snippet struct {
					Title string `json:"title"`
				} `json:"snippet"`
				Id string `json:"id"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&playlists); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return nil
		}

		// Print playlists
		fmt.Println("Your YouTube playlists:")
		for _, playlist := range playlists.Items {
			fmt.Printf("- %s (ID: %s)\n", playlist.Snippet.Title, playlist.Id)
		}

		return config

	default:
		fmt.Printf("Service %s not supported\n", service)
		return nil
	}
}

func ReadPlaylist(service string, playlist string) {
	switch service {
	case "spotify":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}

		// Create request URL for Spotify playlist tracks endpoint
		url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", playlist)

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.Spotify.APIKey)

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// Parse playlist response to get name
		var playlistData struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&playlistData); err != nil {
			fmt.Printf("Error decoding playlist response: %v\n", err)
			return
		}

		// Get tracks URL
		tracksURL := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlist)

		// Create tracks request
		req, err = http.NewRequest("GET", tracksURL, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.Spotify.APIKey)

		// Make request
		resp, err = client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// Parse response
		var tracks struct {
			Items []struct {
				Track struct {
					Name    string `json:"name"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"track"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return
		}

		// Print tracks
		fmt.Printf("Tracks in playlist (%s):\n", playlistData.Name)
		for _, item := range tracks.Items {
			artists := make([]string, len(item.Track.Artists))
			for i, artist := range item.Track.Artists {
				artists[i] = artist.Name
			}
			fmt.Printf("- %s by %s\n", item.Track.Name, strings.Join(artists, ", "))
		}
	case "youtube":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}

		var allVideos []struct {
			Snippet struct {
				Title    string `json:"title"`
				VideoId  string `json:"resourceId.videoId"`
				Position int    `json:"position"`
			} `json:"snippet"`
		}

		pageToken := ""
		for {
			// Create request URL for YouTube playlist tracks endpoint
			url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50", playlist)
			if pageToken != "" {
				url += "&pageToken=" + pageToken
			}

			// Create request
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				return
			}

			// Add authorization header
			req.Header.Add("Authorization", "Bearer "+config.YouTube.Token)

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error making request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			// Parse response
			var videos struct {
				NextPageToken string `json:"nextPageToken"`
				Items         []struct {
					Snippet struct {
						Title    string `json:"title"`
						VideoId  string `json:"resourceId.videoId"`
						Position int    `json:"position"`
					} `json:"snippet"`
				} `json:"items"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
				fmt.Printf("Error decoding response: %v\n", err)
				return
			}

			allVideos = append(allVideos, videos.Items...)

			if videos.NextPageToken == "" {
				break
			}
			pageToken = videos.NextPageToken
		}

		// Print videos
		fmt.Printf("Videos in playlist:\n")
		for _, item := range allVideos {
			fmt.Printf("- %s (Position: %d)\n", item.Snippet.Title, item.Snippet.Position)
		}
	default:
		fmt.Printf("Service %s not supported\n", service)
	}
}

func UpdatePlaylist(service string, playlist string) {

}

func CreatePlaylist(service string, title string) {

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
	case "spotify":
		GetSpotifyAPIKey(config.Spotify.ClientID, config.Spotify.ClientSecret)
	}

	// Get api keys based on service
	// var hostKey string
	switch flags.Service {
	case "spotify":
		ListPlaylists("spotify")
		// ReadPlaylist("spotify", "2l2L7lKGrdLC6TXcPLdfKv")
	case "yt":
		// ListPlaylists("youtube")
		ReadPlaylist("youtube", "PLe0T9j3Sn3hnvtiIxpfx2a0SIy84VSSl7")

	}

}

package main

import (
	"bufio"
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
)

const storageDir = "config"
const configFile = storageDir + "/config.yml"

type Config struct {
	Spotify struct {
		UserID       string `yaml:"user_id"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		Token        string `yaml:"token"`
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
type App struct {
	HostService       string
	HostValidated     bool
	HostPlaylist      string
	TargetService     string
	CreateNewPlaylist bool
	TargetName        string
	TargetID          string
}

func Run(service string) *App {
	app := &App{}
	platforms := []string{"spotify", "youtube"}
	fmt.Println("Welcome to playlistty!")
	// Checks for config file
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		err := os.MkdirAll(storageDir, 0755)
		if err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return app
		}

		// Create empty config struct
		config := Config{}

		// Marshal to YAML
		data, err := yaml.Marshal(&config)
		if err != nil {
			fmt.Printf("Error creating config file: %v\n", err)
			return app
		}

		// Write config file
		err = os.WriteFile(configFile, data, 0644)
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return app
		}

		fmt.Printf("Created new config file: %s\n", configFile)
	}
	switch service {
	case "spotify":
		app.HostService = "spotify"
	case "youtube":
		app.HostService = "youtube"
	}

	// Signin
	ValidateToken(app.HostService)
	app.HostValidated = true

	// List playlists
	ListPlaylists(app.HostService)

	// Choose playlist
	fmt.Printf("\nEnter Playlist id: ")
	fmt.Scan(&app.HostPlaylist)

	// Ask for target platform
	fmt.Println("\nWhat platform do you want to import to?")
	for i, platform := range platforms {
		fmt.Printf("%d. %s\n", i+1, platform)
	}
	var choice int
	fmt.Printf("Enter number 1-%d: ", len(platforms))
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(platforms) {
		fmt.Println("Invalid choice")
		return app
	}
	app.TargetService = platforms[choice-1]

	// Read and parse host playlist
	fmt.Printf("Parsing playlist: %s\n", app.HostPlaylist)
	ReadPlaylist(app.HostService, app.HostPlaylist)
	if app.HostService != "youtube" || app.TargetService != "youtube" { 
		PlaylistFile := storageDir + "/" + app.HostService + "/" + app.HostPlaylist + ".json"
		FindTrackIDFromFile(app.TargetService, PlaylistFile)
	}

	// ask to create or use existing playlist
	fmt.Println("Do you want to create a new playlist?")
	fmt.Println("1. Yes")
	fmt.Println("2. No")
	var CreateNewPlaylist int
	fmt.Scan(&CreateNewPlaylist)
	switch CreateNewPlaylist {
	case 1:
		app.CreateNewPlaylist = true
	case 2:
		app.CreateNewPlaylist = false
	}
	switch app.CreateNewPlaylist {
	case true:
		fmt.Printf("Provide a name for the playlist: ")
		fmt.Printf("playlist name: ")
		app.TargetName = ""
		reader := bufio.NewReader(os.Stdin)
		app.TargetName, _ = reader.ReadString('\n')
		app.TargetName = strings.TrimSpace(app.TargetName)
		fmt.Println("Playlist will default to private")
		CreatePlaylist(app.TargetService, app.TargetName, "Made with Playlistty", false)
	}
	fmt.Println("WARNING IT WILL CLEAR PLAYLIST")
	fmt.Println("Choose target playlist:")
	ListPlaylists(app.TargetService)
	fmt.Printf("\nEnter Playlist id: ")
	fmt.Scan(&app.TargetID)
	ClearPlaylist(app.TargetService, app.TargetID)
	fmt.Printf("Transferring playlist: %s\n", app.TargetID)

	// Update playlist
	UpdatePlaylist(app.TargetService, app.TargetID, app.TargetService, app.HostService, app.HostPlaylist)
	return app
}

func Setup() {

}

func ValidateToken(service string) {
	switch service {
	case "spotify":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}

		// Test token by making a request
		url := "https://api.spotify.com/v1/me"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}

		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			fmt.Println("Token is valid")
			return
		} else {
			GenerateOAuthToken("spotify")
			return
		}
	case "youtube":
	}
}

func ParseFlags() (*Flags, error) {
	flags := &Flags{}
	services := []string{"spotify", "yt"}
	// Define flags
	flag.StringVar(&flags.Service, "service", "", "spotify/yt")
	flag.StringVar(&flags.ConfigPath, "config", configFile, "Path to config file")
	flag.StringVar(&flags.OAuthService, "oauth", "", "Generate OAuth Token for service")
	helpFlag := flag.Bool("help", false, "Shows help screen")
	// Parse flags
	flag.Parse()
	if flags.Service != "" || flags.OAuthService != "" {
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
	}
	switch *helpFlag {
	case true:
		flag.Usage()
	case flags.Service != "" && *helpFlag == false:
		flag.Usage()
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
	case "spotify":
		// OAuth2 config
		gConfig := &oauth2.Config{
			ClientID:     config.Spotify.ClientID,
			ClientSecret: config.Spotify.ClientSecret,
			RedirectURL:  "http://localhost:3000/callback",
			Scopes: []string{
				"playlist-modify-public",
				"playlist-modify-private",
				"playlist-read-private",
				"playlist-read-collaborative",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.spotify.com/authorize",
				TokenURL: "https://accounts.spotify.com/api/token",
			},
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

		// Update token in config
		config.Spotify.Token = token.AccessToken

		// Write updated config
		newConfigData, err := yaml.Marshal(&config)
		if err != nil {
			return nil, fmt.Errorf("error marshaling config: %v", err)
		}

		if err := os.WriteFile(configFile, newConfigData, 0644); err != nil {
			return nil, fmt.Errorf("error writing config: %v", err)
		}

		fmt.Printf("Successfully added Spotify token in: %s\n", configFile)
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
		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

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
		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

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
		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

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

		// Extract songs into list
		var songList []map[string]string
		for _, item := range tracks.Items {
			artists := make([]string, len(item.Track.Artists))
			for i, artist := range item.Track.Artists {
				artists[i] = artist.Name
			}
			songData := map[string]string{
				"name":   item.Track.Name,
				"artist": strings.Join(artists, ", "),
			}
			songList = append(songList, songData)
		}

		// Create storage directory if it doesn't exist
		os.MkdirAll(storageDir+"/spotify", 0755)
		filePath := fmt.Sprintf("%s/spotify/%s.json", storageDir, playlist)

		// Write to file
		jsonData, err := json.MarshalIndent(songList, "", "    ")
		if err != nil {
			fmt.Printf("Error marshaling song data: %v\n", err)
			return
		}

		if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
			fmt.Printf("Error writing song data file: %v\n", err)
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

		var songList []map[string]string
		var nextPageToken string

		for {
			// Create request URL
			listURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&maxResults=50&playlistId=%s", playlist)
			if nextPageToken != "" {
				listURL = fmt.Sprintf("%s&pageToken=%s", listURL, nextPageToken)
			}

			// Create request
			req, err := http.NewRequest("GET", listURL, nil)
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
			var result struct {
				NextPageToken string `json:"nextPageToken"`
				Items []struct {
					Snippet struct {
						Title       string `json:"title"`
						ChannelTitle string `json:"videoOwnerChannelTitle"`
						ResourceId struct {
							VideoId string `json:"videoId"`
						} `json:"resourceId"`
					} `json:"snippet"`
				} `json:"items"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Printf("Error decoding response: %v\n", err)
				return
			}

			// Extract songs from this page
			for _, item := range result.Items {
				songData := map[string]string{
					"name": item.Snippet.Title,
					"artist": strings.TrimSuffix(item.Snippet.ChannelTitle, " - Topic"),
					"id":   item.Snippet.ResourceId.VideoId,
				}
				songList = append(songList, songData)
			}

			nextPageToken = result.NextPageToken
			if nextPageToken == "" {
				break
			}
		}

		// Create storage directory if it doesn't exist
		os.MkdirAll(storageDir+"/youtube", 0755)
		filePath := fmt.Sprintf("%s/youtube/%s.json", storageDir, playlist)

		// Write to file
		jsonData, err := json.MarshalIndent(songList, "", "    ")
		if err != nil {
			fmt.Printf("Error marshaling song data: %v\n", err)
			return
		}

		if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
			fmt.Printf("Error writing song data file: %v\n", err)
			return
		}

		// Print tracks
		fmt.Println("Tracks in playlist:")
		for _, song := range songList {
			fmt.Printf("- %s by %s (ID: %s)\n", song["name"], song["artist"], song["id"])
		}
	default:
		fmt.Printf("Service %s not supported\n", service)
	}
}

func FindTrackIDFromFile(targetService string, file string) {
	// Read song data from file
	songData, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading song file: %v\n", err)
		return
	}

	// Parse song data into slice of maps
	var songs []map[string]string
	if err := json.Unmarshal(songData, &songs); err != nil {
		fmt.Printf("Error parsing song data: %v\n", err)
		return
	}

	// Search for each song and add ID
	for i := range songs {
		name := songs[i]["name"]
		artist := songs[i]["artist"]

		id := SearchSong(targetService, name, artist)
		if id != "" {
			songs[i]["id"] = id
		}
	}

	// Write updated data back to file
	updatedData, err := json.MarshalIndent(songs, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling updated song data: %v\n", err)
		return
	}

	if err := os.WriteFile(file, updatedData, 0644); err != nil {
		fmt.Printf("Error writing updated song data: %v\n", err)
		return
	}

}

func UpdatePlaylist(service string, playlist string, mode string, folder string, trackID string) {
	switch service {
	case "spotify":
		switch mode {
		case "add":
			// Parse config file
			config, err := ParseConfig(configFile)
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				return
			}

			// Create request URL for Spotify playlists endpoint
			url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlist)

			// Create request body with track URI
			requestBody := map[string]interface{}{
				"uris": []string{fmt.Sprintf("spotify:track:%s", trackID)},
			}
			bodyJSON, err := json.Marshal(requestBody)
			if err != nil {
				fmt.Printf("Error marshaling request body: %v\n", err)
				return
			}

			// Create request
			req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyJSON)))
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				return
			}

			// Add authorization header
			req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)
			req.Header.Add("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error making request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 201 {
				fmt.Printf("Successfully added track to playlist\n")
			} else {
				fmt.Printf("Error adding track to playlist: %s\n", resp.Status)
			}
		case "spotify":
			// Parse config file
			config, err := ParseConfig(configFile)
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				return
			}

			// Read song data from file
			filePath := fmt.Sprintf("%s/%s/%s.json", storageDir,folder, trackID)
			songData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading song data file: %v\n", err)
				return
			}

			// Parse song data
			var songs []map[string]string
			if err := json.Unmarshal(songData, &songs); err != nil {
				fmt.Printf("Error parsing song data: %v\n", err)
				return
			}

			// Add each track to playlist
			for _, song := range songs {
				if song["id"] != "" {
					// Create request URL
					url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlist)

					// Create request body
					requestBody := map[string]interface{}{
						"uris": []string{fmt.Sprintf("spotify:track:%s", song["id"])},
					}
					bodyJSON, err := json.Marshal(requestBody)
					if err != nil {
						fmt.Printf("Error marshaling request body: %v\n", err)
						continue
					}

					// Create request
					req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyJSON)))
					if err != nil {
						fmt.Printf("Error creating request: %v\n", err)
						continue
					}

					// Add headers
					req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)
					req.Header.Add("Content-Type", "application/json")

					// Make request
					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						fmt.Printf("Error making request: %v\n", err)
						continue
					}
					defer resp.Body.Close()

					if resp.StatusCode == 201 {
						fmt.Printf("Added track: %s by %s\n", song["name"], song["artist"])
					} else {
						fmt.Printf("Error adding track %s: %s\n", song["name"], resp.Status)
					}
				}
			}

			fmt.Println("Finished adding tracks to playlist")

		}
	case "youtube":
		switch mode {
		case "youtube":
			// Parse config file
			config, err := ParseConfig(configFile)
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				return
			}

			// Read song data from file
			filePath := fmt.Sprintf("%s/%s/%s.json", storageDir,folder, trackID)
			songData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading song data file: %v\n", err)
				return
			}

			// Parse song data
			var songs []map[string]string
			if err := json.Unmarshal(songData, &songs); err != nil {
				fmt.Printf("Error parsing song data: %v\n", err)
				return
			}

			// Collect all video IDs
			var videoIds []string
			for _, song := range songs {
				if song["id"] != "" {
					videoIds = append(videoIds, song["id"])
				}
			}

			// Add tracks in batches of 25
			url := "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet"
			client := &http.Client{}

			// Create batch request body with up to 25 videos
			for i := 0; i < len(videoIds); i += 25 {
				end := i + 25
				if end > len(videoIds) {
					end = len(videoIds)
				}

				batch := make([]interface{}, 0)
				for _, videoId := range videoIds[i:end] {
					batch = append(batch, map[string]interface{}{
						"snippet": map[string]interface{}{
							"playlistId": playlist,
							"resourceId": map[string]string{
								"kind":    "youtube#video",
								"videoId": videoId,
							},
						},
					})
				}

				// Send single request with batch of videos
				bodyJSON, err := json.Marshal(map[string]interface{}{
					"items": batch,
				})
				if err != nil {
					fmt.Printf("Error marshaling request body: %v\n", err)
					continue 
				}

				req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyJSON)))
				if err != nil {
					fmt.Printf("Error creating request: %v\n", err) 
					continue
				}

				req.Header.Add("Authorization", "Bearer "+config.YouTube.Token)
				req.Header.Add("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					fmt.Printf("Error making request: %v\n", err)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode == 200 {
					fmt.Printf("Added videos %d-%d\n", i+1, end)
				} else {
					fmt.Printf("Error adding videos %d-%d: %s\n", i+1, end, resp.Status)
				}
			}

			fmt.Println("Finished adding tracks to playlist")

		}
	}
}

func ClearPlaylist(service string, playlist string) {
	switch service {
	case "spotify":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}

		// Get all tracks in playlist first
		getUrl := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlist)
		req, err := http.NewRequest("GET", getUrl, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}
		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var tracks struct {
			Items []struct {
				Track struct {
					URI string `json:"uri"`
				} `json:"track"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return
		}

		// Delete tracks in batches of 100
		deleteUrl := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlist)
		for i := 0; i < len(tracks.Items); i += 100 {
			end := i + 100
			if end > len(tracks.Items) {
				end = len(tracks.Items)
			}

			// Create array of track URIs to delete
			var trackList []map[string]interface{}
			for _, item := range tracks.Items[i:end] {
				trackList = append(trackList, map[string]interface{}{
					"uri": item.Track.URI,
				})
			}

			requestBody := map[string]interface{}{
				"tracks": trackList,
			}
			bodyJSON, err := json.Marshal(requestBody)
			if err != nil {
				fmt.Printf("Error marshaling request body: %v\n", err)
				return
			}

			req, err := http.NewRequest("DELETE", deleteUrl, strings.NewReader(string(bodyJSON)))
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				return
			}

			req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)
			req.Header.Add("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error making request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				fmt.Printf("Error clearing playlist batch: %s\n", resp.Status)
				return
			}
		}

		fmt.Printf("Successfully cleared playlist\n")
	}
}

func CreatePlaylist(service string, title string, description string, public bool) {
	switch service {
	case "spotify":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
		}

		// Create request URL for Spotify playlists endpoint
		url := fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists", config.Spotify.UserID)

		// Create request body
		requestBody := map[string]interface{}{
			"name":        title,
			"description": description,
			"public":      public,
		}
		bodyJSON, err := json.Marshal(requestBody)
		if err != nil {
			fmt.Printf("Error marshaling request body: %v\n", err)
			return
		}

		// Create request
		req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyJSON)))
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return
		}
		defer resp.Body.Close()
	case "youtube":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return
		}
	
		// Create request URL for YouTube playlists endpoint
		url := "https://www.googleapis.com/youtube/v3/playlists?part=snippet"
	
		// Create request body
		requestBody := map[string]interface{}{
			"snippet": map[string]interface{}{
				"title":       title,
				"description": description,
				"privacyStatus": func() string {
					if public {
						return "public"
					}
					return "private"
				}(),
			},
		}
	
		bodyJSON, err := json.Marshal(requestBody)
		if err != nil {
			fmt.Printf("Error marshaling request body: %v\n", err)
			return
		}
	
		// Create request
		req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyJSON)))
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}
	
		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.YouTube.Token)
		req.Header.Add("Content-Type", "application/json")
	
		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return
		}
		defer resp.Body.Close()
	
		if resp.StatusCode != 200 {
			fmt.Printf("Error creating playlist: %s\n", resp.Status)
			return
		}
	
		fmt.Printf("Successfully created playlist %s\n", title)

	}
}

func SearchSong(service string, song string, artist string) string {
	switch service {
	case "spotify":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return ""
		}

		// Create search query
		query := fmt.Sprintf("track:%s artist:%s", song, artist)
		encodedQuery := strings.ReplaceAll(query, " ", "%20")
		url := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=1", encodedQuery)

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return ""
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.Spotify.Token)

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return ""
		}
		defer resp.Body.Close()

		// Parse response
		var result struct {
			Tracks struct {
				Items []struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"items"`
			} `json:"tracks"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return ""
		}

		// Print results
		fmt.Println("Search results:")
		if len(result.Tracks.Items) > 0 {
			track := result.Tracks.Items[0]
			artists := make([]string, len(track.Artists))
			for j, artist := range track.Artists {
				artists[j] = artist.Name
			}
			fmt.Printf("%s by %s (ID: %s)\n", track.Name, strings.Join(artists, ", "), track.ID)
		}

		// Return first track ID if found
		if len(result.Tracks.Items) > 0 {
			return result.Tracks.Items[0].ID
		}
		return ""
	case "youtube":
		// Parse config file
		config, err := ParseConfig(configFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			return ""
		}

		// Create search query
		query := fmt.Sprintf("%s %s", song, artist)
		encodedQuery := strings.ReplaceAll(query, " ", "%20")
		url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&maxResults=1&q=%s&type=video", encodedQuery)

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return ""
		}

		// Add authorization header
		req.Header.Add("Authorization", "Bearer "+config.YouTube.Token)

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			return ""
		}
		defer resp.Body.Close()

		// Parse response
		var result struct {
			Items []struct {
				Id struct {
					VideoId string `json:"videoId"`
				} `json:"id"`
				Snippet struct {
					Title        string `json:"title"`
					ChannelTitle string `json:"channelTitle"`
				} `json:"snippet"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return ""
		}

		// Print results
		fmt.Println("Search results:")
		if len(result.Items) > 0 {
			video := result.Items[0]
			fmt.Printf("%s by %s (ID: %s)\n", video.Snippet.Title, video.Snippet.ChannelTitle, video.Id.VideoId)
			return video.Id.VideoId
		}

		return ""
	}
	return ""
}

func main() {
	// parse flags
	flags, err := ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}
	// OAuth runner
	switch flags.OAuthService {
	case "yt":
		GenerateOAuthToken("youtube")
	case "spotify":
		GenerateOAuthToken("spotify")
	}

	// Runs migrate process for host service
	switch flags.Service {
	case "spotify":
		Run("spotify")
	case "yt":
		Run("youtube")
		// ListPlaylists("youtube")
		// SearchSong("youtube", "striptease", "carwash")
		// ReadPlaylist("youtube", "PLe0T9j3Sn3hnvtiIxpfx2a0SIy84VSSl7")
		// FindTrackIDFromFile("youtube", storageDir+"/youtube/PLe0T9j3Sn3hlWTqNWn6pP5jnRjVthBDBp.json")
		
	}

}

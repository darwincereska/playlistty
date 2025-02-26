# Playlistty

Playlistty is a command-line tool for transferring playlists between music streaming services. Currently supports Spotify and YouTube.

## Features

- Transfer playlists between Spotify and YouTube
- OAuth2 authentication for secure access
- List all playlists from connected accounts
- Create new playlists
- Clear existing playlists
- Search for songs across platforms

## Installation

```bash
git clone https://github.com/darwincereska/playlistty.git
cd playlistty
go build
```

## Setup

1. Create a `~/.config/playlistty` directory and a `config.yml` file inside it
2. Add your API credentials to `config.yml`:

See [Spotify setup guide](/setup/SPOTIFY.md) for details on obtaining Spotify API credentials.

```yaml
spotify:
  user_id: "your_spotify_user_id"
  client_id: "your_spotify_client_id"
  client_secret: "your_spotify_client_secret"
  token: ""

youtube:
  client_id: "your_youtube_client_id"
  client_secret: "your_youtube_client_secret"
  token: ""
```

3. Generate OAuth tokens:
```bash
playlistty -oauth spotify
playlistty -oauth yt
```

## Usage

```bash
# Transfer from Spotify
playlistty -service spotify

# Transfer from YouTube
playlistty -service yt

# Show help
playlistty -help
```

## How It Works

1. Choose source service (Spotify/YouTube)
2. Select source playlist
3. Choose destination service
4. Create new playlist or select existing one
5. Wait for transfer to complete

## Configuration

The tool stores its configuration in `config/config.yml`. OAuth tokens are automatically refreshed when needed.

## Requirements

- Go 1.x
- Spotify Developer Account (for API credentials)
- Google Developer Account (for YouTube API credentials)

## Dependencies

- golang.org/x/oauth2
- gopkg.in/yaml.v3

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License

[MIT](https://choosealicense.com/licenses/mit/)
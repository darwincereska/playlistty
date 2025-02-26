  
  
  # Setting up Spotify API
  
    
  
  **NOTE:** For now **Spotify Premium** is required to make the Spotify Developers App, but if you know someone who will make one for you that will work
  # Step 1:
  ### Creating the app
  1. Goto [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
  ![Spotify Developer Dashboard](images/spotify/Spotify-Developer-Dashboard.png)
  2. Create an app
  3. Name it whatever you want, but add in the redirect uri: **http://localhost:3000/callback**
  4. Make Sure to tick the Web API option on the bottom
  5. Save, then click options in the top right corner
  6. Copy the client-id and client-secret into `~/.config/playlistty/config.yml`
  ```yml
  spotify:
    client_id: insert-client-id 
    client_secret: insert-client-secret
    token: leave-empty
  ```
  # Step 2:
  ### Getting User ID
  1. Goto [Spotify](https://open.spotify.com/)
  2. Click your profile picture and select **Profile**
  3. Look at the url bar and copy the string of characters: https://open.spotify.com/user/user-id-here into `user_id` in `~/.config/playlistty/config.yml`
  ``` yml
  spotify:
    user_id: insert-user-id
    client_id: existing-client-id
    client_secret: existing-client-secret
    token: leave-empty
  ```
  4. Now you can either run `playlistty` as you normally would, or:
  ``` bash
  playlistty --oauth spotify
  ```
  5. Click the link and authorize **Playlistty**

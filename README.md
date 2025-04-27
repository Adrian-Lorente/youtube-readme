# Instructions

You only need one command to run the server:

```bash
go run ./cmd/api/main.go
```


Now, to see the output, visit `localhost:{port}:/draw/?video_id={youtube-video-id}`
Port is by default `8080`, but you can set it yourself by using an env variable named PORT.
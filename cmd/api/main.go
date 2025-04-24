package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

const YOUTUBE_THUMBNAIL_URL string = "https://img.youtube.com/vi/%s/maxresdefault.jpg"

func ExtractVideoIdFromUrl(w http.ResponseWriter, r *http.Request) string {
	video_id := r.URL.Query().Get("video_id")
	if video_id == "" {
		http.Error(w, "Missing parameter video_id", http.StatusBadRequest)
	}
	return video_id
}

func RequestThumbnail(video_id string) []byte {
	url := fmt.Sprintf(YOUTUBE_THUMBNAIL_URL, video_id)
	resp, err := http.Get(url)
	if err != nil {
		log.Print(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Print(err)
	}

	return body
}

func ProvideThumbnail(w http.ResponseWriter, r *http.Request) {
	video_id := ExtractVideoIdFromUrl(w, r)
	thumbnail := RequestThumbnail(video_id)
	w.Write(thumbnail)
}

func DrawPageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ProvideThumbnail(w, r)
	default:
		http.Error(w, "Only GET method is allowed!", http.StatusMethodNotAllowed)
	}
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Homepage Endpoint Hit")
}

func main() {
	http.HandleFunc("/", HomePage)
	http.HandleFunc("/draw/", DrawPageHandler)

	log.Println("Server running at localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

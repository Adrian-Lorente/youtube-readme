package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"

	"github.com/fogleman/gg"

	"github.com/PuerkitoBio/goquery"
)

const YOUTUBE_THUMBNAIL_URL string = "https://img.youtube.com/vi/%s/maxresdefault.jpg"
const YOUTUBE_VIDEO_URL string = "https://www.youtube.com/watch?v=%s"
const ASSETS_PATH string = "assets/"
const DEFAULT_FONT_PATH string = ASSETS_PATH + "NotoSansJP-VariableFont_wght.ttf"
const DEFAULT_TEMPLATE_PATH string = ASSETS_PATH + "vinyl_template.png"

var TEMPLATE_IMAGE image.Image = LoadTemplate()

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Video id must be provided in the request to the API. This function extracts said id.
func ExtractVideoIdFromUrl(w http.ResponseWriter, r *http.Request) string {
	video_id := r.URL.Query().Get("video_id")
	if video_id == "" {
		http.Error(w, "Missing parameter video_id", http.StatusBadRequest)
	}
	return video_id
}

// Returns the video's thumbnail.
func RequestThumbnail(video_id string) []byte {
	url := fmt.Sprintf(YOUTUBE_THUMBNAIL_URL, video_id)
	resp, err := http.Get(url)
	checkError(err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	checkError(err)

	return body
}

// Returns the video's title.
func RequestTitle(video_id string) string {
	url := fmt.Sprintf(YOUTUBE_VIDEO_URL, video_id)

	resp, err := http.Get(url)
	checkError(err)
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	checkError(err)

	title, exists := doc.Find("meta[name*='title']").Attr("content")

	if !exists {
		log.Fatal(err)
	}

	return title
}

func LoadTemplate() image.Image {
	img, err := gg.LoadImage(DEFAULT_TEMPLATE_PATH)
	checkError(err)
	return img
}

// Draw the text onto the image at fixed coordinates
func DrawText(canvas gg.Context, video_title string) gg.Context {
	canvas.SetColor(color.White)
	canvas.DrawStringWrapped(video_title, 59, 388, 0, 0, 350, 1.5, gg.AlignCenter)
	return canvas
}

// func DrawThumbnail(mutable_context gg.Context, thumbnail []byte){
// 	panic()
// }

// 1. Requests video title & thumbnail.
// 2. Loads the template.
// 3. Adjusts title & thumbnail to fit the template.
// 4. Responds with the image.
func ProvideThumbnail(w http.ResponseWriter, r *http.Request) {
	video_id := ExtractVideoIdFromUrl(w, r)
	// thumbnail := RequestThumbnail(video_id)
	title := RequestTitle(video_id)
	log.Printf("Video title: %s", title)

	// Creates new context (canvas)
	canvas := gg.NewContext(TEMPLATE_IMAGE.Bounds().Dx(), TEMPLATE_IMAGE.Bounds().Dy())
	canvas.DrawImage(TEMPLATE_IMAGE, 0, 0)
	canvas.LoadFontFace(DEFAULT_FONT_PATH, 25)

	// Draws canvas
	DrawText(*canvas, title)
	// DrawThumbnail(*canvas, thumbnail)

	image_buffer := new(bytes.Buffer)
	png.Encode(image_buffer, canvas.Image())
	w.Write(image_buffer.Bytes())
}

// Handler for "{endpoint}/draw".
func DrawPageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ProvideThumbnail(w, r)
	default:
		http.Error(w, "Only GET method is allowed!", http.StatusMethodNotAllowed)
	}
}

// Handler for "{endpoint}/".
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Homepage Endpoint Hit")
}

// Adds the handler and launches the server.
func main() {
	http.HandleFunc("/", HomePageHandler)
	http.HandleFunc("/draw/", DrawPageHandler)

	log.Println("Server running at localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

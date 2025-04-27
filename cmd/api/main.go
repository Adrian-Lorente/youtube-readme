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
	"os"

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
	log.Printf("Thumbnail url: %s", url)

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

func DrawThumbnail(canvas gg.Context, raw_thumbnail []byte) gg.Context {
	// Convert []byte to image
	thumbnail, _, err := image.Decode(bytes.NewReader(raw_thumbnail))
	checkError(err)

	// Define variables
	original_height := float64(thumbnail.Bounds().Dy())
	original_width := float64(thumbnail.Bounds().Dx())
	new_height := float64(300)
	scale := new_height / original_height
	new_width := original_width * scale

	radius := float64(150)

	// Scale image to a height of 300 keeping aspect ration
	scaled_canvas := gg.NewContext(int(new_width), int(new_height))
	scaled_canvas.DrawImageAnchored(thumbnail, 0, 0, 0, 0)
	scaled_canvas.Scale(scale, scale)
	scaled_canvas.DrawImage(thumbnail, 0, 0)

	// Crop to circle
	circle_canvas := gg.NewContext(int(radius*2), int(radius*2))
	circle_canvas.DrawCircle(radius, radius, radius)
	circle_canvas.Clip()
	circle_center_x := float64(new_width) / 2
	circle_center_y := float64(new_height) / 2
	circle_canvas.DrawImage(scaled_canvas.Image(), int(-circle_center_x+radius), int(-circle_center_y+radius))

	// Draw outer border
	circle_canvas.SetRGB255(0, 0, 0)
	circle_canvas.SetLineWidth(10)
	circle_canvas.DrawCircle(radius, radius, radius)
	circle_canvas.Stroke()

	// Draw inner circle
	circle_canvas.SetRGB255(43, 49, 55) // Background color hex #2B3137
	circle_canvas.DrawCircle(radius, radius, radius*0.2)
	circle_canvas.Fill()

	// Draw inner border
	circle_canvas.SetRGB255(0, 0, 0)
	circle_canvas.SetLineWidth(radius * 0.325)
	circle_canvas.DrawCircle(radius, radius, radius*0.2)
	circle_canvas.Stroke()

	// Draw transformed image into the general canvas
	canvas.DrawImage(circle_canvas.Image(), 75, 40)
	return canvas
}

// 1. Requests video title & thumbnail.
// 2. Loads the template.
// 3. Adjusts title & thumbnail to fit the template.
// 4. Responds with the image.
func ProvideThumbnail(w http.ResponseWriter, r *http.Request) {
	video_id := ExtractVideoIdFromUrl(w, r)
	thumbnail := RequestThumbnail(video_id)
	title := RequestTitle(video_id)
	log.Printf("Video title: %s", title)

	// Creates new context (canvas)
	canvas := gg.NewContext(TEMPLATE_IMAGE.Bounds().Dx(), TEMPLATE_IMAGE.Bounds().Dy())
	canvas.DrawImage(TEMPLATE_IMAGE, 0, 0)
	canvas.LoadFontFace(DEFAULT_FONT_PATH, 25)

	// Draws canvas
	DrawText(*canvas, title)
	DrawThumbnail(*canvas, thumbnail)

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
	w.Write([]byte("Welcome! Use the avaibale endpoint /draw/?video_id={your-video-id}"))
}

// Middleware for deploying
func CORSMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		next.ServeHTTP(w, r)
	})
}

// Adds the handler and launches the server.
func main() {
	http.Handle("/", CORSMiddleware(http.HandlerFunc(HomePageHandler)))
	http.Handle("/draw/", CORSMiddleware(http.HandlerFunc(DrawPageHandler)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

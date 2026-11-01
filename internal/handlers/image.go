package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

var (
	imageEndpoint    string
	imageKey         []byte
	imageSalt        []byte
	imageAllowedHost map[string]struct{}
)

// InitImageConfig validates and stores the imgproxy configuration.
// Call it once during server startup, after loading environment variables.
func InitImageConfig() error {
	endpoint := strings.TrimRight(os.Getenv("IMGPROXY_URL"), "/")
	if endpoint == "" {
		return fmt.Errorf("IMGPROXY_URL is required")
	}

	keyHex := os.Getenv("IMGPROXY_KEY")
	saltHex := os.Getenv("IMGPROXY_SALT")
	if keyHex == "" || saltHex == "" {
		return fmt.Errorf("IMGPROXY_KEY and IMGPROXY_SALT are required")
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("decode IMGPROXY_KEY: %w", err)
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return fmt.Errorf("decode IMGPROXY_SALT: %w", err)
	}

	hosts := os.Getenv("IMGPROXY_ALLOWED_HOSTS")
	if hosts == "" {
		hosts = "cdn.sanity.io,blue-nomad.nyc3.cdn.digitaloceanspaces.com"
	}

	allowedHost := make(map[string]struct{})
	for _, host := range strings.Split(hosts, ",") {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			allowedHost[host] = struct{}{}
		}
	}
	if len(allowedHost) == 0 {
		return fmt.Errorf("IMGPROXY_ALLOWED_HOSTS must contain at least one host")
	}

	imageEndpoint = endpoint
	imageKey = key
	imageSalt = salt
	imageAllowedHost = allowedHost
	return nil
}

// RedirectImage sends the browser to a signed imgproxy URL.
func RedirectImage(w http.ResponseWriter, r *http.Request) {
	width, err := strconv.Atoi(mux.Vars(r)["width"])
	if err != nil || !allowedImageWidth(width) {
		http.Error(w, "Unsupported image width", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "webp"
	}
	if !allowedImageFormat(format) {
		http.Error(w, "Unsupported image format", http.StatusBadRequest)
		return
	}

	quality := 85
	if rawQuality := r.URL.Query().Get("quality"); rawQuality != "" {
		quality, err = strconv.Atoi(rawQuality)
		if err != nil || quality < 1 || quality > 100 {
			http.Error(w, "Quality must be between 1 and 100", http.StatusBadRequest)
			return
		}
	}

	source := r.URL.Query().Get("src")
	if source == "" || len(source) > 4096 {
		http.Error(w, "Invalid image URL", http.StatusBadRequest)
		return
	}

	imageURL, err := url.Parse(source)
	if err != nil ||
		imageURL.Scheme != "https" ||
		imageURL.Host == "" ||
		imageURL.User != nil {
		http.Error(w, "Invalid image URL", http.StatusBadRequest)
		return
	}

	if _, ok := imageAllowedHost[strings.ToLower(imageURL.Hostname())]; !ok {
		http.Error(w, "Image host is not allowed", http.StatusBadRequest)
		return
	}

	imageURL.Fragment = ""
	imageURL.RawFragment = ""

	signedURL := signedImageURL(imageURL.String(), width, format, quality)

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.Redirect(w, r, signedURL, http.StatusFound)
}

func signedImageURL(source string, width int, format string, quality int) string {
	encodedSource := base64.RawURLEncoding.EncodeToString([]byte(source))

	path := fmt.Sprintf(
		"/rs:fill:%d:0/q:%d/f:%s/%s.%s",
		width,
		quality,
		format,
		encodedSource,
		format,
	)

	mac := hmac.New(sha256.New, imageKey)
	_, _ = mac.Write(imageSalt)
	_, _ = mac.Write([]byte(path))

	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return imageEndpoint + "/" + signature + path
}

func allowedImageFormat(format string) bool {
	switch format {
	case "webp", "avif":
		return true
	default:
		return false
	}
}

func allowedImageWidth(width int) bool {
	return width >= 1 && width <= 2400
}

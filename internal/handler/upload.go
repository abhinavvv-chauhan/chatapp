package handler

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

func GetCloudinarySignature(w http.ResponseWriter, r *http.Request) {
	// 1. Grab your secret URL from Render
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	if cloudinaryURL == "" {
		http.Error(w, "Cloudinary config missing in production", http.StatusInternalServerError)
		return
	}

	// 2. Parse the URL (cloudinary://API_KEY:API_SECRET@CLOUD_NAME)
	u, err := url.Parse(cloudinaryURL)
	if err != nil {
		http.Error(w, "Invalid Cloudinary URL formatting", http.StatusInternalServerError)
		return
	}

	apiKey := u.User.Username()
	apiSecret, _ := u.User.Password()
	cloudName := u.Host

	// 3. Create the Signature payload
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	
	// Cloudinary requires signing a string of parameters + your API Secret
	strToSign := fmt.Sprintf("timestamp=%s%s", timestamp, apiSecret)

	// Hash it using SHA-1
	h := sha1.New()
	h.Write([]byte(strToSign))
	signature := hex.EncodeToString(h.Sum(nil))

	// 4. Send the credentials back to the frontend!
	response := map[string]string{
		"signature":  signature,
		"timestamp":  timestamp,
		"api_key":    apiKey,
		"cloud_name": cloudName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func UploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	os.MkdirAll("./uploads", os.ModePerm)

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))
	savePath := filepath.Join(".", "uploads", filename)

	dst, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "Error saving file to server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url":  "/uploads/" + filename,
		"name": handler.Filename,
	})
}

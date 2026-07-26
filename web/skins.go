package web

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (api *Api) GetSkin(resp http.ResponseWriter, req *http.Request) {
	filename := strings.TrimPrefix(req.URL.Path, "/api/skins/")
	// A skin is one file directly below SkinsPath. Keep these checks here so
	// static analysis can verify that user input is constrained before ReadFile.
	if filename == "" ||
		strings.Contains(filename, "/") ||
		strings.Contains(filename, `\`) ||
		strings.Contains(filename, "..") ||
		!strings.HasSuffix(strings.ToLower(filename), ".png") {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	filePath := filepath.Join(api.Context.Config.Skins.SkinsPath, filename)

	content, err := os.ReadFile(filePath)
	// make file not found more sensible
	if errors.Is(err, os.ErrNotExist) {
		resp.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// return the file content when available
	if content != nil {
		resp.Header().Set("Content-Type", "image/png")
		_, _ = resp.Write(content)
		return
	}

	// fallback
	resp.WriteHeader(http.StatusNotFound)
	_, _ = resp.Write([]byte(filename))
}

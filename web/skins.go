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
	if !validSkinFilename(filename) {
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

func validSkinFilename(filename string) bool {
	// A skin is one file directly below SkinsPath. Check both separator styles so
	// the validation remains safe when mapserver is built for another platform.
	return filename != "" &&
		!strings.ContainsAny(filename, `/\`) &&
		filename != "." && filename != ".." &&
		strings.HasSuffix(strings.ToLower(filename), ".png")
}

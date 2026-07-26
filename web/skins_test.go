package web

import (
	"mapserver/app"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetSkinRejectsUnsafeFilenames(t *testing.T) {
	skinsPath := t.TempDir()
	api := &Api{Context: &app.App{Config: &app.Config{
		Skins: &app.SkinsConfig{SkinsPath: skinsPath},
	}}}

	tests := []string{
		"/api/skins/",
		"/api/skins/character.jpg",
		"/api/skins/../secret.png",
		`/api/skins/..\secret.png`,
		"/api/skins/subdir/character.png",
		`/api/skins/C:\tmp\character.png`,
	}

	for _, requestPath := range tests {
		t.Run(requestPath, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, requestPath, nil)
			response := httptest.NewRecorder()
			api.GetSkin(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("GetSkin(%q) returned %d, want %d", requestPath, response.Code, http.StatusNotFound)
			}
		})
	}
}

func TestGetSkinServesPNG(t *testing.T) {
	skinsPath := t.TempDir()
	const filename = "spelare-åäö.png"
	const content = "png content"
	if err := os.WriteFile(filepath.Join(skinsPath, filename), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	api := &Api{Context: &app.App{Config: &app.Config{
		Skins: &app.SkinsConfig{SkinsPath: skinsPath},
	}}}
	request := httptest.NewRequest(http.MethodGet, "/api/skins/"+filename, nil)
	response := httptest.NewRecorder()
	api.GetSkin(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GetSkin returned %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != content {
		t.Fatalf("GetSkin returned %q, want %q", response.Body.String(), content)
	}
	if got := response.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("GetSkin Content-Type = %q, want image/png", got)
	}
}

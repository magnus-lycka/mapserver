package web

import "testing"

func TestValidSkinFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "png", filename: "character.png", want: true},
		{name: "uppercase extension", filename: "character.PNG", want: true},
		{name: "unicode", filename: "spelare-åäö.png", want: true},
		{name: "empty", filename: "", want: false},
		{name: "wrong extension", filename: "character.jpg", want: false},
		{name: "unix traversal", filename: "../secret.png", want: false},
		{name: "windows traversal", filename: `..\secret.png`, want: false},
		{name: "nested path", filename: "subdir/character.png", want: false},
		{name: "absolute unix path", filename: "/tmp/character.png", want: false},
		{name: "absolute windows path", filename: `C:\tmp\character.png`, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validSkinFilename(test.filename); got != test.want {
				t.Fatalf("validSkinFilename(%q) = %t, want %t", test.filename, got, test.want)
			}
		})
	}
}

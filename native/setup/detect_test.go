package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverUnpackedExtension(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "Default")
	extension := filepath.Join(root, "source", "extension")
	if err := os.MkdirAll(profile, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extension, 0755); err != nil {
		t.Fatal(err)
	}
	id := "abcdefghijklmnopabcdefghijklmnop"
	prefs := map[string]any{"extensions": map[string]any{"settings": map[string]any{id: map[string]any{"path": extension}}}}
	b, _ := json.Marshal(prefs)
	if err := os.WriteFile(filepath.Join(profile, "Secure Preferences"), b, 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"WT Live Skin Installer","permissions":["nativeMessaging"]}`
	if err := os.WriteFile(filepath.Join(extension, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	ids := discoverExtensions(root)
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("got %v", ids)
	}
}

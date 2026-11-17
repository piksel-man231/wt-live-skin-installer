package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var extensionID = regexp.MustCompile(`^[a-p]{32}$`)

type extensionPrefs struct {
	Extensions struct {
		Settings map[string]struct {
			Path string `json:"path"`
		} `json:"settings"`
	} `json:"extensions"`
}

type extensionManifest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func discoverExtensions(userDataRoots ...string) []string {
	seen := map[string]bool{}
	for _, root := range userDataRoots {
		profiles, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, profile := range profiles {
			if !profile.IsDir() {
				continue
			}
			base := filepath.Join(root, profile.Name())
			for _, prefName := range []string{"Secure Preferences", "Preferences"} {
				data, err := os.ReadFile(filepath.Join(base, prefName))
				if err != nil {
					continue
				}
				var prefs extensionPrefs
				if json.Unmarshal(data, &prefs) != nil {
					continue
				}
				for id, setting := range prefs.Extensions.Settings {
					if !extensionID.MatchString(id) || setting.Path == "" {
						continue
					}
					candidate := setting.Path
					if !filepath.IsAbs(candidate) {
						candidate = filepath.Join(base, "Extensions", candidate)
					}
					data, err := os.ReadFile(filepath.Join(candidate, "manifest.json"))
					if err != nil {
						continue
					}
					var manifest extensionManifest
					if json.Unmarshal(data, &manifest) != nil || manifest.Name != "WT Live Skin Installer" {
						continue
					}
					for _, p := range manifest.Permissions {
						if strings.EqualFold(p, "nativeMessaging") {
							seen[id] = true
							break
						}
					}
				}
			}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

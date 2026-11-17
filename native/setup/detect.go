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
	Name  string   `json:"name"`
	Hosts []string `json:"host_permissions"`
}

type extensionInstall struct {
	ID   string
	Path string
}

func discoverExtensionInstalls(userDataRoots ...string) []extensionInstall {
	seen := map[string]extensionInstall{}
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
					if !extensionID.MatchString(id) || !filepath.IsAbs(setting.Path) {
						continue
					}
					data, err := os.ReadFile(filepath.Join(setting.Path, "manifest.json"))
					if err != nil {
						continue
					}
					var manifest extensionManifest
					if json.Unmarshal(data, &manifest) != nil || manifest.Name != "WT Live Skin Installer" {
						continue
					}
					for _, host := range manifest.Hosts {
						if strings.EqualFold(host, "https://live.warthunder.com/*") {
							seen[id] = extensionInstall{ID: id, Path: setting.Path}
							break
						}
					}
				}
			}
		}
	}
	installs := make([]extensionInstall, 0, len(seen))
	for _, install := range seen {
		installs = append(installs, install)
	}
	return installs
}

func discoverExtensions(userDataRoots ...string) []string {
	installs := discoverExtensionInstalls(userDataRoots...)
	ids := make([]string, len(installs))
	for i, install := range installs {
		ids[i] = install.ID
	}
	return ids
}

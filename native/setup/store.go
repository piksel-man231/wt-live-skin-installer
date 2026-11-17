package main

import "errors"

// Set with -X main.storeExtensionID=<Chrome Web Store item ID> for store builds.
var storeExtensionID string

func storeListingURL(id string) (string, error) {
	if id == "" {
		return "", nil
	}
	if !extensionID.MatchString(id) {
		return "", errors.New("Установщик собран с некорректным ID Chrome Web Store.")
	}
	return "https://chromewebstore.google.com/detail/" + id, nil
}

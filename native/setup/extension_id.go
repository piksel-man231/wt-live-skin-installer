package main

import (
	"crypto/sha256"
	"encoding/binary"
	"unicode/utf16"
)

// Chromium hashes the absolute Windows path as UTF-16LE for unpacked extensions.
func unpackedWindowsID(path string) string {
	if len(path) >= 2 && path[1] == ':' && path[0] >= 'a' && path[0] <= 'z' {
		path = string(path[0]-'a'+'A') + path[1:]
	}
	units := utf16.Encode([]rune(path))
	data := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[i*2:], unit)
	}
	hash := sha256.Sum256(data)
	var id [32]byte
	for i, value := range hash[:16] {
		id[i*2] = 'a' + value>>4
		id[i*2+1] = 'a' + value&15
	}
	return string(id[:])
}

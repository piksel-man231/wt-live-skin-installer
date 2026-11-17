package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed extension.zip
var extensionArchive []byte

func updateExtension(destination string) error {
	zr, err := zip.NewReader(bytes.NewReader(extensionArchive), int64(len(extensionArchive)))
	if err != nil {
		return err
	}
	for _, entry := range zr.File {
		if !strings.HasPrefix(entry.Name, "extension/") {
			return errors.New("Некорректный файл расширения в установщике.")
		}
		rel := strings.TrimPrefix(entry.Name, "extension/")
		if rel == "" {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(rel))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return errors.New("Некорректный путь в установщике.")
		}
		target := filepath.Join(destination, clean)
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if entry.UncompressedSize64 > 1<<20 {
			return errors.New("Слишком большой файл расширения в установщике.")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(in, 1<<20))
		in.Close()
		if err != nil {
			return err
		}
		if uint64(len(data)) != entry.UncompressedSize64 {
			return errors.New("Повреждён файл расширения в установщике.")
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

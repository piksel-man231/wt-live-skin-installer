//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unicode/utf16"
)

func showFirstRun(folder string) {
	pageOpened := openExtensionsPage()
	guide := "В браузере откройте vivaldi://extensions или chrome://extensions."
	if pageOpened {
		guide = "Установщик открыл страницу расширений браузера."
	}
	pathHint := "Вставьте этот путь в окно выбора папки:"
	if copyFolderPath(folder) {
		pathHint = "В окне выбора папки нажмите Alt+D, вставьте путь (Ctrl+V), нажмите Enter и выберите папку."
	}
	messageBox("WT Live Skin Installer", "Помощник установлен.\n\n"+guide+
		" Включите «Режим разработчика» и нажмите «Загрузить распакованное расширение».\n\n"+
		pathHint+"\n"+folder+"\n\nЗатем обновите вкладку WT Live.", 0x40)
}

func openExtensionsPage() bool {
	browsers := []struct {
		executable string
		folder     string
		page       string
	}{
		{"vivaldi.exe", "Vivaldi", "vivaldi://extensions/"},
		{"chrome.exe", filepath.Join("Google", "Chrome"), "chrome://extensions/"},
	}
	for _, browser := range browsers {
		for _, variable := range []string{"LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)"} {
			root := os.Getenv(variable)
			if root == "" {
				continue
			}
			executable := filepath.Join(root, browser.folder, "Application", browser.executable)
			if info, err := os.Stat(executable); err != nil || info.IsDir() {
				continue
			}
			if err := exec.Command(executable, browser.page).Start(); err == nil {
				return true
			}
		}
	}
	return false
}

func copyFolderPath(folder string) bool {
	// clip.exe accepts UTF-16 text with a BOM, including Windows user names.
	units := utf16.Encode([]rune(folder))
	data := make([]byte, 2+2*len(units))
	binary.LittleEndian.PutUint16(data, 0xfeff)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2+i*2:], unit)
	}
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "clip.exe"))
	cmd.Stdin = bytes.NewReader(data)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

//go:build windows

package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"unsafe"
)

//go:embed wt-live-host.exe
var hostBinary []byte

func main() {
	if err := install(); err != nil {
		messageBox("WT Live Skin Installer", "Не удалось установить помощник:\n\n"+err.Error(), 0x10)
		os.Exit(1)
	}
	messageBox("WT Live Skin Installer", "Расширение обновлено и помощник установлен. Полностью закройте и снова откройте браузер. Теперь кнопка на WT Live будет устанавливать скин сразу, без отдельной вкладки.", 0x40)
}

func messageBox(title, body string, style uintptr) {
	dll := syscall.NewLazyDLL("user32.dll")
	fn := dll.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(body)
	fn.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), style)
}

func install() error {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return errors.New("Не найдена папка AppData текущего пользователя.")
	}
	gameRoot := filepath.Join(local, "WarThunder")
	info, err := os.Stat(gameRoot)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("Игра не найдена в %s. Папку можно указать через расширенную установку в README.", gameRoot)
	}
	userSkins := filepath.Join(gameRoot, "UserSkins")
	if err := os.MkdirAll(userSkins, 0755); err != nil {
		return err
	}
	installs := discoverExtensionInstalls(
		filepath.Join(local, "Vivaldi", "User Data"),
		filepath.Join(local, "Google", "Chrome", "User Data"),
	)
	if len(installs) == 0 {
		return errors.New("Не найдено расширение WT Live Skin Installer в Vivaldi или Chrome. Установите расширение из репозитория и запустите установщик снова.")
	}
	ids := make([]string, 0, len(installs))
	for _, extension := range installs {
		if err := updateExtension(extension.Path); err != nil {
			return fmt.Errorf("Не удалось обновить расширение: %w", err)
		}
		ids = append(ids, extension.ID)
	}
	target := filepath.Join(local, "WtLiveSkinInstaller")
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	hostPath := filepath.Join(target, "wt-live-host.exe")
	if err := os.WriteFile(hostPath, hostBinary, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(target, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var old struct {
			IDs []string `json:"extension_ids"`
		}
		if json.Unmarshal(data, &old) == nil {
			seen := map[string]bool{}
			for _, id := range ids {
				seen[id] = true
			}
			for _, id := range old.IDs {
				if extensionID.MatchString(id) {
					seen[id] = true
				}
			}
			ids = ids[:0]
			for id := range seen {
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	config := struct {
		IDs       []string `json:"extension_ids"`
		UserSkins string   `json:"user_skins"`
	}{ids, userSkins}
	if err := writeJSON(configPath, config); err != nil {
		return err
	}
	origins := make([]string, len(ids))
	for i, id := range ids {
		origins[i] = "chrome-extension://" + id + "/"
	}
	manifestPath := filepath.Join(target, "com.wtlive.skin_installer.json")
	manifest := struct {
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		Path           string   `json:"path"`
		Type           string   `json:"type"`
		AllowedOrigins []string `json:"allowed_origins"`
	}{"com.wtlive.skin_installer", "WT Live Skin Installer helper", hostPath, "stdio", origins}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return err
	}
	for _, browser := range []string{"Google\\Chrome", "Vivaldi"} {
		key := `HKCU\Software\` + browser + `\NativeMessagingHosts\com.wtlive.skin_installer`
		output, err := exec.Command("reg.exe", "add", key, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f").CombinedOutput()
		if err != nil {
			return fmt.Errorf("Не удалось зарегистрировать помощник для %s: %s", browser, output)
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

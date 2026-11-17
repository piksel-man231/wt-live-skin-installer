package main

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxArchive = 250 << 20
const maxUnpacked = 1 << 30
const hostName = "com.wtlive.skin_installer"

var postID = regexp.MustCompile(`^[0-9]{1,12}$`)
var allowedFile = regexp.MustCompile(`(?i)\.(blk|dds|tga|png|jpg|jpeg|bmp|webp|tif|tiff|ktx|txt)$`)
var safeLabel = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
var reservedName = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)`)

type config struct {
	ExtensionIDs []string `json:"extension_ids"`
	UserSkins    string   `json:"user_skins"`
}
type request struct {
	Post string `json:"post"`
}
type reply struct {
	OK    bool   `json:"ok"`
	Count int    `json:"count,omitempty"`
	Error string `json:"error,omitempty"`
}
type livePost struct {
	Type string `json:"type"`
	File struct {
		Link string `json:"link"`
		Size int64  `json:"size"`
	} `json:"file"`
}
type entry struct {
	Parts []string
	File  *zip.File
}
type group struct {
	Label string
	Files []entry
}

func main() {
	var answer reply
	defer func() { _ = send(answer) }()
	if err := run(&answer); err != nil {
		answer = reply{Error: err.Error()}
	}
}

func run(answer *reply) error {
	var length uint32
	if err := binary.Read(os.Stdin, binary.LittleEndian, &length); err != nil {
		return err
	}
	if length == 0 || length > 1024 {
		return errors.New("Некорректный запрос расширения.")
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(os.Stdin, data); err != nil {
		return err
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil || !postID.MatchString(req.Post) {
		return errors.New("Некорректный номер публикации.")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	settings, err := os.ReadFile(filepath.Join(filepath.Dir(self), "config.json"))
	if err != nil {
		return errors.New("Не найдена настройка помощника. Запустите install.ps1.")
	}
	var cfg config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		return err
	}
	allowed := false
	if len(os.Args) >= 2 {
		origin := strings.TrimSuffix(os.Args[1], "/")
		for _, id := range cfg.ExtensionIDs {
			if origin == "chrome-extension://"+id {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		return errors.New("Расширение не разрешено для этого помощника.")
	}
	if !strings.EqualFold(filepath.Base(filepath.Clean(cfg.UserSkins)), "UserSkins") {
		return errors.New("В настройках нужна папка UserSkins.")
	}
	if info, err := os.Stat(cfg.UserSkins); err != nil || !info.IsDir() {
		return errors.New("Папка UserSkins не найдена.")
	}
	archive, err := download(req.Post)
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(archive, info.Size())
	if err != nil {
		return errors.New("WT Live вернул не ZIP-архив.")
	}
	groups, err := plan(zr.File)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if err := installGroup(cfg.UserSkins, req.Post, g); err != nil {
			return err
		}
	}
	*answer = reply{OK: true, Count: len(groups)}
	return nil
}

func send(value reply) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := binary.Write(os.Stdout, binary.LittleEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func allowedHost(u *url.URL) bool {
	return u.Scheme == "https" && (u.Hostname() == "live.warthunder.com" || u.Hostname() == "cdn-live.warthunder.com")
}

func download(post string) (*os.File, error) {
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 || !allowedHost(r.URL) {
			return errors.New("Недопустимое перенаправление WT Live.")
		}
		return nil
	}}
	form := url.Values{"lang_group": {post}, "language": {"en"}}
	resp, err := client.PostForm("https://live.warthunder.com/api/posts/get/", form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("WT Live не ответил на запрос публикации.")
	}
	var item livePost
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&item); err != nil {
		return nil, err
	}
	if item.Type != "camouflage" || item.File.Link == "" {
		return nil, errors.New("Эта публикация не содержит камуфляж.")
	}
	u, err := url.Parse(item.File.Link)
	if err != nil || u.Scheme != "https" || u.Hostname() != "live.warthunder.com" || !strings.HasPrefix(u.Path, "/dl/") {
		return nil, errors.New("WT Live вернул неожиданную ссылку.")
	}
	if item.File.Size > maxArchive {
		return nil, errors.New("Архив больше 250 МБ.")
	}
	resp, err = client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Не удалось скачать архив.")
	}
	if resp.ContentLength > maxArchive {
		return nil, errors.New("Архив больше 250 МБ.")
	}
	tmp, err := os.CreateTemp("", "wt-live-*.zip")
	if err != nil {
		return nil, err
	}
	cleanup := func(e error) (*os.File, error) { tmp.Close(); os.Remove(tmp.Name()); return nil, e }
	n, err := io.Copy(tmp, io.LimitReader(resp.Body, maxArchive+1))
	if err != nil {
		return cleanup(err)
	}
	if n > maxArchive {
		return cleanup(errors.New("Архив больше 250 МБ."))
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return cleanup(err)
	}
	return tmp, nil
}

func safeParts(name string) ([]string, error) {
	name = strings.ReplaceAll(name, `\`, `/`)
	if strings.HasPrefix(name, "/") {
		return nil, errors.New("В архиве есть небезопасный путь.")
	}
	parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
	if len(parts) == 0 || len(parts) > 8 {
		return nil, errors.New("Некорректная структура архива.")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || len(part) > 180 || strings.ContainsAny(part, `<>:"|?*`) || strings.HasSuffix(part, " ") || strings.HasSuffix(part, ".") || reservedName.MatchString(part) {
			return nil, errors.New("В архиве есть небезопасный путь.")
		}
		for _, r := range part {
			if r < 32 {
				return nil, errors.New("В архиве есть небезопасный путь.")
			}
		}
	}
	return parts, nil
}

func plan(files []*zip.File) ([]group, error) {
	entries := make([]entry, 0, len(files))
	var total uint64
	for _, f := range files {
		parts, err := safeParts(f.Name)
		if err != nil {
			return nil, err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("Архив содержит ссылку на файл.")
		}
		if !allowedFile.MatchString(parts[len(parts)-1]) {
			continue
		}
		if f.UncompressedSize64 > 512<<20 {
			return nil, errors.New("Слишком большой файл в архиве.")
		}
		total += f.UncompressedSize64
		if total > maxUnpacked || len(entries) >= 500 {
			return nil, errors.New("Архив слишком большой для установки.")
		}
		entries = append(entries, entry{parts, f})
	}
	if len(entries) == 0 {
		return nil, errors.New("В архиве нет файлов скина.")
	}
	for {
		common := entries[0].Parts[0]
		all := true
		for _, e := range entries {
			if len(e.Parts) < 2 || e.Parts[0] != common {
				all = false
				break
			}
		}
		if !all {
			break
		}
		for i := range entries {
			entries[i].Parts = entries[i].Parts[1:]
		}
	}
	var groups []group
	for _, cfg := range entries {
		if !strings.EqualFold(path.Ext(cfg.Parts[len(cfg.Parts)-1]), ".blk") {
			continue
		}
		parent := cfg.Parts[:len(cfg.Parts)-1]
		label := safeLabel.ReplaceAllString(strings.Join(append(append([]string{}, parent...), strings.TrimSuffix(cfg.Parts[len(cfg.Parts)-1], path.Ext(cfg.Parts[len(cfg.Parts)-1]))), "_"), "_")
		if len(label) > 90 {
			label = label[:90]
		}
		if label == "" {
			label = "skin"
		}
		g := group{Label: label, Files: []entry{{[]string{cfg.Parts[len(cfg.Parts)-1]}, cfg.File}}}
		for _, asset := range entries {
			if strings.EqualFold(path.Ext(asset.Parts[len(asset.Parts)-1]), ".blk") {
				continue
			}
			inTree := len(asset.Parts) > len(parent)
			for i, p := range parent {
				if !inTree || asset.Parts[i] != p {
					inTree = false
					break
				}
			}
			if inTree {
				g.Files = append(g.Files, entry{asset.Parts[len(parent):], asset.File})
			} else if len(parent) > 0 && len(asset.Parts) == 1 {
				g.Files = append(g.Files, asset)
			}
		}
		groups = append(groups, g)
	}
	if len(groups) == 0 {
		return nil, errors.New("В архиве нет файла .blk.")
	}
	return groups, nil
}

func installGroup(root, post string, g group) error {
	base := "wt_live_" + post + "_" + g.Label
	var dest string
	created := false
	for i := 0; i < 1000; i++ {
		name := base
		if i > 0 {
			name = fmt.Sprintf("%s_%d", base, i+1)
		}
		dest = filepath.Join(root, name)
		err := os.Mkdir(dest, 0755)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		created = true
		break
	}
	if !created {
		return errors.New("Не удалось создать папку скина.")
	}
	for _, file := range g.Files {
		p := filepath.Join(append([]string{dest}, file.Parts...)...)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			os.RemoveAll(dest)
			return err
		}
		in, err := file.File.Open()
		if err != nil {
			os.RemoveAll(dest)
			return err
		}
		out, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			in.Close()
			os.RemoveAll(dest)
			return err
		}
		n, err := io.Copy(out, io.LimitReader(in, 512<<20+1))
		closeErr := out.Close()
		in.Close()
		if err != nil || closeErr != nil || uint64(n) != file.File.UncompressedSize64 {
			os.RemoveAll(dest)
			return errors.New("Ошибка распаковки скина.")
		}
	}
	return nil
}

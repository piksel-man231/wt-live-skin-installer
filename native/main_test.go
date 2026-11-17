package main

import (
	"archive/zip"
	"bytes"
	"net/url"
	"strings"
	"testing"
)

func makeZIP(t *testing.T, names ...string) []*zip.File {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range names {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte("fixture")); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r.File
}

func TestMultipleVehicles(t *testing.T) {
	files := makeZIP(t,
		"pack/tank/tank.blk", "pack/tank/tank.dds",
		"pack/plane/plane.blk", "pack/plane/plane.tga")
	groups, err := plan(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(groups))
	}
	for _, g := range groups {
		if len(g.Files) != 2 {
			t.Errorf("%s has %d files", g.Label, len(g.Files))
		}
	}
}

func TestRejectTraversal(t *testing.T) {
	_, err := plan(makeZIP(t, "../escape.blk"))
	if err == nil || !strings.Contains(err.Error(), "небезопасный") {
		t.Fatalf("got %v", err)
	}
}

func TestRejectOtherHosts(t *testing.T) {
	for _, raw := range []string{"http://live.warthunder.com/dl/", "https://evil.example/dl/"} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if allowedHost(u) {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestFolderNames(t *testing.T) {
	if got := folderName("Maus, 'Alles Kaputt', 1945.zip", "germ_pzkpfw_Maus", 1); got != "Maus, 'Alles Kaputt', 1945" {
		t.Fatalf("single skin: %q", got)
	}
	if got := folderName("Blackout_10.3+10.7_Germany.zip", "Blackout_boxer_3105", 8); got != "Blackout 10.3+10.7 Germany - Blackout boxer 3105" {
		t.Fatalf("pack skin: %q", got)
	}
}

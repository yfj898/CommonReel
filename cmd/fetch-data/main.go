package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dir := flag.String("dir", "data/ml-1m", "local data directory")
	archive := flag.String("archive", "", "extract an already downloaded official MovieLens ZIP")
	flag.Parse()
	if err := fetch(*dir, *archive); err != nil {
		log.Fatal(err)
	}
}
func fetch(dir, archiveInput string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for _, name := range []string{"movies.dat", "ratings.dat"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return fmt.Errorf("%s already exists; refusing to overwrite", name)
		}
	}
	url := "https://files.grouplens.org/datasets/movielens/ml-1m.zip"
	const max = 20 << 20
	var bytes []byte
	var err error
	if archiveInput != "" {
		f, e := os.Open(archiveInput)
		if e != nil {
			return e
		}
		bytes, err = io.ReadAll(io.LimitReader(f, max+1))
		f.Close()
	} else {
		client := &http.Client{Timeout: 2 * time.Minute}
		response, e := client.Get(url)
		if e != nil {
			return e
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			return fmt.Errorf("download HTTP %d", response.StatusCode)
		}
		bytes, err = io.ReadAll(io.LimitReader(response.Body, max+1))
	}
	if err != nil {
		return err
	}
	if len(bytes) > max {
		return fmt.Errorf("archive exceeds 20 MiB")
	}
	archive := filepath.Join(dir, "ml-1m.zip")
	if err := os.WriteFile(archive, bytes, 0600); err != nil {
		return err
	}
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close()
	found := map[string]bool{}
	for _, f := range z.File {
		name := filepath.Base(f.Name)
		if f.Name != "ml-1m/"+name || (name != "movies.dat" && name != "ratings.dat" && name != "README") {
			continue
		}
		if f.UncompressedSize64 > 100<<20 {
			return fmt.Errorf("unexpected oversized %s", name)
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		data, e := io.ReadAll(io.LimitReader(r, (100<<20)+1))
		r.Close()
		if e != nil {
			return e
		}
		if len(data) > 100<<20 {
			return fmt.Errorf("oversized data")
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			return err
		}
		found[name] = true
	}
	if !found["movies.dat"] || !found["ratings.dat"] || !found["README"] {
		return fmt.Errorf("required archive files absent")
	}
	h := sha256.Sum256(bytes)
	manifest := map[string]any{"source": url, "local_archive_input": archiveInput, "extracted_utc": time.Now().UTC().Format(time.RFC3339), "sha256": hex.EncodeToString(h[:]), "bytes": len(bytes), "license": "MovieLens research-only terms; no redistribution/commercial use without permission", "excluded": "users.dat demographics not extracted"}
	encoded, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "DOWNLOAD.json"), encoded, 0600); err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

package main

import (
	"commonreel/internal/workflow"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

func loadEnv(path string) error {
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	for _, line := range strings.Split(strings.TrimPrefix(string(b), "\ufeff"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && os.Getenv(strings.TrimSpace(k)) == "" {
			os.Setenv(strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), "\"'"))
		}
	}
	return nil
}
func main() {
	listen := flag.String("listen", "127.0.0.1:8093", "explicit loopback/private IP:port; private IP enables same-Wi-Fi use")
	lan := flag.String("lan", "", "optional second listener on an explicit private interface IP, using the same port")
	db := flag.String("db", "data/commonreel-v2.db", "versioned database; old prototype remains untouched")
	catalog := flag.String("catalog", "assets/catalog-v2.json", "verified CC0 movie catalog JSON")
	manifest := flag.String("manifest", "assets/catalog-v2.manifest.json", "catalog manifest with expected SHA-256")
	cap := flag.Int("supplier-request-cap", 600, "maximum Watchmode HTTP requests in this process; never upgrades plans")
	flag.Parse()
	if e := loadEnv(".env.local"); e != nil {
		log.Fatal("could not read local configuration")
	}
	host, _, e := net.SplitHostPort(*listen)
	if e != nil {
		log.Fatal(e)
	}
	ip := net.ParseIP(host)
	if ip == nil || (!ip.IsLoopback() && !ip.IsPrivate()) {
		log.Fatal("choose an explicit loopback or private interface IP; wildcard/public binding is disabled")
	}
	if *cap < 1 || *cap > 2000 {
		log.Fatal("supplier-request-cap must be 1-2000")
	}
	var films []workflow.Film
	raw, e := os.ReadFile(*catalog)
	if e != nil {
		log.Fatal("catalog missing; run python scripts/build-catalog-entities.py first")
	}
	if e = json.Unmarshal(raw, &films); e != nil {
		log.Fatal("invalid catalog JSON")
	}
	var mf struct {
		SHA   string `json:"sha256"`
		Count int    `json:"catalog_count"`
	}
	b, e := os.ReadFile(*manifest)
	if e != nil {
		log.Fatal("catalog manifest missing")
	}
	json.Unmarshal(b, &mf)
	sum := sha256.Sum256(raw)
	if mf.SHA != hex.EncodeToString(sum[:]) || mf.Count != len(films) {
		log.Fatal("catalog does not match manifest")
	}
	ids := map[string]bool{}
	for _, f := range films {
		if f.ID == "" || ids[f.ID] || f.Version == "" {
			log.Fatal("catalog IDs/version invalid")
		}
		ids[f.ID] = true
	}
	if len(films) < 200 || len(films) > 300 {
		log.Printf("catalog coverage incomplete: %d verified movies; readiness will report incomplete", len(films))
	}
	if e = os.MkdirAll(filepath.Dir(*db), 0700); e != nil {
		log.Fatal(e)
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen3.5:4b"
	}
	supplier := &workflow.Watchmode{Key: os.Getenv("WATCHMODE_API_KEY"), Limit: *cap}
	origin := "http://" + *listen
	inviteOrigin := origin
	allowedHosts := []string{host}
	var lanListener net.Listener
	if *lan != "" {
		lanIP := net.ParseIP(*lan)
		if lanIP == nil || !lanIP.IsPrivate() || lanIP.IsLoopback() {
			log.Fatal("-lan must specify a private interface IP")
		}
		_, port, _ := net.SplitHostPort(*listen)
		lanAddress := net.JoinHostPort(*lan, port)
		lanListener, e = net.Listen("tcp", lanAddress)
		if e != nil {
			log.Fatal("LAN listener could not start: ", e)
		}
		defer lanListener.Close()
		inviteOrigin = "http://" + lanAddress
		allowedHosts = append(allowedHosts, *lan)
	}
	app, e := workflow.Open(workflow.Config{Database: *db, Films: films, Supplier: supplier, AllowedHosts: allowedHosts, InviteOrigin: inviteOrigin, OllamaEndpoint: os.Getenv("OLLAMA_ENDPOINT"), OllamaModel: model})
	if e != nil {
		log.Fatal(e)
	}
	defer app.Close()
	if e = app.SetMCPEndpoint(origin + "/mcp"); e != nil {
		log.Fatal(e)
	}
	server := &http.Server{Addr: *listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	fmt.Printf("CommonReel v2: %s\nCatalog: %d verified movies; database: %s\nNo public tunnel or firewall changes. Model and supplier readiness: /health\n", origin, len(films), *db)
	if lanListener != nil {
		fmt.Println("Same-Wi-Fi invitation origin: " + inviteOrigin)
		go func() {
			if err := server.Serve(lanListener); err != nil && err != http.ErrServerClosed {
				log.Print("LAN listener stopped")
			}
		}()
	}
	if e = server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}

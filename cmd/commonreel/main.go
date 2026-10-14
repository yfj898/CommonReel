package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"commonreel/internal/app"
	"commonreel/internal/catalog"
	"commonreel/internal/room"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8092", "loopback listen address")
	dir := flag.String("data", "data/ml-1m", "MovieLens directory")
	db := flag.String("db", "data/commonreel.db", "room database")
	demo := flag.Bool("demo", false, "explicit invented-data demo; not real-data evidence")
	stdio := flag.Bool("stdio", false, "serve MCP over stdio instead of HTTP")
	flag.Parse()
	var c *catalog.Catalog
	var err error
	if *demo {
		c = catalog.Fixture()
	} else {
		c, err = catalog.Load(*dir, 0.8)
		if err != nil {
			log.Fatal(err)
		}
	}
	store, err := room.Open(*db)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	a := app.New(store, c)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *stdio {
		if err := a.MCP.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
		return
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil {
		log.Fatal(err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		log.Fatal("this local research build binds only to a loopback IP")
	}
	server := &http.Server{Addr: *listen, Handler: a.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	fmt.Printf("CommonReel ready at http://%s\nMCP: http://%s/mcp (2025-11-25 / Streamable HTTP)\nDataset: %s; %d ratings\n", *listen, *listen, c.Meta.Source, c.Meta.Ratings)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

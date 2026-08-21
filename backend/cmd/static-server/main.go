package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	root := "/app/public"
	if value := os.Getenv("STATIC_ROOT"); value != "" {
		root = value
	}
	var proxy http.Handler
	if target := os.Getenv("API_PROXY_TARGET"); target != "" {
		proxyTarget, err := url.Parse(target)
		if err != nil {
			log.Fatalf("invalid API_PROXY_TARGET: %v", err)
		}
		proxy = httputil.NewSingleHostReverseProxy(proxyTarget)
	}
	fs := http.Dir(root)
	fileServer := http.FileServer(fs)
	spaHandler := newStaticHandler(root, fs, fileServer, "")
	sweetAdminHandler := newStaticHandler(root, fs, fileServer, "/sweet_admin")
	mux := http.NewServeMux()
	mux.HandleFunc("/sweet_admin", func(w http.ResponseWriter, r *http.Request) {
		if proxy != nil && shouldProxyToBackend(r.URL.Path) {
			proxy.ServeHTTP(w, r)
			return
		}
		sweetAdminHandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/sweet_admin/", func(w http.ResponseWriter, r *http.Request) {
		if proxy != nil && shouldProxyToBackend(r.URL.Path) {
			proxy.ServeHTTP(w, r)
			return
		}
		sweetAdminHandler.ServeHTTP(w, r)
	})
	mux.Handle("/", spaHandler)
	addr := ":80"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("static server listening on %s", addr)
	if err := serveStatic(ctx, listener, mux); err != nil {
		log.Fatal(err)
	}
}

func serveStatic(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	serveError := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveError <- err
	}()
	select {
	case err := <-serveError:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func shouldProxyToBackend(path string) bool {
	for _, prefix := range []string{"/sweet_admin/admin", "/sweet_admin/api", "/sweet_admin/files"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func newStaticHandler(root string, fs http.FileSystem, fileServer http.Handler, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if prefix != "" {
			path = strings.TrimPrefix(path, prefix)
			if path == "" {
				path = "/"
			}
		}
		name := strings.TrimPrefix(filepath.Clean(path), "/")
		if name == "" || name == "." {
			serveIndex(w, r, root)
			return
		}
		file, err := fs.Open(name)
		if err != nil {
			if isStaticAssetRequest(name) {
				http.NotFound(w, r)
				return
			}
			serveIndex(w, r, root)
			return
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil || stat.IsDir() {
			if isStaticAssetRequest(name) {
				http.NotFound(w, r)
				return
			}
			serveIndex(w, r, root)
			return
		}
		setStaticCacheHeader(w, name)
		if prefix == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		rewritten := new(http.Request)
		*rewritten = *r
		rewrittenURL := *r.URL
		rewrittenURL.Path = "/" + name
		rewritten.URL = &rewrittenURL
		fileServer.ServeHTTP(w, rewritten)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, root string) {
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filepath.Join(root, "index.html"))
}

func isStaticAssetRequest(name string) bool {
	if strings.HasPrefix(name, "assets/") || strings.HasPrefix(name, "icons/") || strings.HasPrefix(name, "resource/") {
		return true
	}
	return filepath.Ext(name) != ""
}

func setStaticCacheHeader(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

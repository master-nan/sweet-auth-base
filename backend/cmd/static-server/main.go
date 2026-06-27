package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	http.HandleFunc("/sweet_admin", func(w http.ResponseWriter, r *http.Request) {
		if proxy != nil && shouldProxyToBackend(r.URL.Path) {
			proxy.ServeHTTP(w, r)
			return
		}
		sweetAdminHandler.ServeHTTP(w, r)
	})
	http.HandleFunc("/sweet_admin/", func(w http.ResponseWriter, r *http.Request) {
		if proxy != nil && shouldProxyToBackend(r.URL.Path) {
			proxy.ServeHTTP(w, r)
			return
		}
		sweetAdminHandler.ServeHTTP(w, r)
	})
	http.Handle("/", spaHandler)
	addr := ":80"
	log.Printf("static server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
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

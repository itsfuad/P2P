package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"meshfile/internal/node"
)

//go:embed templates/* static/*
var content embed.FS

var (
	templates    *template.Template
	updates      = make(map[string]chan interface{})
	updatesMu    sync.RWMutex
	nodeInstance *node.Node // Assuming you have a global node instance
)

// SetNode sets the global node instance for the webui package.
func SetNode(n *node.Node) {
	nodeInstance = n
}

func Start(port int) error {
	var err error
	templates, err = template.ParseFS(content, "templates/*.html")
	if err != nil {
		return err
	}

	// Route handlers
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/updates", handleUpdates)
	http.HandleFunc("/api/peers", handlePeers)
	http.HandleFunc("/api/files", handleFiles)

	// Serve static files
	fileServer := http.FileServer(http.FS(content))
	http.Handle("/static/", fileServer)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting Web UI on http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	templates.ExecuteTemplate(w, "index.html", nil)
}

func handleUpdates(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("id")
	if clientID == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	updatesChan := make(chan interface{}, 1)
	updatesMu.Lock()
	updates[clientID] = updatesChan
	updatesMu.Unlock()

	defer func() {
		updatesMu.Lock()
		delete(updates, clientID)
		updatesMu.Unlock()
	}()

	select {
	case update := <-updatesChan:
		json.NewEncoder(w).Encode(update)
	case <-time.After(30 * time.Second):
		w.WriteHeader(http.StatusNoContent)
	}
}

func broadcastUpdate(update interface{}) {
	updatesMu.RLock()
	defer updatesMu.RUnlock()

	for _, ch := range updates {
		select {
		case ch <- update:
		default:
		}
	}
}

func handlePeers(w http.ResponseWriter, r *http.Request) {
	if nodeInstance == nil {
		http.Error(w, "Node not initialized", http.StatusInternalServerError)
		return
	}

	peers := nodeInstance.ListPeers()
	peerList := make([]map[string]interface{}, 0, len(peers))
	for _, peer := range peers {
		peerList = append(peerList, map[string]interface{}{
			"address":  peer.Address,
			"lastSeen": peer.LastSeen,
		})
	}
	json.NewEncoder(w).Encode(peerList)
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	if nodeInstance == nil {
		http.Error(w, "Node not initialized", http.StatusInternalServerError)
		return
	}

	files := nodeInstance.ListFiles()
	fileList := make([]map[string]interface{}, 0, len(files))
	for _, file := range files {
		fileList = append(fileList, map[string]interface{}{
			"name": file.Name,
			"size": file.Size,
			"hash": fmt.Sprintf("%x", file.Hash),
		})
	}
	json.NewEncoder(w).Encode(fileList)
}

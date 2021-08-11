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
	updatesMu    sync.RWMutexnode      *Node // Assuming you have a global node instance
	nodeInstance *node.Node // Assuming you have a global node instance)
)
t int) error {
// SetNode sets the global node instance for the webui package.
func SetNode(n *node.Node) {= template.ParseFS(content, "templates/*.html")
	nodeInstance = nil {
}return err
	}
func Start(port int) error {
	var err error
	templates, err = template.ParseFS(content, "templates/*.html")
	if err != nil {tes)
		return err
	}	http.HandleFunc("/api/files", handleFiles)

	// Route handlers
	http.HandleFunc("/", handleHome)FS(content))
	http.HandleFunc("/api/updates", handleUpdates)	http.Handle("/static/", fileServer)
	http.HandleFunc("/api/peers", handlePeers)
	http.HandleFunc("/api/files", handleFiles)
/localhost%s", addr)
	// Serve static filesreturn http.ListenAndServe(addr, nil)
	fileServer := http.FileServer(http.FS(content))}
	http.Handle("/static/", fileServer)
ResponseWriter, r *http.Request) {
	addr := fmt.Sprintf(":%d", port) {
	log.Printf("Starting Web UI on http://localhost%s", addr)otFound(w, r)
	return http.ListenAndServe(addr, nil)return
}
templates.ExecuteTemplate(w, "index.html", nil)
func handleHome(w http.ResponseWriter, r *http.Request) {}
	if r.URL.Path != "/" {
		http.NotFound(w, r)iter, r *http.Request) {
		returnuery().Get("id")
	}
	templates.ExecuteTemplate(w, "index.html", nil)rror(w, "Client ID required", http.StatusBadRequest)
}return
	}
func handleUpdates(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("id")ake(chan interface{}, 1)
	if clientID == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)= updatesChan
		return	updatesMu.Unlock()
	}

	updatesChan := make(chan interface{}, 1)
	updatesMu.Lock()ientID)
	updates[clientID] = updatesChandatesMu.Unlock()
	updatesMu.Unlock()	}()

	defer func() {
		updatesMu.Lock()
		delete(updates, clientID)
		updatesMu.Unlock()
	}()w.WriteHeader(http.StatusNoContent)
}
	select {}
	case update := <-updatesChan:
		json.NewEncoder(w).Encode(update)te(update interface{}) {
	case <-time.After(30 * time.Second):
		w.WriteHeader(http.StatusNoContent)	defer updatesMu.RUnlock()
	}
} := range updates {

func broadcastUpdate(update interface{}) {<- update:
	updatesMu.RLock()efault:
	defer updatesMu.RUnlock()}
}
	for _, ch := range updates {}
		select {
		case ch <- update:ttp.Request) {
		default:
		}nalServerError)
	}
}

func handlePeers(w http.ResponseWriter, r *http.Request) {peers := node.ListPeers()
	if nodeInstance == nil {	peerList := make([]map[string]interface{}, 0, len(peers))
		http.Error(w, "Node not initialized", http.StatusInternalServerError)
		returnerface{}{
	}

	peers := nodeInstance.ListPeers()
	peerList := make([]map[string]interface{}, 0, len(peers))
	for _, peer := range peers {st)
		peerList = append(peerList, map[string]interface{}{
			"address":  peer.Address,
























}	json.NewEncoder(w).Encode(fileList)	}		})			"hash": fmt.Sprintf("%x", file.Hash),			"size": file.Size,			"name": file.Name,		fileList = append(fileList, map[string]interface{}{	for _, file := range files {	fileList := make([]map[string]interface{}, 0, len(files))	files := nodeInstance.ListFiles()	}		return		http.Error(w, "Node not initialized", http.StatusInternalServerError)	if nodeInstance == nil {func handleFiles(w http.ResponseWriter, r *http.Request) {}	json.NewEncoder(w).Encode(peerList)	}		})			"lastSeen": peer.LastSeen,func handleFiles(w http.ResponseWriter, r *http.Request) {
	if node == nil {
		http.Error(w, "Node not initialized", http.StatusInternalServerError)
		return
	}

	files := node.ListFiles()
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

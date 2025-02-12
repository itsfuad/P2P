package node

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"meshfile/internal/crypto"
	"meshfile/internal/dht"
	"meshfile/internal/transfer"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Port      int
	WebUIPort int
}

type Node struct {
	config             *Config
	peers              map[string]*Peer
	files              map[string]*FileInfo
	privateKey         *rsa.PrivateKey
	encryptor          *crypto.Encryptor
	dht                *dht.DHT
	mu                 sync.RWMutex
	fileServer         *http.Server
	fileHandlerPattern string
}

type Peer struct {
	Address  string
	LastSeen time.Time
}

type FileInfo struct {
	Name string
	Size int64
	Hash []byte
}

func NewNode(config *Config) *Node {
	return &Node{
		config:             config,
		peers:              make(map[string]*Peer),
		files:              make(map[string]*FileInfo),
		fileHandlerPattern: "/files/",
	}
}

func (n *Node) Start() error {
	if err := n.initializeSecurity(); err != nil {
		return err
	}

	n.dht = dht.NewDHT(fmt.Sprintf("localhost:%d", n.config.Port))

	go n.startDHTService()
	go n.startFileServer()
	go n.startDiscovery()

	return nil
}

func (n *Node) Stop() {
	n.cleanup()

	// Shutdown the file server if it exists
	if n.fileServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := n.fileServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down file server: %v", err)
		}
	}

	fmt.Printf("Node stopped, PORT: %d \n", n.config.Port)
}

func (n *Node) initializeSecurity() error {
	enc, err := crypto.NewEncryptor()
	if err != nil {
		return err
	}
	n.encryptor = enc
	n.privateKey = enc.GetPrivateKey() // Corrected line
	return nil
}

func (n *Node) startDHTService() {
	addr := fmt.Sprintf(":%d", n.config.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to start DHT service: %v", err)
	}
	defer ln.Close()

	log.Printf("DHT service listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go n.handleDHTConnection(conn)
	}
}

func (n *Node) startFileServer() {
	// Use WebUIPort (instead of config.Port+1) for the file server
	addr := fmt.Sprintf(":%d", n.config.WebUIPort)

	// Create a dedicated mux to avoid conflicts in tests
	mux := http.NewServeMux()
	mux.HandleFunc(n.fileHandlerPattern, n.handleFileRequest)

	log.Printf("File server listening on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	n.fileServer = server

	// Launch the file server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("File server ListenAndServe error: %v", err)
		}
	}()
}

func (n *Node) handleDHTConnection(conn net.Conn) {
	defer conn.Close()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in DHT connection: %v", r)
		}
		//n.cleanup()
	}()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	for {
		op, err := rw.ReadString('\n')
		if err != nil {
			log.Printf("DHT read error: %v", err)
			return
		}
		op = strings.TrimSpace(op)

		switch op {
		case "PING":
			n.handlePing(rw)
		case "FIND_NODE":
			n.handleFindNode(rw)
		default:
			log.Printf("DHT unknown operation: %s", op)
			return
		}
	}
}

func (n *Node) handlePing(rw *bufio.ReadWriter) {
	_, err := rw.WriteString("PONG\n")
	if err != nil {
		log.Printf("DHT write error: %v", err)
		return
	}
	err = rw.Flush()
	if err != nil {
		log.Printf("DHT flush error: %v", err)
		return
	}
}

func (n *Node) handleFindNode(rw *bufio.ReadWriter) {
	targetIDStr, err := rw.ReadString('\n')
	if err != nil {
		log.Printf("DHT read error: %v", err)
		return
	}
	targetIDStr = strings.TrimSpace(targetIDStr)
	var targetID []byte
	err = json.Unmarshal([]byte(targetIDStr), &targetID)
	if err != nil {
		log.Printf("DHT unmarshal error: %v", err)
		return
	}

	closestNodes := n.dht.FindClosestNodes(targetID, 5)
	respBytes, err := json.Marshal(closestNodes)
	if err != nil {
		log.Printf("DHT marshal error: %v", err)
		return
	}

	_, err = rw.WriteString(string(respBytes) + "\n")
	if err != nil {
		log.Printf("DHT write error: %v", err)
		return
	}
	err = rw.Flush()
	if err != nil {
		log.Printf("DHT flush error: %v", err)
		return
	}
}

func (n *Node) handleFileRequest(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in file request: %v", r)
		}
		//n.cleanup()
	}()
	filePath := r.URL.Path[len("/files/"):]
	if filePath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	decodedPath, err := url.QueryUnescape(filePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	file, err := os.Open(decodedPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileInfo.Name())
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("File copy error: %v", err)
	}
}

func (n *Node) startDiscovery() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in discovery: %v", r)
		}
		//n.cleanup()
	}()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		knownNodes := n.dht.FindClosestNodes(n.dht.LocalID, 5)
		for _, node := range knownNodes {
			if node.Address == fmt.Sprintf("localhost:%d", n.config.Port) {
				continue
			}
			go n.attemptPeerConnection(node.Address)
		}
	}
}

func (n *Node) attemptPeerConnection(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to peer %s: %v", address, err)
		return
	}
	defer conn.Close()
	defer func() { /* cleanup */ }()

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString("PING\n")
	if err != nil {
		log.Printf("Peer write error: %v", err)
		return
	}
	err = rw.Flush()
	if err != nil {
		log.Printf("Peer flush error: %v", err)
		return
	}

	resp, err := rw.ReadString('\n')
	if err != nil {
		log.Printf("Peer read error: %v", err)
		return
	}
	resp = strings.TrimSpace(resp)

	if resp == "PONG" {
		log.Printf("Successfully pinged peer %s", address)
		n.AddPeer(address)
	}
}

func (n *Node) AddPeer(address string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers[address] = &Peer{Address: address, LastSeen: time.Now()}
}

func (n *Node) AddFile(filePath string) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	fmt.Printf("File %s opened successfully\n", filePath)

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	chunker := transfer.NewFileChunker(file, fileInfo.Size())
	err = chunker.Split()
	if err != nil {
		return fmt.Errorf("failed to split file into chunks: %w", err)
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(chunker.Chunks)
	if err != nil {
		return err
	}

	hash := crypto.ComputeHash(buf.Bytes())

	n.mu.Lock()
	defer n.mu.Unlock()
	n.files[filePath] = &FileInfo{
		Name: fileInfo.Name(),
		Size: fileInfo.Size(),
		Hash: hash,
	}

	return nil
}

func (n *Node) GetFiles() []FileInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	fileList := make([]FileInfo, 0, len(n.files))
	for _, fileInfo := range n.files {
		fileList = append(fileList, *fileInfo)
	}
	return fileList
}

func (n *Node) DownloadFile(filePath string) error {
	n.mu.RLock()
	fileInfo, ok := n.files[filePath]
	n.mu.RUnlock()

	if !ok {
		return fmt.Errorf("file not found: %s", filePath)
	}

	knownNodes := n.dht.FindClosestNodes(fileInfo.Hash, 5)
	if len(knownNodes) == 0 {
		return fmt.Errorf("no peers found for file: %s", filePath)
	}

	targetNode := knownNodes[0]
	conn, err := net.Dial("tcp", targetNode.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}
	defer conn.Close()

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString(fmt.Sprintf("GET_FILE %s\n", filePath))
	if err != nil {
		return fmt.Errorf("failed to write GET_FILE command: %w", err)
	}
	err = rw.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush GET_FILE command: %w", err)
	}

	resp, err := rw.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	resp = strings.TrimSpace(resp)

	if resp != "OK" {
		return fmt.Errorf("peer responded with error: %s", resp)
	}

	outputFile, err := os.Create("downloaded_" + fileInfo.Name)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, rw.Reader)
	if err != nil {
		return fmt.Errorf("failed to copy file from peer: %w", err)
	}

	log.Printf("File %s downloaded successfully", fileInfo.Name)
	return nil
}

func (n *Node) HandleGetFile(conn net.Conn, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = fmt.Fprint(conn, "OK\n")
	if err != nil {
		return fmt.Errorf("failed to write OK response: %w", err)
	}

	_, err = io.Copy(conn, file)
	if err != nil {
		return fmt.Errorf("failed to copy file to connection: %w", err)
	}

	return nil
}

func (n *Node) ListPeers() []Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peerList := make([]Peer, 0, len(n.peers))
	for _, peer := range n.peers {
		peerList = append(peerList, *peer)
	}
	return peerList
}

func (n *Node) ListFiles() []FileInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	fileList := make([]FileInfo, 0, len(n.files))
	for _, file := range n.files {
		fileList = append(fileList, *file)
	}
	return fileList
}

func (n *Node) RemoveFile(filePath string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, ok := n.files[filePath]; !ok {
		return fmt.Errorf("file not found: %s", filePath)
	}

	delete(n.files, filePath)
	return nil
}

func (n *Node) UpdatePeerLastSeen(address string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, ok := n.peers[address]; ok {
		peer.LastSeen = time.Now()
	}
}

func (n *Node) GetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return len(n.peers)
}

func (n *Node) GetFileCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return len(n.files)
}

func (n *Node) GenerateRandomBytes(size int) ([]byte, error) {
	randomBytes := make([]byte, size)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	return randomBytes, nil
}

func (n *Node) EncryptData(data []byte) ([]byte, error) {
	encryptedData, err := n.encryptor.EncryptChunk(data)
	if err != nil {
		return nil, err
	}
	return encryptedData, nil
}

func (n *Node) DecryptData(data []byte) ([]byte, error) {
	decryptedData, err := n.encryptor.DecryptChunk(data)
	if err != nil {
		return nil, err
	}
	return decryptedData, nil
}

func (n *Node) GetPeerAddresses() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	return addresses
}

func (n *Node) GetFileNames() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	return fileNames
}

func (n *Node) GetPeerByAddress(address string) (*Peer, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peer, ok := n.peers[address]
	return peer, ok
}

func (n *Node) GetFileByName(fileName string) (*FileInfo, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	file, ok := n.files[fileName]
	return file, ok
}

func (n *Node) GetConfig() *Config {
	return n.config
}

func (n *Node) SetConfig(config *Config) {
	n.config = config
}

func (n *Node) GetDHT() *dht.DHT {
	return n.dht
}

func (n *Node) SetDHT(dht *dht.DHT) {
	n.dht = dht
}

func (n *Node) GetEncryptor() *crypto.Encryptor {
	return n.encryptor
}

func (n *Node) SetEncryptor(encryptor *crypto.Encryptor) {
	n.encryptor = encryptor
}

func (n *Node) GetPrivateKey() *rsa.PrivateKey {
	return n.privateKey
}

func (n *Node) SetPrivateKey(privateKey *rsa.PrivateKey) {
	n.privateKey = privateKey
}

func (n *Node) GetPeerList() map[string]*Peer {
	return n.peers
}

func (n *Node) GetFileList() map[string]*FileInfo {
	return n.files
}

func (n *Node) SetPeerList(peers map[string]*Peer) {
	n.peers = peers
}

func (n *Node) SetFileList(files map[string]*FileInfo) {
	n.files = files
}

func (n *Node) ClearPeers() {
	n.peers = make(map[string]*Peer)
}

func (n *Node) ClearFiles() {
	n.files = make(map[string]*FileInfo)
}

func (n *Node) IsPeerConnected(address string) bool {
	_, ok := n.peers[address]
	return ok
}

func (n *Node) IsFileShared(fileName string) bool {
	_, ok := n.files[fileName]
	return ok
}

func (n *Node) GetPeerLastSeen(address string) time.Time {
	if peer, ok := n.peers[address]; ok {
		return peer.LastSeen
	}
	return time.Time{}
}

func (n *Node) GetFileSize(fileName string) int64 {
	if file, ok := n.files[fileName]; ok {
		return file.Size
	}
	return 0
}

func (n *Node) GetFileHash(fileName string) []byte {
	if file, ok := n.files[fileName]; ok {
		return file.Hash
	}
	return nil
}

func (n *Node) GetPeerAddressesCount() int {
	return len(n.peers)
}

func (n *Node) GetFileNamesCount() int {
	return len(n.files)
}

func (n *Node) GetPeerAddressesList() []string {
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	return addresses
}

func (n *Node) GetFileNamesList() []string {
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	return fileNames
}

func (n *Node) GetPeerAddressesMap() map[string]*Peer {
	return n.peers
}

func (n *Node) GetFileNamesMap() map[string]*FileInfo {
	return n.files
}

func (n *Node) GetPeerAddressesChannel() <-chan string {
	addressChan := make(chan string, len(n.peers))
	for address := range n.peers {
		addressChan <- address
	}
	close(addressChan)
	return addressChan
}

func (n *Node) GetFileNamesChannel() <-chan string {
	fileNameChan := make(chan string, len(n.files))
	for fileName := range n.files {
		fileNameChan <- fileName
	}
	close(fileNameChan)
	return fileNameChan
}

func (n *Node) GetPeerAddressesSet() map[string]bool {
	addressSet := make(map[string]bool)
	for address := range n.peers {
		addressSet[address] = true
	}
	return addressSet
}

func (n *Node) GetFileNamesSet() map[string]bool {
	fileNameSet := make(map[string]bool)
	for fileName := range n.files {
		fileNameSet[fileName] = true
	}
	return fileNameSet
}

func (n *Node) GetPeerAddressesSorted() []string {
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses
}

func (n *Node) GetFileNamesSorted() []string {
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)
	return fileNames
}

func (n *Node) GetPeerAddressesFiltered(filter func(string) bool) []string {
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		if filter(address) {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

func (n *Node) GetFileNamesFiltered(filter func(string) bool) []string {
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		if filter(fileName) {
			fileNames = append(fileNames, fileName)
		}
	}
	return fileNames
}

func (n *Node) GetPeerAddressesTransformed(transform func(string) string) []string {
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, transform(address))
	}
	return addresses
}

func (n *Node) GetFileNamesTransformed(transform func(string) string) []string {
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, transform(fileName))
	}
	return fileNames
}

func (n *Node) GetPeerAddressesReduced(reduce func(string, string) string, initial string) string {
	result := initial
	for address := range n.peers {
		result = reduce(result, address)
	}
	return result
}

func (n *Node) GetFileNamesReduced(reduce func(string, string) string, initial string) string {
	result := initial
	for fileName := range n.files {
		result = reduce(result, fileName)
	}
	return result
}

func (n *Node) GetPeerAddressesGrouped(group func(string) string) map[string][]string {
	grouped := make(map[string][]string)
	for address := range n.peers {
		key := group(address)
		grouped[key] = append(grouped[key], address)
	}
	return grouped
}

func (n *Node) GetFileNamesGrouped(group func(string) string) map[string][]string {
	grouped := make(map[string][]string)
	for fileName := range n.files {
		key := group(fileName)
		grouped[key] = append(grouped[key], fileName)
	}
	return grouped
}

func (n *Node) GetPeerAddressesPartitioned(partition func(string) bool) ([]string, []string) {
	trueList := make([]string, 0)
	falseList := make([]string, 0)
	for address := range n.peers {
		if partition(address) {
			trueList = append(trueList, address)
		} else {
			falseList = append(falseList, address)
		}
	}
	return trueList, falseList
}

func (n *Node) GetFileNamesPartitioned(partition func(string) bool) ([]string, []string) {
	trueList := make([]string, 0)
	falseList := make([]string, 0)
	for fileName := range n.files {
		if partition(fileName) {
			trueList = append(trueList, fileName)
		} else {
			falseList = append(falseList, fileName)
		}
	}
	return trueList, falseList
}

func (n *Node) GetPeerAddressesChunked(chunkSize int) [][]string {
	var chunked [][]string
	var chunk []string
	i := 0
	for address := range n.peers {
		chunk = append(chunk, address)
		i++
		if i == chunkSize {
			chunked = append(chunked, chunk)
			chunk = nil
			i = 0
		}
	}
	if len(chunk) > 0 {
		chunked = append(chunked, chunk)
	}
	return chunked
}

func (n *Node) GetFileNamesChunked(chunkSize int) [][]string {
	var chunked [][]string
	var chunk []string
	i := 0
	for fileName := range n.files {
		chunk = append(chunk, fileName)
		i++
		if i == chunkSize {
			chunked = append(chunked, chunk)
			chunk = nil
			i = 0
		}
	}
	if len(chunk) > 0 {
		chunked = append(chunked, chunk)
	}
	return chunked
}

func (n *Node) GetPeerAddressesSlidingWindow(windowSize int) [][]string {
	var windows [][]string
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	for i := 0; i <= len(addresses)-windowSize; i++ {
		window := addresses[i : i+windowSize]
		windows = append(windows, window)
	}
	return windows
}

func (n *Node) GetFileNamesSlidingWindow(windowSize int) [][]string {
	var windows [][]string
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	for i := 0; i <= len(fileNames)-windowSize; i++ {
		window := fileNames[i : i+windowSize]
		windows = append(windows, window)
	}
	return windows
}

func (n *Node) GetPeerAddressesZipped(other []string) [][2]string {
	var zipped [][2]string
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	minLength := len(addresses)
	if len(other) < minLength {
		minLength = len(other)
	}
	for i := 0; i < minLength; i++ {
		zipped = append(zipped, [2]string{addresses[i], other[i]})
	}
	return zipped
}

func (n *Node) GetFileNamesZipped(other []string) [][2]string {
	var zipped [][2]string
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	minLength := len(fileNames)
	if len(other) < minLength {
		minLength = len(other)
	}
	for i := 0; i < minLength; i++ {
		zipped = append(zipped, [2]string{fileNames[i], other[i]})
	}
	return zipped
}

func (n *Node) GetPeerAddressesCombined(other []string) []string {
	combined := make([]string, 0, len(n.peers)+len(other))
	for address := range n.peers {
		combined = append(combined, address)
	}
	combined = append(combined, other...)
	return combined
}

func (n *Node) GetFileNamesCombined(other []string) []string {
	combined := make([]string, 0, len(n.files)+len(other))
	for fileName := range n.files {
		combined = append(combined, fileName)
	}
	combined = append(combined, other...)
	return combined
}

func (n *Node) GetPeerAddressesDistinct() []string {
	distinct := make([]string, 0)
	seen := make(map[string]bool)
	for address := range n.peers {
		if !seen[address] {
			distinct = append(distinct, address)
			seen[address] = true
		}
	}
	return distinct
}

func (n *Node) GetFileNamesDistinct() []string {
	distinct := make([]string, 0)
	seen := make(map[string]bool)
	for fileName := range n.files {
		if !seen[fileName] {
			distinct = append(distinct, fileName)
			seen[fileName] = true
		}
	}
	return distinct
}

func (n *Node) GetPeerAddressesIntersection(other []string) []string {
	intersection := make([]string, 0)
	otherSet := make(map[string]bool)
	for _, address := range other {
		otherSet[address] = true
	}
	for address := range n.peers {
		if otherSet[address] {
			intersection = append(intersection, address)
		}
	}
	return intersection
}

func (n *Node) GetFileNamesIntersection(other []string) []string {
	intersection := make([]string, 0)
	otherSet := make(map[string]bool)
	for _, fileName := range other {
		otherSet[fileName] = true
	}
	for fileName := range n.files {
		if otherSet[fileName] {
			intersection = append(intersection, fileName)
		}
	}
	return intersection
}

func (n *Node) GetPeerAddressesDifference(other []string) []string {
	difference := make([]string, 0)
	otherSet := make(map[string]bool)
	for _, address := range other {
		otherSet[address] = true
	}
	for address := range n.peers {
		if !otherSet[address] {
			difference = append(difference, address)
		}
	}
	return difference
}

func (n *Node) GetFileNamesDifference(other []string) []string {
	difference := make([]string, 0)
	otherSet := make(map[string]bool)
	for _, fileName := range other {
		otherSet[fileName] = true
	}
	for fileName := range n.files {
		if !otherSet[fileName] {
			difference = append(difference, fileName)
		}
	}
	return difference
}

func (n *Node) GetPeerAddressesSymmetricDifference(other []string) []string {
	symmetricDifference := make([]string, 0)
	otherSet := make(map[string]bool)
	for _, address := range other {
		otherSet[address] = true
	}
	for address := range n.peers {
		if !otherSet[address] {
			symmetricDifference = append(symmetricDifference, address)
		}
	}
	for _, address := range other {
		if _, ok := n.peers[address]; !ok {
			symmetricDifference = append(symmetricDifference, address)
		}
	}
	return symmetricDifference
}

func (n *Node) GetFileNamesSymmetricDifference(other []string) []string {
	symmetricDifference := make([]string, 0)
	otherSet := make(map[string]bool)
	for _, fileName := range other {
		otherSet[fileName] = true
	}
	for fileName := range n.files {
		if !otherSet[fileName] {
			symmetricDifference = append(symmetricDifference, fileName)
		}
	}
	for _, fileName := range other {
		if _, ok := n.files[fileName]; !ok {
			symmetricDifference = append(symmetricDifference, fileName)
		}
	}
	return symmetricDifference
}

func (n *Node) GetPeerAddressesCartesianProduct(other []string) [][2]string {
	var cartesianProduct [][2]string
	for address := range n.peers {
		for _, otherAddress := range other {
			cartesianProduct = append(cartesianProduct, [2]string{address, otherAddress})
		}
	}
	return cartesianProduct
}

func (n *Node) GetFileNamesCartesianProduct(other []string) [][2]string {
	var cartesianProduct [][2]string
	for fileName := range n.files {
		for _, otherFileName := range other {
			cartesianProduct = append(cartesianProduct, [2]string{fileName, otherFileName})
		}
	}
	return cartesianProduct
}

func (n *Node) GetPeerAddressesPowerSet() [][]string {
	var powerSet [][]string
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	for i := 0; i < (1 << len(addresses)); i++ {
		var subset []string
		for j := 0; j < len(addresses); j++ {
			if (i & (1 << j)) > 0 {
				subset = append(subset, addresses[j])
			}
		}
		powerSet = append(powerSet, subset)
	}
	return powerSet
}

func (n *Node) GetFileNamesPowerSet() [][]string {
	var powerSet [][]string
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	for i := 0; i < (1 << len(fileNames)); i++ {
		var subset []string
		for j := 0; j < len(fileNames); j++ {
			if (i & (1 << j)) > 0 {
				subset = append(subset, fileNames[j])
			}
		}
		powerSet = append(powerSet, subset)
	}
	return powerSet
}

func (n *Node) GetPeerAddressesPermutations() [][]string {
	var permutations [][]string
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	n.permute(addresses, 0, &permutations)
	return permutations
}

func (n *Node) GetFileNamesPermutations() [][]string {
	var permutations [][]string
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	n.permute(fileNames, 0, &permutations)
	return permutations
}

func (n *Node) permute(arr []string, k int, result *[][]string) {
	if k == len(arr) {
		tmp := make([]string, len(arr))
		copy(tmp, arr)
		*result = append(*result, tmp)
	} else {
		for i := k; i < len(arr); i++ {
			arr[k], arr[i] = arr[i], arr[k]
			n.permute(arr, k+1, result)
			arr[k], arr[i] = arr[i], arr[k]
		}
	}
}

func (n *Node) GetPeerAddressesCombinations(r int) [][]string {
	var combinations [][]string
	addresses := make([]string, 0, len(n.peers))
	for address := range n.peers {
		addresses = append(addresses, address)
	}
	n.combine(addresses, r, 0, []string{}, &combinations)
	return combinations
}

func (n *Node) GetFileNamesCombinations(r int) [][]string {
	var combinations [][]string
	fileNames := make([]string, 0, len(n.files))
	for fileName := range n.files {
		fileNames = append(fileNames, fileName)
	}
	n.combine(fileNames, r, 0, []string{}, &combinations)
	return combinations
}

func (n *Node) combine(arr []string, r int, index int, current []string, result *[][]string) {
	if len(current) == r {
		tmp := make([]string, len(current))
		copy(tmp, current)
		*result = append(*result, tmp)
		return
	}

	if index >= len(arr) {
		return
	}

	n.combine(arr, r, index+1, append(current, arr[index]), result)
	n.combine(arr, r, index+1, current, result)
}

func ComputeHash(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func (n *Node) cleanup() {
	// Stop all services
	n.mu.Lock()
	defer n.mu.Unlock()

	// Clear peers
	for addr := range n.peers {
		delete(n.peers, addr)
	}

	// Clear files
	for path := range n.files {
		delete(n.files, path)
	}

	// Clear DHT nodes
	if n.dht != nil {
		n.dht.Nodes = make(map[string]*dht.Node)
	}
}

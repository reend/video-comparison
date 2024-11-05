package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sync"
)

type VideoData struct {
	MediaLink    string `json:"mediaLink"`
	ID string `json:"ID"`
}

type VideoRequest struct {
	URL          string      `json:"url"`
	Data         []VideoData `json:"data"`
	IsBigUrlDone int         `json:"isBigUrlDone"`
}

type VideoComparisonResponse struct {
	Results []string `json:"results"`
}

var videoBuffers = struct {
	sync.RWMutex
	data map[string][]byte
}{data: make(map[string][]byte)}

var isDownloaded bool

var isBigUrlDone int
var bigUrlValue string

func downloadFileToBuffer(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: received status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

func computeHash(data []byte) string {
	hash := sha1.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func areVideosIdentical(hash1 string, buffer1 []byte, buffer2 []byte) bool {
	hash2 := computeHash(buffer2)
	if hash1 != hash2 {
		return false
	}

	if len(buffer1) != len(buffer2) {
		return false
	}

	for i := range buffer1 {
		if buffer1[i] != buffer2[i] {
			return false
		}
	}

	return true
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	isDownloaded = false

	var req VideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	chunkSize := 8
	totalVideos := len(req.Data)
	log.Printf("Starting download of %d videos", totalVideos)

	for i := 0; i < totalVideos; i += chunkSize {
		end := i + chunkSize
		if end >= totalVideos {
			end = totalVideos
			isDownloaded = true
		}

		var wg sync.WaitGroup
		for _, item := range req.Data[i:end] {
			videoBuffers.RLock()
			_, exists := videoBuffers.data[item.ID]
			videoBuffers.RUnlock()

			if !exists {
				wg.Add(1)
				go func(item VideoData) {
					defer wg.Done()
					buffer, err := downloadFileToBuffer(item.MediaLink)
					if err != nil {
						log.Printf("Error downloading video %s: %v", item.ID, err)
						return
					}

					videoBuffers.Lock()
					videoBuffers.data[item.ID] = buffer
					videoBuffers.Unlock()
				}(item)
			}
		}

		wg.Wait()
		log.Printf("Chunk %d-%d downloaded", i+1, end)
	}

	videoBuffers.RLock()
	downloadedCount := len(videoBuffers.data)
	videoBuffers.RUnlock()

	log.Printf("Downloaded %d/%d videos", downloadedCount, totalVideos)

	if isDownloaded {
		log.Println("All videos have been successfully downloaded.")

		totalSize := 0
		videoBuffers.RLock()
		for _, buffer := range videoBuffers.data {
			totalSize += len(buffer)
		}
		videoBuffers.RUnlock()

		totalSizeMB := float64(totalSize) / (1024 * 1024)
		log.Printf("Total size of all downloaded videos: %.2f MB", totalSizeMB)
	}

	w.WriteHeader(http.StatusOK)
}

func CompareHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isDownloaded {
		http.Error(w, "Videos are not fully downloaded", http.StatusBadRequest)
		return
	}

	var req VideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	isBigUrlDone = req.IsBigUrlDone

	if isBigUrlDone == 1 {
		bigUrlValue = req.URL
		http.Error(w, "Processing big URL part 1", http.StatusOK)
		return
	} else if isBigUrlDone == 2 {
		bigUrlValue += req.URL
		isBigUrlDone = 0
		req.URL = bigUrlValue
	}

	re := regexp.MustCompile(`^data:video\/mp4;base64,`)
	cleanedURL := re.ReplaceAllString(req.URL, "")

	buffer1, err := base64.StdEncoding.DecodeString(cleanedURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode base64 main video: %v", err), http.StatusBadRequest)
		return
	}

	hash1 := computeHash(buffer1)

	var results []string
	for _, item := range req.Data {
		videoBuffers.RLock()
		buffer2, exists := videoBuffers.data[item.ID]
		videoBuffers.RUnlock()

		if exists && areVideosIdentical(hash1, buffer1, buffer2) {
			results = append(results, item.ID)
		}
	}

	response := VideoComparisonResponse{Results: results}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}

func TriggerHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    log.Println("triggered")
    w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/download", DownloadHandler)
	http.HandleFunc("/compare", CompareHandler)
	http.HandleFunc("/trigger", TriggerHandler)

	port := "8080"
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

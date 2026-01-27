package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/HuyTanVan/objectstore/internal/storage"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

func main() {
	// load .env file if it exists (for local development)
	// Railway will use environment variables from dashboard
	if err := godotenv.Load(); err != nil {
		log.Println("no .env found -> using environment variables on Railway")
	}

	// init directories first
	if err := storage.InitDirs(); err != nil {
		log.Fatal("Failed to create directories:", err)
	}

	// init DB
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Routes
	mux.HandleFunc("PUT /buckets/{bucket}/objects/{key}", putObject)
	mux.HandleFunc("GET /buckets/{bucket}/objects/{key}", getObject)
	mux.HandleFunc("DELETE /buckets/{bucket}/objects/{key}", deleteObject)
	mux.HandleFunc("HEAD /buckets/{bucket}/objects/{key}", headObject)
	mux.HandleFunc("GET /objects", listObjects)

	// get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("port is not set. please set PORT environment variable")
	}

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("node id is not set. please set NODE_ID environment variable")
	}
	// if nodeID == "" {
	// 	nodeID = "node-1"
	// }

	log.Printf("===========================================")
	log.Printf("Node ID: %s", nodeID)
	log.Printf("Data Directory: %s", storage.DataDir)
	log.Printf("Server starting on port %s", port)
	log.Printf("===========================================")
	log.Println("Routes:")
	log.Println("  PUT    /buckets/{bucket}/objects/{key}")
	log.Println("  GET    /buckets/{bucket}/objects/{key}")
	log.Println("  DELETE /buckets/{bucket}/objects/{key}")
	log.Println("  HEAD   /buckets/{bucket}/objects/{key}")
	log.Println("  GET    /objects")
	log.Printf("===========================================")

	log.Fatal(http.ListenAndServe(":"+port, enableCORS(mux)))
}

// CORS middleware
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initDB() error {
	// init SQLite database - path = ./data/metadata.db
	dbPath := filepath.Join(storage.DataDir, "metadata.db")

	var err error
	storage.DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	_, err = storage.DB.Exec(`CREATE TABLE IF NOT EXISTS objects (
		bucket TEXT,
		key TEXT,
		hash TEXT,
		size INTEGER,
		PRIMARY KEY(bucket, key)
	)`)

	if err != nil {
		return err
	}

	log.Printf("database initialized at: %s", dbPath)
	return nil
}

// endpoint: PUT /buckets/{bucket}/objects/{key}
func putObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := strings.TrimPrefix(
		r.URL.Path,
		"/buckets/"+bucket+"/objects/",
	)

	hash, err := storage.PutObject(bucket, key, r.Body)
	if err != nil {
		log.Printf("Error uploading object: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	log.Printf("Object uploaded: %s/%s -> %s", bucket, key, hash[:12])

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", hash)
	w.WriteHeader(http.StatusOK)

	resp := map[string]string{
		"message": "Upload successful",
		"path":    r.URL.Path,
		"etag":    hash,
	}

	json.NewEncoder(w).Encode(resp)
}

// endpoint: GET /buckets/{bucket}/objects/{key}
func getObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := strings.TrimPrefix(
		r.URL.Path,
		"/buckets/"+bucket+"/objects/",
	)

	reader, err := storage.GetObject(bucket, key)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	defer reader.Close()

	log.Printf("object downloaded: %s/%s", bucket, key)
	io.Copy(w, reader)
}

// endpoint: GET /buckets/{bucket}/objects/{key}
func deleteObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := strings.TrimPrefix(
		r.URL.Path,
		"/buckets/"+bucket+"/objects/",
	)

	err := storage.DeleteObject(bucket, key)
	if err != nil {
		if err == storage.ErrNotFound {
			http.Error(w, "Object not found", 404)
		} else {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	log.Printf("Object deleted: %s/%s", bucket, key)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Delete successful",
	})
}

// endpoint: HEAD /buckets/{bucket}/objects/{key}
func headObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := strings.TrimPrefix(
		r.URL.Path,
		"/buckets/"+bucket+"/objects/",
	)

	meta, err := storage.HeadObject(bucket, key)
	if err != nil {
		http.Error(w, "Object not found", 404)
		return
	}

	w.Header().Set("Content-Length", string(meta.Size))
	w.Header().Set("ETag", meta.Hash)
	w.WriteHeader(http.StatusOK)
}

// endpoint: GET /objects?bucket={bucket}
func listObjects(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")

	objects, err := storage.ListObjects(bucket)
	if err != nil {
		log.Printf("Error listing objects: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(objects)
}

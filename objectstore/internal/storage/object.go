package storage

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var (
	// Use environment variable, fallback to ./data
	DataDir = getEnv("DATA_DIR", "./data")
	TmpDir  = filepath.Join(DataDir, "tmp")
	ObjDir  = filepath.Join(DataDir, "objects")
	DB      *sql.DB
)

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// InitDirs creates necessary directories on startup
func InitDirs() error {
	// dirs = ["./data/tmp", "./data/objects"]
	dirs := []string{DataDir, TmpDir, ObjDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", dir, err)
		}
	}
	return nil
}

func PutObject(bucket, key string, r io.Reader) (string, error) {
	// 1. create temp file
	tmpFile := filepath.Join(TmpDir, uuid.New().String())
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}

	// 2. Write + hash
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(f, h), r)
	if err != nil {
		f.Close()
		os.Remove(tmpFile)
		return "", err
	}

	// 3. Close file before rename (Windows-safe)
	if err := f.Close(); err != nil {
		os.Remove(tmpFile)
		return "", err
	}

	// 4. Compute hash and final path
	hash := fmt.Sprintf("%x", h.Sum(nil))
	finalDir := filepath.Join(ObjDir, hash[:2], hash[2:4])
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		os.Remove(tmpFile)
		return "", err
	}

	finalFile := filepath.Join(finalDir, hash)

	// 5. Move temp → final
	if err := os.Rename(tmpFile, finalFile); err != nil {
		os.Remove(tmpFile)
		return "", err
	}

	// 6. Upsert metadata
	_, err = DB.Exec(`INSERT OR REPLACE INTO objects(bucket,key,hash,size) VALUES(?,?,?,?)`,
		bucket, key, hash, size)
	if err != nil {
		return "", err
	}

	return hash, nil
}

// GetObject reads object bytes by bucket/key
func GetObject(bucket, key string) (io.ReadCloser, error) {
	row := DB.QueryRow(`SELECT hash FROM objects WHERE bucket=? AND key=?`, bucket, key)
	var hash string
	if err := row.Scan(&hash); err != nil {
		return nil, ErrNotFound
	}

	objPath := filepath.Join(ObjDir, hash[:2], hash[2:4], hash)
	file, err := os.Open(objPath)
	if err != nil {
		return nil, ErrNotFound
	}
	return file, nil
}

// DeleteObject removes metadata only (object file stays)
func DeleteObject(bucket, key string) error {
	res, err := DB.Exec(`DELETE FROM objects WHERE bucket=? AND key=?`, bucket, key)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// HeadObject returns metadata without reading file
func HeadObject(bucket, key string) (*ObjectMeta, error) {
	row := DB.QueryRow(`SELECT hash, size FROM objects WHERE bucket=? AND key=?`, bucket, key)
	var hash string
	var size int64
	if err := row.Scan(&hash, &size); err != nil {
		return nil, ErrNotFound
	}
	return &ObjectMeta{
		Bucket: bucket,
		Key:    key,
		Hash:   hash,
		Size:   size,
	}, nil
}

// ListObjects returns all objects, optionally filtered by bucket
func ListObjects(bucket string) ([]ObjectMeta, error) {
	var query string
	var args []interface{}

	if bucket != "" {
		query = `SELECT bucket, key, hash, size FROM objects WHERE bucket=? ORDER BY bucket, key`
		args = []interface{}{bucket}
	} else {
		query = `SELECT bucket, key, hash, size FROM objects ORDER BY bucket, key`
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []ObjectMeta
	for rows.Next() {
		var obj ObjectMeta
		if err := rows.Scan(&obj.Bucket, &obj.Key, &obj.Hash, &obj.Size); err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

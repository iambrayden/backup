package backup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/waylontech/backup/agent/internal/client"
	"github.com/waylontech/backup/pkg/chunker"
	"github.com/waylontech/backup/pkg/crypto"
)

type Engine struct {
	client    *client.Client
	watchPath string
	log       *zap.Logger
}

func New(c *client.Client, watchPath string, log *zap.Logger) *Engine {
	return &Engine{
		client:    c,
		watchPath: watchPath,
		log:       log,
	}
}

type Result struct {
	SnapshotID     string
	FileCount      int
	BytesScanned   int64
	ChunksTotal    int
	ChunksUploaded int
	ChunksDeduped  int
}

type fileInfo struct {
	path       string
	size       int64
	mtime      time.Time
	mode       fs.FileMode
	chunkHexes []string
}

func (e *Engine) Run(ctx context.Context, jobID string) (*Result, error) {
	dekB64, err := e.client.GetDEK()
	if err != nil {
		return nil, fmt.Errorf("get dek: %w", err)
	}
	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		return nil, fmt.Errorf("decode dek: %w", err)
	}

	entries, result, err := e.collectHashes(ctx)
	if err != nil {
		return nil, err
	}

	if err := e.uploadMissing(ctx, dek, result, entries); err != nil {
		return nil, err
	}

	snapshotID, err := e.client.CreateSnapshot(jobID, manifestEntries(entries))
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	result.SnapshotID = snapshotID
	return result, nil
}

func (e *Engine) collectHashes(ctx context.Context) ([]*fileInfo, *Result, error) {
	result := &Result{}
	var files []*fileInfo

	err := filepath.WalkDir(e.watchPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			e.log.Warn("walk", zap.String("path", path), zap.Error(walkErr))
			return nil
		}
		if d.IsDir() {
			files = append(files, &fileInfo{path: path, mode: d.Type()})
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		result.BytesScanned += info.Size()

		f, err := os.Open(path)
		if err != nil {
			e.log.Warn("open", zap.String("path", path), zap.Error(err))
			return nil
		}
		defer f.Close()

		chunks, err := chunker.ChunkReader(f)
		if err != nil {
			e.log.Warn("chunk", zap.String("path", path), zap.Error(err))
			return nil
		}

		fi := &fileInfo{
			path:  path,
			size:  info.Size(),
			mtime: info.ModTime(),
			mode:  info.Mode(),
		}
		for _, ch := range chunks {
			fi.chunkHexes = append(fi.chunkHexes, hex.EncodeToString(ch.Hash[:]))
		}
		files = append(files, fi)
		result.FileCount++
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk: %w", err)
	}

	seen := map[string]bool{}
	for _, fi := range files {
		for _, h := range fi.chunkHexes {
			seen[h] = true
		}
	}
	result.ChunksTotal = len(seen)

	return files, result, nil
}

func (e *Engine) uploadMissing(ctx context.Context, dek []byte, result *Result, files []*fileInfo) error {
	missingSet := map[string]bool{}

	var allHashes []string
	seen := map[string]bool{}
	for _, fi := range files {
		for _, h := range fi.chunkHexes {
			if !seen[h] {
				seen[h] = true
				allHashes = append(allHashes, h)
			}
		}
	}

	const batchSize = 1000
	for i := 0; i < len(allHashes); i += batchSize {
		end := i + batchSize
		if end > len(allHashes) {
			end = len(allHashes)
		}
		missing, err := e.client.CheckChunks(allHashes[i:end])
		if err != nil {
			return fmt.Errorf("check chunks: %w", err)
		}
		for _, h := range missing {
			missingSet[h] = true
		}
	}
	result.ChunksDeduped = len(allHashes) - len(missingSet)

	httpClient := &http.Client{Timeout: 5 * time.Minute}

	for _, fi := range files {
		if fi.size == 0 || len(fi.chunkHexes) == 0 {
			continue
		}

		needsUpload := false
		for _, h := range fi.chunkHexes {
			if missingSet[h] {
				needsUpload = true
				break
			}
		}
		if !needsUpload {
			continue
		}

		f, err := os.Open(fi.path)
		if err != nil {
			return fmt.Errorf("open %s: %w", fi.path, err)
		}

		chunks, err := chunker.ChunkReader(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("chunk %s: %w", fi.path, err)
		}

		for _, ch := range chunks {
			h := hex.EncodeToString(ch.Hash[:])
			if !missingSet[h] {
				continue
			}

			encrypted, err := crypto.EncryptChunk(dek, ch.Hash, ch.Data)
			if err != nil {
				return fmt.Errorf("encrypt chunk %s: %w", h, err)
			}

			presignURL, objectKey, err := e.client.GetPresignURL(h)
			if err != nil {
				return fmt.Errorf("presign %s: %w", h, err)
			}

			putReq, err := http.NewRequestWithContext(ctx, "PUT", presignURL, bytes.NewReader(encrypted))
			if err != nil {
				return err
			}
			putResp, err := httpClient.Do(putReq)
			if err != nil {
				return fmt.Errorf("upload chunk %s: %w", h, err)
			}
			putResp.Body.Close()
			if putResp.StatusCode >= 400 {
				return fmt.Errorf("upload chunk %s: HTTP %d", h, putResp.StatusCode)
			}

			if err := e.client.ConfirmChunk(h, int64(len(ch.Data)), int64(len(encrypted)), objectKey); err != nil {
				return fmt.Errorf("confirm chunk %s: %w", h, err)
			}

			delete(missingSet, h)
			result.ChunksUploaded++
		}
	}
	return nil
}

func manifestEntries(files []*fileInfo) []client.FileManifestEntry {
	entries := make([]client.FileManifestEntry, 0, len(files))
	for _, fi := range files {
		relPath := absPath(fi.path)
		parentPath, name := splitPath(relPath)

		if fi.size == 0 && len(fi.chunkHexes) == 0 {
			entries = append(entries, client.FileManifestEntry{
				Path:        relPath + "/",
				ParentPath:  parentPath,
				Name:        name,
				IsDir:       true,
				ChunkHashes: []string{},
			})
			continue
		}

		mtime := fi.mtime.UTC().Format(time.RFC3339)
		entries = append(entries, client.FileManifestEntry{
			Path:        relPath,
			ParentPath:  parentPath,
			Name:        name,
			Size:        fi.size,
			Mtime:       &mtime,
			Mode:        int(fi.mode),
			IsDir:       false,
			ChunkHashes: fi.chunkHexes,
		})
	}
	return entries
}

func absPath(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func splitPath(path string) (parent, name string) {
	path = strings.TrimRight(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "/", path
	}
	return path[:idx+1], path[idx+1:]
}

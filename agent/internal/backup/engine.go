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

type fileWork struct {
	path   string
	size   int64
	mtime  time.Time
	mode   fs.FileMode
	chunks []chunker.Chunk
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

	entries, result, err := e.processPath(ctx, dek, jobID)
	if err != nil {
		return nil, err
	}

	snapshotID, err := e.client.CreateSnapshot(jobID, entries)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	result.SnapshotID = snapshotID
	return result, nil
}

func (e *Engine) processPath(ctx context.Context, dek []byte, jobID string) ([]client.FileManifestEntry, *Result, error) {
	result := &Result{}
	var entries []client.FileManifestEntry
	var files []fileWork
	allHashes := map[string]bool{}

	err := filepath.WalkDir(e.watchPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			e.log.Warn("walk", zap.String("path", path), zap.Error(walkErr))
			return nil
		}

		relPath := absPath(path)
		parentPath, name := splitPath(relPath)

		if d.IsDir() {
			entries = append(entries, client.FileManifestEntry{
				Path:        relPath + "/",
				ParentPath:  parentPath,
				Name:        name,
				IsDir:       true,
				ChunkHashes: []string{},
			})
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

		var hashes []string
		for _, ch := range chunks {
			h := hex.EncodeToString(ch.Hash[:])
			hashes = append(hashes, h)
			allHashes[h] = true
		}

		mtime := info.ModTime().UTC().Format(time.RFC3339)
		entries = append(entries, client.FileManifestEntry{
			Path:        relPath,
			ParentPath:  parentPath,
			Name:        name,
			Size:        info.Size(),
			Mtime:       &mtime,
			Mode:        int(info.Mode()),
			IsDir:       false,
			ChunkHashes: hashes,
		})
		files = append(files, fileWork{
			path:   path,
			size:   info.Size(),
			mtime:  info.ModTime(),
			mode:   info.Mode(),
			chunks: chunks,
		})
		result.FileCount++
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk: %w", err)
	}

	allHashSlice := make([]string, 0, len(allHashes))
	for h := range allHashes {
		allHashSlice = append(allHashSlice, h)
	}
	result.ChunksTotal = len(allHashSlice)

	missing, err := e.client.CheckChunks(allHashSlice)
	if err != nil {
		return nil, nil, fmt.Errorf("check chunks: %w", err)
	}
	result.ChunksDeduped = len(allHashSlice) - len(missing)

	missingSet := map[string]bool{}
	for _, h := range missing {
		missingSet[h] = true
	}

	httpClient := &http.Client{Timeout: 5 * time.Minute}

	for _, fw := range files {
		for _, ch := range fw.chunks {
			h := hex.EncodeToString(ch.Hash[:])
			if !missingSet[h] {
				continue
			}

			encrypted, err := crypto.EncryptChunk(dek, ch.Hash, ch.Data)
			if err != nil {
				return nil, nil, fmt.Errorf("encrypt chunk %s: %w", h, err)
			}

			presignURL, objectKey, err := e.client.GetPresignURL(h)
			if err != nil {
				return nil, nil, fmt.Errorf("presign %s: %w", h, err)
			}

			putReq, err := http.NewRequestWithContext(ctx, "PUT", presignURL, bytes.NewReader(encrypted))
			if err != nil {
				return nil, nil, err
			}
			putResp, err := httpClient.Do(putReq)
			if err != nil {
				return nil, nil, fmt.Errorf("upload chunk %s: %w", h, err)
			}
			putResp.Body.Close()
			if putResp.StatusCode >= 400 {
				return nil, nil, fmt.Errorf("upload chunk %s: HTTP %d", h, putResp.StatusCode)
			}

			if err := e.client.ConfirmChunk(h, int64(len(ch.Data)), int64(len(encrypted)), objectKey); err != nil {
				return nil, nil, fmt.Errorf("confirm chunk %s: %w", h, err)
			}

			result.ChunksUploaded++
		}
	}

	return entries, result, nil
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

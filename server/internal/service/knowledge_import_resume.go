package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/google/uuid"
)

const (
	importChunkExpiry   = 30 * time.Minute
	defaultMaxChunkSize = 32 << 20
)

type importChunkUpload struct {
	totalChunks int
	expectedSHA string
	chunks      map[int][]byte
	updatedAt   time.Time
}

// ImportChunkStore 内存分片上传存储。
// 单实例部署下满足断点续传；多实例/Serverless 部署时应替换为对象存储或共享 DB。
type ImportChunkStore struct {
	mu      sync.Mutex
	uploads map[string]*importChunkUpload
	maxSize int
}

func NewImportChunkStore() *ImportChunkStore {
	s := &ImportChunkStore{
		uploads: make(map[string]*importChunkUpload),
		maxSize: defaultMaxChunkSize,
	}
	go s.cleanupLoop()
	return s
}

func (s *ImportChunkStore) Init(totalChunks int, expectedSHA string) (string, error) {
	if totalChunks <= 0 || totalChunks > 10000 {
		return "", errors.New("total_chunks 必须在 1~10000 之间")
	}
	uploadID := "kbchunk_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	s.mu.Lock()
	s.uploads[uploadID] = &importChunkUpload{
		totalChunks: totalChunks,
		expectedSHA: expectedSHA,
		chunks:      make(map[int][]byte, totalChunks),
		updatedAt:   time.Now(),
	}
	s.mu.Unlock()
	return uploadID, nil
}

func (s *ImportChunkStore) Put(uploadID string, chunkIndex int, data []byte, chunkSHA string) error {
	if len(data) > s.maxSize {
		return fmt.Errorf("分片过大，单分片最大 %d 字节", s.maxSize)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	up, ok := s.uploads[uploadID]
	if !ok {
		return errors.New("upload_id 不存在或已过期")
	}
	if chunkIndex < 0 || chunkIndex >= up.totalChunks {
		return fmt.Errorf("chunk_index 越界：0~%d", up.totalChunks-1)
	}
	if chunkSHA != "" {
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), chunkSHA) {
			return errors.New("分片 sha256 校验失败")
		}
	}
	up.chunks[chunkIndex] = append([]byte(nil), data...)
	up.updatedAt = time.Now()
	return nil
}

func (s *ImportChunkStore) Status(uploadID string) (*model.KBImportChunkStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	up, ok := s.uploads[uploadID]
	if !ok {
		return nil, errors.New("upload_id 不存在或已过期")
	}
	received := make([]int, 0, len(up.chunks))
	missing := make([]int, 0)
	for i := 0; i < up.totalChunks; i++ {
		if _, ok := up.chunks[i]; ok {
			received = append(received, i)
		} else {
			missing = append(missing, i)
		}
	}
	return &model.KBImportChunkStatus{
		UploadID:       uploadID,
		TotalChunks:    up.totalChunks,
		ReceivedCount:  len(received),
		ReceivedChunks: received,
		MissingChunks:  missing,
		Complete:       len(missing) == 0,
	}, nil
}

func (s *ImportChunkStore) Assemble(uploadID string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	up, ok := s.uploads[uploadID]
	if !ok {
		return nil, "", errors.New("upload_id 不存在或已过期")
	}
	if len(up.chunks) != up.totalChunks {
		return nil, "", errors.New("分片未全部上传")
	}
	var buf bytes.Buffer
	for i := 0; i < up.totalChunks; i++ {
		buf.Write(up.chunks[i])
	}
	data := append([]byte(nil), buf.Bytes()...)
	if up.expectedSHA != "" {
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), up.expectedSHA) {
			return nil, "", errors.New("整包 sha256 校验失败")
		}
	}
	return data, up.expectedSHA, nil
}

func (s *ImportChunkStore) Remove(uploadID string) {
	s.mu.Lock()
	delete(s.uploads, uploadID)
	s.mu.Unlock()
}

func (s *ImportChunkStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, up := range s.uploads {
			if now.Sub(up.updatedAt) > importChunkExpiry {
				delete(s.uploads, id)
			}
		}
		s.mu.Unlock()
	}
}

// InitChunkUpload 初始化知识包分片上传。
func (s *KnowledgePackageService) InitChunkUpload(totalChunks int, expectedSHA string) (*model.KBImportChunkInitResponse, error) {
	uploadID, err := s.chunkStore.Init(totalChunks, expectedSHA)
	if err != nil {
		return nil, err
	}
	return &model.KBImportChunkInitResponse{
		UploadID:    uploadID,
		TotalChunks: totalChunks,
		ExpiresIn:   int(importChunkExpiry.Seconds()),
	}, nil
}

func (s *KnowledgePackageService) PutChunk(uploadID string, chunkIndex int, data []byte, chunkSHA string) error {
	return s.chunkStore.Put(uploadID, chunkIndex, data, chunkSHA)
}

func (s *KnowledgePackageService) ChunkStatus(uploadID string) (*model.KBImportChunkStatus, error) {
	return s.chunkStore.Status(uploadID)
}

// CompleteChunkUpload 汇总分片并导入标准知识包。
func (s *KnowledgePackageService) CompleteChunkUpload(ctx context.Context, uploadID, username, traceID string) (*model.KBImportPackageResponse, error) {
	data, _, err := s.chunkStore.Assemble(uploadID)
	if err != nil {
		return nil, err
	}
	resp, err := s.ImportPackage(ctx, data, username, traceID)
	if err != nil {
		return nil, err
	}
	s.chunkStore.Remove(uploadID)
	return resp, nil
}

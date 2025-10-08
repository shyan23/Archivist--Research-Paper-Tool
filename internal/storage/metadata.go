package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusCompleted  ProcessingStatus = "completed"
	StatusFailed     ProcessingStatus = "failed"
)

type ProcessingRecord struct {
	FilePath    string           `json:"file_path"`
	FileHash    string           `json:"file_hash"`
	PaperTitle  string           `json:"paper_title"`
	ProcessedAt time.Time        `json:"processed_at"`
	TexFilePath string           `json:"tex_file"`
	ReportPath  string           `json:"report_file"`
	Status      ProcessingStatus `json:"status"`
	Error       string           `json:"error,omitempty"`
}

type MetadataStore struct {
	Version         string                      `json:"version"`
	LastUpdated     time.Time                   `json:"last_updated"`
	ProcessedPapers map[string]ProcessingRecord `json:"processed_papers"`
	dbPath          string
	mu              sync.RWMutex
}

// NewMetadataStore creates or loads a metadata store
func NewMetadataStore(metadataDir string) (*MetadataStore, error) {
	dbPath := filepath.Join(metadataDir, "hashes.json")

	store := &MetadataStore{
		Version:         "1.0",
		ProcessedPapers: make(map[string]ProcessingRecord),
		dbPath:          dbPath,
	}

	// Load existing metadata if file exists
	if _, err := os.Stat(dbPath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load metadata: %w", err)
		}
	}

	return store, nil
}

// IsProcessed checks if a file hash has been successfully processed
func (ms *MetadataStore) IsProcessed(hash string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	record, exists := ms.ProcessedPapers[hash]
	return exists && record.Status == StatusCompleted
}

// GetRecord retrieves a processing record by hash
func (ms *MetadataStore) GetRecord(hash string) (ProcessingRecord, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	record, exists := ms.ProcessedPapers[hash]
	return record, exists
}

// MarkProcessing marks a file as currently being processed
func (ms *MetadataStore) MarkProcessing(hash, filePath string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.ProcessedPapers[hash] = ProcessingRecord{
		FilePath:    filePath,
		FileHash:    hash,
		ProcessedAt: time.Now(),
		Status:      StatusProcessing,
	}

	return ms.persist()
}

// MarkCompleted marks a file as successfully processed
func (ms *MetadataStore) MarkCompleted(record ProcessingRecord) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	record.ProcessedAt = time.Now()
	record.Status = StatusCompleted
	ms.ProcessedPapers[record.FileHash] = record

	return ms.persist()
}

// MarkFailed marks a file as failed
func (ms *MetadataStore) MarkFailed(hash string, errorMsg string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if record, exists := ms.ProcessedPapers[hash]; exists {
		record.Status = StatusFailed
		record.Error = errorMsg
		record.ProcessedAt = time.Now()
		ms.ProcessedPapers[hash] = record
	}

	return ms.persist()
}

// GetAllRecords returns all processing records
func (ms *MetadataStore) GetAllRecords() []ProcessingRecord {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	records := make([]ProcessingRecord, 0, len(ms.ProcessedPapers))
	for _, record := range ms.ProcessedPapers {
		records = append(records, record)
	}

	return records
}

// load reads the metadata from disk
func (ms *MetadataStore) load() error {
	data, err := os.ReadFile(ms.dbPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, ms)
}

// persist saves the metadata to disk
func (ms *MetadataStore) persist() error {
	ms.LastUpdated = time.Now()

	data, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ms.dbPath, data, 0644)
}

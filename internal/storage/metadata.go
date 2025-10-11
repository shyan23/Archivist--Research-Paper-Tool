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
	// Ensure metadata directory exists
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	dbPath := filepath.Join(metadataDir, "hashes.json")

	store := &MetadataStore{
		Version:         "1.0",
		ProcessedPapers: make(map[string]ProcessingRecord),
		dbPath:          dbPath,
	}

	// Load existing metadata if file exists
	if fileInfo, err := os.Stat(dbPath); err == nil {
		// Check if file has content
		if fileInfo.Size() > 0 {
			if err := store.load(); err != nil {
				return nil, fmt.Errorf("failed to load metadata: %w", err)
			}
		} else {
			// File exists but is empty, initialize with empty data
			if err := store.persist(); err != nil {
				return nil, fmt.Errorf("failed to initialize empty metadata: %w", err)
			}
		}
	} else if os.IsNotExist(err) {
		// File doesn't exist, create it
		if err := store.persist(); err != nil {
			return nil, fmt.Errorf("failed to create metadata file: %w", err)
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

// IsProcessedOrProcessing checks if a file is already processed or currently being processed
func (ms *MetadataStore) IsProcessedOrProcessing(hash string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	record, exists := ms.ProcessedPapers[hash]
	if !exists {
		return false
	}
	// Consider it "in use" if it's completed, processing, or even failed (to avoid retries without force flag)
	return record.Status == StatusCompleted || record.Status == StatusProcessing
}

// TryMarkProcessing atomically checks if a file can be processed and marks it as processing
// Returns true if successfully marked for processing, false if already processed/processing
func (ms *MetadataStore) TryMarkProcessing(hash, filePath string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if already exists and is completed or currently processing
	if record, exists := ms.ProcessedPapers[hash]; exists {
		if record.Status == StatusCompleted || record.Status == StatusProcessing {
			return false // Already handled
		}
	}

	// Mark as processing
	ms.ProcessedPapers[hash] = ProcessingRecord{
		FilePath:    filePath,
		FileHash:    hash,
		ProcessedAt: time.Now(),
		Status:      StatusProcessing,
	}

	ms.persist()
	return true
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

	// Check if data is empty
	if len(data) == 0 {
		return fmt.Errorf("metadata file is empty")
	}

	// Unmarshal with validation
	if err := json.Unmarshal(data, ms); err != nil {
		return fmt.Errorf("invalid JSON in metadata file: %w", err)
	}

	// Ensure map is initialized
	if ms.ProcessedPapers == nil {
		ms.ProcessedPapers = make(map[string]ProcessingRecord)
	}

	return nil
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

package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetadataStore tests creating a new metadata store
func TestNewMetadataStore(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, "1.0", store.Version)
	assert.Empty(t, store.ProcessedPapers)
	assert.Equal(t, filepath.Join(metadataDir, "hashes.json"), store.dbPath)
	
	// Check if the file was created
	_, err = os.Stat(store.dbPath)
	assert.NoError(t, err)
}

// TestNewMetadataStoreLoadExisting tests loading an existing metadata store
func TestNewMetadataStoreLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	err := os.MkdirAll(metadataDir, 0755)
	require.NoError(t, err)

	// Create an existing metadata file
	existingData := `{
    "version": "1.0",
    "last_updated": "2023-01-01T00:00:00Z",
    "processed_papers": {
        "hash1": {
            "file_path": "/path/to/paper1.pdf",
            "file_hash": "hash1",
            "paper_title": "Test Paper 1",
            "processed_at": "2023-01-01T00:00:00Z",
            "tex_file": "/path/to/tex1.tex",
            "report_file": "/path/to/report1.pdf",
            "status": "completed"
        }
    }
}`
	
	dbPath := filepath.Join(metadataDir, "hashes.json")
	err = os.WriteFile(dbPath, []byte(existingData), 0644)
	require.NoError(t, err)

	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)
	assert.NotNil(t, store)
	
	// Check if data was loaded
	record, exists := store.GetRecord("hash1")
	assert.True(t, exists)
	assert.Equal(t, "Test Paper 1", record.PaperTitle)
	assert.Equal(t, StatusCompleted, record.Status)
}

// TestIsProcessed tests the IsProcessed method
func TestIsProcessed(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	hash := "test-hash"
	
	// Initially should not be processed
	assert.False(t, store.IsProcessed(hash))
	
	// Add a completed record
	record := ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    hash,
		PaperTitle:  "Test Paper",
		ProcessedAt: time.Now(),
		TexFilePath: "/path/to/output.tex",
		ReportPath:  "/path/to/output.pdf",
		Status:      StatusCompleted,
	}
	err = store.MarkCompleted(record)
	require.NoError(t, err)
	
	// Should now be processed
	assert.True(t, store.IsProcessed(hash))
	
	// Add a failed record with the same hash
	failedRecord := ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    hash,
		PaperTitle:  "Test Paper",
		ProcessedAt: time.Now(),
		Status:      StatusFailed,
	}
	err = store.MarkCompleted(failedRecord)
	require.NoError(t, err)
	
	// Should not be processed because the last status is failed
	assert.False(t, store.IsProcessed(hash))
}

// TestGetRecord tests the GetRecord method
func TestGetRecord(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	hash := "test-hash"
	
	// Initially should not exist
	_, exists := store.GetRecord(hash)
	assert.False(t, exists)
	
	// Add a record
	record := ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    hash,
		PaperTitle:  "Test Paper",
		ProcessedAt: time.Now(),
		TexFilePath: "/path/to/output.tex",
		ReportPath:  "/path/to/output.pdf",
		Status:      StatusCompleted,
	}
	err = store.MarkCompleted(record)
	require.NoError(t, err)
	
	// Should now exist
	retrievedRecord, exists := store.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, record.FilePath, retrievedRecord.FilePath)
	assert.Equal(t, record.FileHash, retrievedRecord.FileHash)
	assert.Equal(t, record.PaperTitle, retrievedRecord.PaperTitle)
	assert.Equal(t, record.Status, retrievedRecord.Status)
}

// TestMarkProcessing tests the MarkProcessing method
func TestMarkProcessing(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	hash := "test-hash"
	filePath := "/path/to/paper.pdf"
	
	err = store.MarkProcessing(hash, filePath)
	require.NoError(t, err)
	
	record, exists := store.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, filePath, record.FilePath)
	assert.Equal(t, hash, record.FileHash)
	assert.Equal(t, StatusProcessing, record.Status)
	assert.WithinDuration(t, time.Now(), record.ProcessedAt, 5*time.Second)
}

// TestMarkCompleted tests the MarkCompleted method
func TestMarkCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	hash := "test-hash"
	record := ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    hash,
		PaperTitle:  "Test Paper",
		TexFilePath: "/path/to/output.tex",
		ReportPath:  "/path/to/output.pdf",
		Status:      StatusProcessing, // Start with processing status
	}
	
	err = store.MarkCompleted(record)
	require.NoError(t, err)
	
	retrievedRecord, exists := store.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, StatusCompleted, retrievedRecord.Status)
	assert.WithinDuration(t, time.Now(), retrievedRecord.ProcessedAt, 5*time.Second)
	// Note: The status in the saved record should be completed, not the original processing status
	assert.Equal(t, StatusCompleted, retrievedRecord.Status)
}

// TestMarkFailed tests the MarkFailed method
func TestMarkFailed(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	hash := "test-hash"

	// First, mark as processing (which creates an initial record)
	err = store.MarkProcessing(hash, "/path/to/paper.pdf")
	require.NoError(t, err)
	
	errorMsg := "processing failed due to API error"
	err = store.MarkFailed(hash, errorMsg)
	require.NoError(t, err)
	
	record, exists := store.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, StatusFailed, record.Status)
	assert.Equal(t, errorMsg, record.Error)
	assert.WithinDuration(t, time.Now(), record.ProcessedAt, 5*time.Second)
}

// TestGetAllRecords tests the GetAllRecords method
func TestGetAllRecords(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Add multiple records
	records := []ProcessingRecord{
		{
			FilePath:    "/path/to/paper1.pdf",
			FileHash:    "hash1",
			PaperTitle:  "Test Paper 1",
			ProcessedAt: time.Now(),
			TexFilePath: "/path/to/output1.tex",
			ReportPath:  "/path/to/output1.pdf",
			Status:      StatusCompleted,
		},
		{
			FilePath:    "/path/to/paper2.pdf",
			FileHash:    "hash2",
			PaperTitle:  "Test Paper 2",
			ProcessedAt: time.Now(),
			TexFilePath: "/path/to/output2.tex",
			ReportPath:  "/path/to/output2.pdf",
			Status:      StatusProcessing,
		},
		{
			FilePath:    "/path/to/paper3.pdf",
			FileHash:    "hash3",
			PaperTitle:  "Test Paper 3",
			ProcessedAt: time.Now(),
			Status:      StatusFailed,
			Error:       "API error",
		},
	}

	for _, record := range records {
		err = store.MarkCompleted(record) // This will set the status correctly
		require.NoError(t, err)
	}

	// Update the second record to Processing status
	err = store.MarkProcessing("hash2", "/path/to/paper2.pdf")
	require.NoError(t, err)

	// Update the third record to Failed status
	err = store.MarkFailed("hash3", "API error")
	require.NoError(t, err)

	allRecords := store.GetAllRecords()
	assert.Len(t, allRecords, 3)

	// Check that all records are present (order may vary)
	recordMap := make(map[string]ProcessingRecord)
	for _, r := range allRecords {
		recordMap[r.FileHash] = r
	}

	assert.Contains(t, recordMap, "hash1")
	assert.Contains(t, recordMap, "hash2")
	assert.Contains(t, recordMap, "hash3")

	assert.Equal(t, StatusCompleted, recordMap["hash1"].Status)
	assert.Equal(t, StatusProcessing, recordMap["hash2"].Status)
	assert.Equal(t, StatusFailed, recordMap["hash3"].Status)
}

// TestPersistence tests that records are persisted to disk
func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Add a record
	record := ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    "persistent-hash",
		PaperTitle:  "Persistent Paper",
		ProcessedAt: time.Now(),
		TexFilePath: "/path/to/output.tex",
		ReportPath:  "/path/to/output.pdf",
		Status:      StatusCompleted,
	}
	err = store.MarkCompleted(record)
	require.NoError(t, err)

	// Load a new store instance from the same directory
	newStore, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)
	
	// Check if the record is still there
	retrievedRecord, exists := newStore.GetRecord("persistent-hash")
	assert.True(t, exists)
	assert.Equal(t, "Persistent Paper", retrievedRecord.PaperTitle)
	assert.Equal(t, StatusCompleted, retrievedRecord.Status)
}

// TestConcurrentAccess tests concurrent access to the metadata store
func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Simulate concurrent access by calling methods multiple times
	for i := 0; i < 10; i++ {
		go func(i int) {
			hash := string(rune('a' + i))
			filePath := "/path/to/paper" + string(rune('0' + i)) + ".pdf"
			
			err := store.MarkProcessing(hash, filePath)
			assert.NoError(t, err)
			
			record := ProcessingRecord{
				FilePath:    filePath,
				FileHash:    hash,
				PaperTitle:  "Test Paper " + string(rune('0' + i)),
				ProcessedAt: time.Now(),
				TexFilePath: "/path/to/output" + string(rune('0' + i)) + ".tex",
				ReportPath:  "/path/to/output" + string(rune('0' + i)) + ".pdf",
				Status:      StatusCompleted,
			}
			err = store.MarkCompleted(record)
			assert.NoError(t, err)
		}(i)
	}

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
	
	// Check that all records were stored
	for i := 0; i < 10; i++ {
		hash := string(rune('a' + i))
		_, exists := store.GetRecord(hash)
		assert.True(t, exists, "Record %d should exist", i)
	}
}

// TestLoadInvalidJSON tests loading from an invalid JSON file
func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	err := os.MkdirAll(metadataDir, 0755)
	require.NoError(t, err)

	// Create an invalid JSON file
	dbPath := filepath.Join(metadataDir, "hashes.json")
	err = os.WriteFile(dbPath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// This should fail
	_, err = NewMetadataStore(metadataDir)
	assert.Error(t, err)
}
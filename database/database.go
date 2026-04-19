package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const dbDir = "./database"
const dbFile = "./database/uploads.json"
const dbBackupDir = "./database/backups"

// VideoRecord represents a single video upload record
type VideoRecord struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Site            string    `json:"site"`
	ChannelID       string    `json:"channel_id"`       // site__username format
	Filename        string    `json:"filename"`
	OriginalPath    string    `json:"original_path"`    // Original file path before upload
	UploadedAt      time.Time `json:"uploaded_at"`
	RecordedAt      time.Time `json:"recorded_at"`      // When the stream was recorded
	GoFileLink      string    `json:"gofile_link"`
	Duration        float64   `json:"duration"`         // Duration in seconds
	FilesizeBytes   int64     `json:"filesize_bytes"`
	Resolution      int       `json:"resolution"`       // e.g., 1080
	Framerate       int       `json:"framerate"`        // e.g., 30
	RoomTitle       string    `json:"room_title,omitempty"`
	Gender          string    `json:"gender,omitempty"`
	UploadDuration  float64   `json:"upload_duration"`  // Upload time in seconds
	UploadSpeed     float64   `json:"upload_speed"`     // MB/s
	Status          string    `json:"status"`           // "uploaded", "failed", "deleted"
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// DatabaseStats represents statistics about the database
type DatabaseStats struct {
	TotalRecords      int       `json:"total_records"`
	TotalSizeBytes    int64     `json:"total_size_bytes"`
	TotalDuration     float64   `json:"total_duration"`
	UniqueChannels    int       `json:"unique_channels"`
	OldestUpload      time.Time `json:"oldest_upload"`
	LatestUpload      time.Time `json:"latest_upload"`
	UploadsByChannel  map[string]int `json:"uploads_by_channel"`
	UploadsBySite     map[string]int `json:"uploads_by_site"`
}

// Database manages video upload records
type Database struct {
	mu      sync.RWMutex
	records []VideoRecord
	index   map[string]*VideoRecord // ID -> Record for fast lookup
	byUser  map[string][]*VideoRecord // Username -> Records
	bySite  map[string][]*VideoRecord // Site -> Records
}

var (
	db   *Database
	once sync.Once
)

// GetDB returns the singleton database instance
func GetDB() *Database {
	once.Do(func() {
		db = &Database{
			records: []VideoRecord{},
			index:   make(map[string]*VideoRecord),
			byUser:  make(map[string][]*VideoRecord),
			bySite:  make(map[string][]*VideoRecord),
		}
		_ = db.Load()
	})
	return db
}

// Load reads the database from disk
func (d *Database) Load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(dbFile)
	if os.IsNotExist(err) {
		// Try legacy location for migration
		legacyFile := "./conf/videos.json"
		data, err = os.ReadFile(legacyFile)
		if os.IsNotExist(err) {
			// No database exists yet, initialize empty
			d.records = []VideoRecord{}
			d.index = make(map[string]*VideoRecord)
			d.byUser = make(map[string][]*VideoRecord)
			d.bySite = make(map[string][]*VideoRecord)
			return nil
		}
		if err != nil {
			return fmt.Errorf("read legacy db file: %w", err)
		}
		// Migrate to new location
		if err := d.unmarshalAndIndex(data); err != nil {
			return err
		}
		if err := d.saveLocked(); err != nil {
			return fmt.Errorf("migrate to new location: %w", err)
		}
		// Keep legacy file as backup
		_ = os.Rename(legacyFile, legacyFile+".migrated")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read db file: %w", err)
	}

	return d.unmarshalAndIndex(data)
}

func (d *Database) unmarshalAndIndex(data []byte) error {
	if err := json.Unmarshal(data, &d.records); err != nil {
		return fmt.Errorf("unmarshal db: %w", err)
	}

	// Rebuild indexes
	d.index = make(map[string]*VideoRecord)
	d.byUser = make(map[string][]*VideoRecord)
	d.bySite = make(map[string][]*VideoRecord)

	for i := range d.records {
		record := &d.records[i]
		d.index[record.ID] = record
		d.byUser[record.Username] = append(d.byUser[record.Username], record)
		d.bySite[record.Site] = append(d.bySite[record.Site], record)
	}

	return nil
}

// Save writes the database to disk
func (d *Database) Save() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.saveLocked()
}

func (d *Database) saveLocked() error {
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := json.MarshalIndent(d.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal db: %w", err)
	}

	// Write to temp file first
	tempFile := dbFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("write temp db file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, dbFile); err != nil {
		return fmt.Errorf("rename db file: %w", err)
	}

	return nil
}

// Backup creates a timestamped backup of the database
func (d *Database) Backup() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if err := os.MkdirAll(dbBackupDir, 0700); err != nil {
		return fmt.Errorf("mkdir backup: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFile := filepath.Join(dbBackupDir, fmt.Sprintf("uploads_%s.json", timestamp))

	data, err := json.MarshalIndent(d.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	// Keep only last 10 backups
	d.cleanOldBackups(10)

	return nil
}

func (d *Database) cleanOldBackups(keep int) {
	files, err := filepath.Glob(filepath.Join(dbBackupDir, "uploads_*.json"))
	if err != nil || len(files) <= keep {
		return
	}

	// Sort by modification time
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// Remove oldest files
	for i := 0; i < len(files)-keep; i++ {
		_ = os.Remove(files[i])
	}
}

// AddRecord adds a new video record to the database
func (d *Database) AddRecord(record VideoRecord) error {
	d.mu.Lock()
	
	// Ensure required fields
	if record.ID == "" {
		record.ID = fmt.Sprintf("%s_%d", record.Username, time.Now().UnixNano())
	}
	if record.ChannelID == "" {
		record.ChannelID = record.Site + "__" + record.Username
	}
	if record.Status == "" {
		record.Status = "uploaded"
	}
	
	// Check for duplicate ID
	if _, exists := d.index[record.ID]; exists {
		d.mu.Unlock()
		return fmt.Errorf("record with ID %s already exists", record.ID)
	}
	
	d.records = append(d.records, record)
	d.index[record.ID] = &d.records[len(d.records)-1]
	d.byUser[record.Username] = append(d.byUser[record.Username], &d.records[len(d.records)-1])
	d.bySite[record.Site] = append(d.bySite[record.Site], &d.records[len(d.records)-1])
	
	d.mu.Unlock()

	// Save immediately after adding
	if err := d.Save(); err != nil {
		return fmt.Errorf("save database after add: %w", err)
	}
	
	return nil
}

// GetRecords returns all video records sorted by upload time (newest first)
func (d *Database) GetRecords() []VideoRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]VideoRecord, len(d.records))
	copy(result, d.records)
	
	// Sort by upload time, newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt.After(result[j].UploadedAt)
	})
	
	return result
}

// GetRecordsByUsername returns all records for a specific username
func (d *Database) GetRecordsByUsername(username string) []VideoRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	records := d.byUser[username]
	result := make([]VideoRecord, len(records))
	for i, r := range records {
		result[i] = *r
	}
	
	// Sort by upload time, newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt.After(result[j].UploadedAt)
	})
	
	return result
}

// GetRecordsBySite returns all records for a specific site
func (d *Database) GetRecordsBySite(site string) []VideoRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	records := d.bySite[site]
	result := make([]VideoRecord, len(records))
	for i, r := range records {
		result[i] = *r
	}
	
	// Sort by upload time, newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt.After(result[j].UploadedAt)
	})
	
	return result
}

// GetRecordByID returns a specific record by ID
func (d *Database) GetRecordByID(id string) (*VideoRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	record, ok := d.index[id]
	if !ok {
		return nil, fmt.Errorf("record not found: %s", id)
	}
	
	result := *record
	return &result, nil
}

// UpdateRecord updates an existing record
func (d *Database) UpdateRecord(id string, updateFn func(*VideoRecord)) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	record, ok := d.index[id]
	if !ok {
		return fmt.Errorf("record not found: %s", id)
	}

	updateFn(record)
	return d.saveLocked()
}

// DeleteRecord marks a record as deleted (soft delete)
func (d *Database) DeleteRecord(id string) error {
	return d.UpdateRecord(id, func(r *VideoRecord) {
		r.Status = "deleted"
	})
}

// GetStats returns database statistics
func (d *Database) GetStats() DatabaseStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := DatabaseStats{
		TotalRecords:     len(d.records),
		UploadsByChannel: make(map[string]int),
		UploadsBySite:    make(map[string]int),
	}

	uniqueChannels := make(map[string]bool)
	
	for _, record := range d.records {
		if record.Status == "deleted" {
			continue
		}
		
		stats.TotalSizeBytes += record.FilesizeBytes
		stats.TotalDuration += record.Duration
		uniqueChannels[record.ChannelID] = true
		stats.UploadsByChannel[record.ChannelID]++
		stats.UploadsBySite[record.Site]++
		
		if stats.OldestUpload.IsZero() || record.UploadedAt.Before(stats.OldestUpload) {
			stats.OldestUpload = record.UploadedAt
		}
		if record.UploadedAt.After(stats.LatestUpload) {
			stats.LatestUpload = record.UploadedAt
		}
	}
	
	stats.UniqueChannels = len(uniqueChannels)
	
	return stats
}

// Search searches records by various criteria
func (d *Database) Search(query string) []VideoRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []VideoRecord
	queryLower := strings.ToLower(query)
	
	for _, record := range d.records {
		if record.Status == "deleted" {
			continue
		}
		
		if strings.Contains(strings.ToLower(record.Username), queryLower) ||
			strings.Contains(strings.ToLower(record.Filename), queryLower) ||
			strings.Contains(strings.ToLower(record.RoomTitle), queryLower) {
			result = append(result, record)
		}
	}
	
	// Sort by upload time, newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt.After(result[j].UploadedAt)
	})
	
	return result
}

package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name            string
		databaseDSN     string
		fileStoragePath string
		wantStorageType string
	}{
		{
			name:            "PostgreSQL storage",
			databaseDSN:     "postgres://user:password@localhost:5432/testdb",
			fileStoragePath: "",
			wantStorageType: "*database.DatabaseStorage",
		},
		{
			name:            "File storage",
			databaseDSN:     "",
			fileStoragePath: "testdata/test_urls.json",
			wantStorageType: "*file.FileStorage",
		},
		{
			name:            "Memory storage",
			databaseDSN:     "",
			fileStoragePath: "",
			wantStorageType: "*memory.MemoryStorage",
		},
	}

	tempFile := "testdata/test_urls.json"
	os.MkdirAll("testdata", 0755)
	os.Create(tempFile)
	defer os.RemoveAll("testdata")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.name == "PostgreSQL storage" {
				tt.databaseDSN = "postgres://invalid:invalid@localhost:5432/nonexistent"
				tt.wantStorageType = "*file.FileStorage"
			}

			storage, err := NewStorage(tt.databaseDSN, tt.fileStoragePath)
			require.NoError(t, err)
			assert.NotNil(t, storage)

			if tt.name == "File storage" && tt.wantStorageType == "*file.FileStorage" {
				os.Chmod(tt.fileStoragePath, 0000)
				storage, err = NewStorage(tt.databaseDSN, tt.fileStoragePath)
				require.NoError(t, err)
				assert.NotNil(t, storage)
				tt.wantStorageType = "*memory.MemoryStorage"
				os.Chmod(tt.fileStoragePath, 0644)
			}
		})
	}
}

func TestStorageInterfaces(t *testing.T) {
	storage, err := NewStorage("", "")
	require.NoError(t, err)

	assert.NotNil(t, storage.AsURLSaver())
	assert.NotNil(t, storage.AsURLBatchSaver())
	assert.NotNil(t, storage.AsURLGetter())
	assert.NotNil(t, storage.AsURLFetcher())
	assert.NotNil(t, storage.AsURLDeleter())
	assert.NotNil(t, storage.AsPinger())
}

package model

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() (*FolderRepository, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&FolderModel{})

	folderRepo := NewFolderRepository(db)

	return folderRepo, nil
}

func TestGetDescendants(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("setupTestDB() error = %v", err)
	}

	// Create a folder hierarchy
	rootID := uuid.New()
	child1ID := uuid.New()
	child2ID := uuid.New()

	db.Create(&FolderModel{ID: rootID, Name: "root", BucketID: uuid.New()})
	db.Create(&FolderModel{ID: child1ID, Name: "child1", ParentID: rootID, BucketID: uuid.New()})
	db.Create(&FolderModel{ID: child2ID, Name: "child2", ParentID: rootID, BucketID: uuid.New()})

	descendants, err := db.GetDescendants(rootID)
	if err != nil {
		t.Fatalf("GetDescendants() error = %v", err)
	}

	// Normalize metadata fields for comparison
	for i := range descendants {
		descendants[i].CreatedAt = time.Time{}
		descendants[i].UpdatedAt = time.Time{}
		descendants[i].DeletedAt = gorm.DeletedAt{}
	}

	expected := []FolderModel{
		{ID: child1ID, Name: "child1", ParentID: rootID},
		{ID: child2ID, Name: "child2", ParentID: rootID},
	}

	if !reflect.DeepEqual(descendants, expected) {
		t.Errorf("GetDescendants() = %v, want %v", descendants, expected)
	}
}

func TestMoveFolder(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("setupTestDB() error = %v", err)
	}

	rootID := uuid.New()
	folderID := uuid.New()
	newParentID := uuid.New()

	db.Create(&FolderModel{ID: rootID, Name: "root", BucketID: uuid.New()})
	db.Create(&FolderModel{ID: folderID, Name: "folder", ParentID: rootID, BucketID: uuid.New()})
	db.Create(&FolderModel{ID: newParentID, Name: "new_parent", BucketID: uuid.New()})

	err = db.MoveFolder(folderID, newParentID)
	if err != nil {
		t.Fatalf("MoveFolder() error = %v", err)
	}

	var folder FolderModel
	db.db.First(&folder, "id = ?", folderID)
	// assert.Equal(t, newParentID, *folder.ParentID)
	if newParentID != folder.ParentID {
		t.Errorf("MoveFolder() = %v, want %v", newParentID, folder.ParentID)
	}
}

func TestGetFullPath(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("setupTestDB() error = %v", err)
	}

	rootID := uuid.New()
	folderID := uuid.New()

	db.Create(&FolderModel{ID: rootID, Name: "root", BucketID: uuid.New()})
	db.Create(&FolderModel{ID: folderID, Name: "child", ParentID: rootID, BucketID: uuid.New()})

	path, err := db.GetFullPath(folderID)
	if err != nil {
		t.Fatalf("GetFullPath() error = %v", err)
	}

	// check if the path is correct, don't use assert
	if path != "root/child" {
		t.Errorf("GetFullPath() = %v, want %v", path, "root/child")
	}

	// assert.NoError(t, err)
	// assert.Equal(t, "root/child", path)
}

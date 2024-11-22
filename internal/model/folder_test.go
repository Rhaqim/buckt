package model

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() (*FolderRepository, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&FolderModel{}, &FileModel{}, &BucketModel{})

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
func TestGetFilesFromPath(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("setupTestDB() error = %v", err)
	}

	// Create a folder hierarchy
	rootID := uuid.New()
	childID := uuid.New()
	fileID := uuid.New()
	bucketID := uuid.New()

	db.db.Create(&BucketModel{ID: bucketID, Name: "bucket"})
	db.Create(&FolderModel{ID: rootID, Name: "root", BucketID: bucketID})
	db.Create(&FolderModel{ID: childID, Name: "child", ParentID: rootID, BucketID: bucketID})
	db.db.Create(&FileModel{ID: fileID, Name: "file.txt", ParentID: childID, BucketID: bucketID})

	files, err := db.GetFilesFromPath("bucket", "root/child")
	if err != nil {
		t.Fatalf("GetFilesFromPath() error = %v", err)
	}

	expected := []FileModel{
		{ID: fileID, Name: "file.txt", ParentID: childID},
	}

	if !reflect.DeepEqual(files, expected) {
		t.Errorf("GetFilesFromPath() = %v, want %v", files, expected)
	}
}
func TestGetSubfolders(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("setupTestDB() error = %v", err)
	}

	// Create a folder hierarchy
	bucketID := uuid.New()
	rootID := uuid.New()
	child1ID := uuid.New()
	child2ID := uuid.New()
	subChild1ID := uuid.New()

	db.db.Create(&BucketModel{ID: bucketID, Name: "bucket"})

	db.Create(&FolderModel{ID: rootID, Name: "root", BucketID: bucketID})
	db.Create(&FolderModel{ID: child1ID, Name: "child1", ParentID: rootID, BucketID: bucketID})
	db.Create(&FolderModel{ID: child2ID, Name: "child2", ParentID: rootID, BucketID: bucketID})
	db.Create(&FolderModel{ID: subChild1ID, Name: "subchild1", ParentID: child1ID, BucketID: bucketID})

	subfolders, err := db.GetSubfolders("bucket", "root/child1")
	if err != nil {
		t.Fatalf("GetSubfolders() error = %v", err)
	}

	expected := []FolderModel{
		{ID: subChild1ID, Name: "subchild1", ParentID: child1ID},
	}

	if !reflect.DeepEqual(subfolders, expected) {
		t.Errorf("GetSubfolders() = %v, want %v", subfolders, expected)
	}
}

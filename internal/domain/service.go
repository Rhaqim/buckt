package domain

import (
	"io"

	"github.com/Rhaqim/buckt/internal/model"
)

// BucktFileSystemService defines the interface for file system operations within the Buckt domain.
// It provides methods to validate paths, write, retrieve, update, and delete files.
type FileSystemService interface {
	// FSValidatePath validates the given file path and returns the validated path or an error.
	FSValidatePath(path string) (string, error)

	// FSWriteFile writes the given file data to the specified path.
	// Returns an error if the operation fails.
	FSWriteFile(path string, file []byte) error

	// FSGetFile retrieves the file data from the specified path.
	// Returns the file data or an error if the operation fails.
	FSGetFile(path string) ([]byte, error)

	// FSGetFileStream retrieves the file data from the specified path.
	// Returns an io.ReadCloser or an error if the operation fails.
	FSGetFileStream(path string) (io.ReadCloser, error)

	// FSUpdateFile updates the file from the old path to the new path.
	// Returns an error if the operation fails.
	FSUpdateFile(oldPath, newPath string) error

	// FSDeleteFile deletes the file or folder at the specified path.
	// Returns an error if the operation fails.
	FSDeleteFile(folderPath string) error

	// FSDeleteFolder deletes the folder at the specified path.
	// Returns an error if the operation fails.
	FSDeleteFolder(folderPath string) error
}

type BucktFileSystemServiceMock struct {
	FSValidatePathFunc func(path string) (string, error)
	FSWriteFileFunc    func(path string, file []byte) error
	FSGetFileFunc      func(path string) ([]byte, error)
	FSUpdateFileFunc   func(oldPath, newPath string) error
	FSDeleteFileFunc   func(folderPath string) error
}

type FolderService interface {
	CreateFolder(user_id, parent_id, folder_name, description string) (string, error)
	GetRootFolder(user_id string) (*model.FolderModel, error)
	GetFolder(user_id, folder_id string) (*model.FolderModel, error)
	GetFolders(parent_id string) ([]model.FolderModel, error)
	MoveFolder(folder_id, new_parent_id string) error
	RenameFolder(user_id, folder_id, new_name string) error
	DeleteFolder(folder_id string) (string, error)
	ScrubFolder(user_id, folder_id string) (string, error)
}

type FileService interface {
	CreateFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error)
	GetFile(file_id string) (*model.FileModel, error)
	GetFileStream(file_id string) (*model.FileModel, io.ReadCloser, error)
	GetFiles(parent_id string) ([]model.FileModel, error)
	MoveFile(file_id, new_parent_id string) error
	RenameFile(file_id, new_name string) error
	UpdateFile(user_id, file_id, new_file_name string, new_file_data []byte) error
	DeleteFile(file_id string) (string, error)
	ScrubFile(file_id string) (string, error)
}

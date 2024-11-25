package domain

type BucktFileSystemService interface {
	FSValidatePath(path string) (string, error)
	FSWriteFile(path string, file []byte) error
	FSGetFile(path string) ([]byte, error)
	FSUpdateFile(oldPath, newPath string) error
	FSDeleteFile(folderPath string) error
}

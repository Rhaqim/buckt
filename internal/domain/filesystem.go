package domain

type BucktFileSystemService interface {
	FSGetFile(path string) ([]byte, error)
	FSWriteFile(path string, file []byte) error
	FSUpdateFile(oldPath, newPath string) error
	FSDeleteFile(folderPath string) error
	FSValidatePath(path string) (string, error)
}

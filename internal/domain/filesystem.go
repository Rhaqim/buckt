package domain

type BucktFileSystemService interface {
	FSWriteFile(path string, file []byte) error
	FSUpdateFile(oldPath, newPath string) error
	FSDeleteFile(folderPath string) error
}

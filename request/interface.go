package request

type BaseFileRequest struct {
	BucketName string `json:"bucket_name"`
	FolderPath string `json:"folder_path"`
}

type FileRequest struct {
	BaseFileRequest
	Filename string `json:"filename"`
}

type RenameFileRequest struct {
	BaseFileRequest
	Filename    string `json:"filename"`
	NewFilename string `json:"new_filename"`
}

type MoveFileRequest struct {
	OldFolderPath string `json:"old_folder_path"`
	NewFolderPath string `json:"new_folder_path"`
}

type RenameFolderRequest struct {
	BaseFileRequest
	NewFolderName string `json:"new_folder_name"`
}

type MoveFolderRequest struct {
	BaseFileRequest
	NewFolderPath string `json:"new_folder_path"`
}

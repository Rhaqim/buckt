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
	FileRequest
	NewFilename string `json:"new_filename"`
}

type MoveFileRequest struct {
	FileRequest
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

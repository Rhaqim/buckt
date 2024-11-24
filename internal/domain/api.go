package domain

import (
	"github.com/gin-gonic/gin"
)

// Upload media - requires a bucket name used as the top level folder, folder path for subsequent folders and the file to be uploaded
// e.g /upload data = {bucket_name: "bucket_name", folder_path: "folder_path", file: "file"}

// Fetch sub folders - requires a bucket name used as the top level folder and the folder path for subsequent folders. If the bucket name alone is provided, the top level folders will be returned
// if the folder path is provided, the sub folders in that folder will be returned
// e.g /fetch/folders data = {bucket_name: "bucket_name", folder_path: "folder_path"}

// Rename folder - requires a bucket name and the folder path to the folder to be renamed
// e.g /rename/folder data = {bucket_name: "bucket_name", folder_path: "folder_path", new_folder_name: "new_folder_name"}

// Move folder - requires a bucket name and the folder path to the folder to be moved
// e.g /move/folder data = {bucket_name: "bucket_name", folder_path: "folder_path", new_folder_path: "new_folder_path"}

// Fetch files in folder - requires a bucket name and the full folder path to the folder containing the files must be provided.
// e.g /fetch/files data = {bucket_name: "bucket_name", folder_path: "folder_path"}

// Fetch files - requires a bucket name and the folder path to the file i.e bukcet_name/folder_path/filename
// e.g /fetch/files/filename data = {bucket_name: "bucket_name", folder_path: "folder_path", filename: "filename"}

// Download file - requires a bucket name and the folder path to the file i.e bukcet_name/folder_path/filename
// e.g /download/filename data = {bucket_name: "bucket_name", folder_path: "folder_path", filename: "filename"}

// Delete file - requires a bucket name and the folder path to the file i.e bukcet_name/folder_path/filename
// e.g /delete/filename data = {bucket_name: "bucket_name", folder_path: "folder_path", filename: "filename"}

// Move file - requires a bucket name and the folder path to the file i.e bukcet_name/folder_path/filename
// e.g /move/file data = {bucket_name: "bucket_name", folder_path: "folder_path", new_folder_path: "new_folder_path", filename: "filename"}

type APIHTTPService interface {
	NewUser(*gin.Context)
	NewBucket(*gin.Context)

	// File operations
	FileUpload(*gin.Context)
	FileDownload(*gin.Context)
	FileRename(*gin.Context)
	FileMove(*gin.Context)
	FileServe(*gin.Context)
	FileDelete(*gin.Context)

	// Folder operations
	FolderFiles(*gin.Context)
	FolderSubFolders(*gin.Context)
	FolderRename(*gin.Context)
	FolderMove(*gin.Context)
	FolderDelete(*gin.Context)
}

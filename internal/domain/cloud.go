package domain

type CloudService interface {
	UploadFileToCloud(file_id string) error
	UploadFolderToCloud(user_id, folder_id string) error
}

package domain

type CloudService interface {
	UploadFile(file_id string) error
	UploadFolder(user_id, folder_id string) error
}

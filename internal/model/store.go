package model

import "github.com/Rhaqim/buckt/internal/domain"

type BucktStore struct {
	OwnerStore  domain.BucktRepository[OwnerModel]
	BucketStore domain.BucktRepository[BucketModel]
	FolderStore domain.BucktRepository[FolderModel]
	FileStore   domain.BucktRepository[FileModel]
	TagStore    domain.BucktRepository[TagModel]
}

package model

import "github.com/Rhaqim/buckt/internal/domain_old"

type BucktStore struct {
	OwnerStore  domain_old.BucktRepository[OwnerModel]
	BucketStore domain_old.BucktRepository[BucketModel]
	FolderStore domain_old.BucktRepository[FolderModel]
	FileStore   domain_old.BucktRepository[FileModel]
	TagStore    domain_old.BucktRepository[TagModel]
}

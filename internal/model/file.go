package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"gorm.io/gorm"
)

type FileModel struct {
	gorm.Model
	Name string
	Path string
}

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) domain.Repository[FileModel] {
	return &FileRepository{db}
}

func (r *FileRepository) Create(file *FileModel) error {
	return r.db.Create(file).Error
}

func (r *FileRepository) FindAll() ([]FileModel, error) {
	var files []FileModel
	err := r.db.Find(&files).Error
	return files, err
}

func (r *FileRepository) FindByID(id uint) (FileModel, error) {
	var file FileModel
	err := r.db.First(&file, id).Error
	return file, err
}

func (r *FileRepository) Delete(id uint) error {
	return r.db.Delete(&FileModel{}, id).Error
}

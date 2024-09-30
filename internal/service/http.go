package service

import (
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/gin-gonic/gin"
)

type httpService struct {
	domain.StorageFileService
}

func NewHTTPService(s domain.StorageFileService) domain.StorageHTTPService {
	return &httpService{s}
}

func (s *httpService) Download(c *gin.Context) {
	filename := c.Param("filename")
	file, err := s.DownloadFile(filename)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/octet-stream", file)
}

func (s *httpService) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	err = s.UploadFile(data, file.Filename)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "File uploaded successfully"})
}

func (s *httpService) Delete(c *gin.Context) {
	filename := c.Param("filename")
	err := s.DeleteFile(filename)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "File deleted successfully"})
}

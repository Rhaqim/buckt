package service

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/gin-gonic/gin"
)

type httpService struct {
	domain.StorageFileService
}

func NewHTTPService(s domain.StorageFileService) domain.StorageHTTPService {
	return &httpService{s}
}

func (s *httpService) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	bucketname := c.PostForm("bucketname")
	folderPath := c.PostForm("folderPath")

	err = s.UploadFile(file, bucketname, folderPath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	path, err := s.Serve(file.Filename, true)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "File uploaded successfully",
		"path":    path,
	})
}

func (s *httpService) Download(c *gin.Context) {
	filename := c.Param("filename")

	file, err := s.DownloadFile(filename)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "application/octet-stream", file)
}

func (s *httpService) ServeFile(c *gin.Context) {
	filename := c.Param("filename")

	path, err := s.Serve(filename, false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.File(path)
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

func (s *httpService) NewUser(c *gin.Context) {

	req := struct {
		Name  string
		Email string
	}{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err = s.CreateOwner(req.Name, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "User created successfully"})
}

func (s *httpService) NewBucket(c *gin.Context) {

	req := struct {
		Name        string
		Description string
		OwnerID     string `json:"owner_id"`
	}{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err = s.CreateBucket(req.Name, req.Description, req.OwnerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Bucket created successfully"})
}

func (s *httpService) FetchFiles(c *gin.Context) {
	bucketname := c.Param("bucketname")

	files, err := s.GetFiles(bucketname)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"files": files})
}

package service

import (
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/gin-gonic/gin"
)

type portalService struct {
	domain.StorageFileService
}

func NewPortalService(s domain.StorageFileService) domain.StorageHTTPService {
	return &portalService{s}
}

func (s *portalService) FetchFiles(c *gin.Context) {
	// bucketName := c.Param("bucket_name")

	// files, err := s.GetFiles(bucketName)
	// if err != nil {
	// 	c.JSON(500, gin.H{"error": "Failed to retrieve files"})
	// 	return
	// }

	files := []string{"file1.txt", "file2.txt", "file3.txt"} // Replace with actual files

	// Render the partial template to update only the file list section
	c.HTML(200, "partials/files.html", gin.H{
		"Files": files,
	})
}

func (s *portalService) NewUser(c *gin.Context) {
	var req struct {
		Name  string `form:"name"`
		Email string `form:"email"`
	}

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.CreateOwner(req.Name, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "User created successfully"})
}

func (s *portalService) NewBucket(c *gin.Context) {
	var req struct {
		Name        string `form:"bucket_name"`
		Description string `form:"description"`
		OwnerID     string // Example owner ID, adjust as needed
	}

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.CreateBucket(req.Name, req.Description, "example_owner_id") // Replace with appropriate owner ID
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Bucket created successfully"})
}

func (s *portalService) Upload(c *gin.Context) {
	bucketName := c.PostForm("bucket_name")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to retrieve file"})
		return
	}

	fileData, err := file.Open()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to open file"})
		return
	}
	defer fileData.Close()

	fileBytes, err := io.ReadAll(fileData)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read file"})
		return
	}

	err = s.UploadFile(fileBytes, bucketName, file.Filename)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to upload file"})
		return
	}

	c.JSON(200, gin.H{"message": "File uploaded successfully"})
}

func (s *portalService) Download(c *gin.Context) {
	filename := c.Param("filename")
	fileData, err := s.DownloadFile(filename)
	if err != nil {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "application/octet-stream", fileData)
}

func (s *portalService) Delete(c *gin.Context) {
	filename := c.Param("filename")
	err := s.DeleteFile(filename)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(200, gin.H{"message": "File deleted successfully"})
}

func (s *portalService) ServeFile(c *gin.Context) {
	filename := c.Param("filename")
	fileData, err := s.DownloadFile(filename)
	if err != nil {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	c.Data(200, "application/octet-stream", fileData)
}

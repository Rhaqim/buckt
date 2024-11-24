package service

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/request"
	"github.com/gin-gonic/gin"
)

type APIService struct {
	domain.BucktService
}

func NewAPIService(s domain.BucktService) domain.APIHTTPService {
	return &APIService{s}
}

func (s *APIService) NewUser(c *gin.Context) {
	clientType, _ := c.Get("clientType")

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

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "User created successfully"})
	default:
		c.JSON(200, gin.H{"message": "User created successfully"})
	}
}

func (s *APIService) NewBucket(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req struct {
		Name        string `form:"bucket_name"`
		Description string `form:"description"`
		OwnerID     string // Example owner ID, adjust as needed
	}

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.CreateBucket(req.Name, req.Description, req.OwnerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "Bucket created successfully"})
	default:
		c.JSON(200, gin.H{"message": "Bucket created successfully"})
	}
}

func (s *APIService) FileUpload(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	bucketName := c.PostForm("bucket_name")
	folderPath := c.PostForm("folder_path")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to retrieve file", "message": err.Error()})
		return
	}

	err = s.UploadFile(file, bucketName, folderPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to upload file", "message": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "File uploaded successfully"})
	default:
		c.JSON(200, gin.H{"message": "File uploaded successfully"})
	}
}

func (s *APIService) FileDownload(c *gin.Context) {

	var req request.FileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	fileData, err := s.DownloadFile(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+req.Filename)
	c.Data(200, "application/octet-stream", fileData)

}

func (s *APIService) FileRename(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.RenameFileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.RenameFile(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "File renamed successfully"})
	default:
		c.JSON(200, gin.H{"message": "File renamed successfully"})
	}
}

func (s *APIService) FileMove(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.MoveFileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.MoveFile(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "File moved successfully"})
	default:
		c.JSON(200, gin.H{"message": "File moved successfully"})
	}
}

func (s *APIService) FileServe(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.FileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	fileData, err := s.ServeFile(req, true)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "File served successfully"})
	default:
		c.File(fileData)
	}
}

func (s *APIService) FileDelete(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.FileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.DeleteFile(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"message": "File deleted successfully"})
	default:
		c.JSON(200, gin.H{"message": "File deleted successfully"})
	}
}

func (s *APIService) FolderFiles(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.BaseFileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	files, err := s.GetFilesInFolder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/files.html", gin.H{"files": files})
	default:
		c.JSON(200, gin.H{"files": files})
	}
}

func (s *APIService) FolderSubFolders(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.BaseFileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	folders, err := s.GetSubFolders(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/folders.html", gin.H{"folders": folders})
	default:
		c.JSON(200, gin.H{"folders": folders})
	}
}

func (s *APIService) FolderRename(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.RenameFolderRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.RenameFolder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/folders.html", gin.H{"message": "Folder renamed successfully"})
	default:
		c.JSON(200, gin.H{"message": "Folder renamed successfully"})
	}
}

func (s *APIService) FolderMove(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.MoveFolderRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.MoveFolder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/folders.html", gin.H{"message": "Folder moved successfully"})
	default:
		c.JSON(200, gin.H{"message": "Folder moved successfully"})
	}
}

func (s *APIService) FolderDelete(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.BaseFileRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := s.DeleteFolder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/folders.html", gin.H{"message": "Folder deleted successfully"})
	default:
		c.JSON(200, gin.H{"message": "Folder deleted successfully"})
	}
}

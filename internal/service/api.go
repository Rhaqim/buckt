package service

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/request"
	"github.com/gin-gonic/gin"
)

type APIService struct {
	domain.BucktService
}

func NewAPIService(s domain.BucktService) domain.APIHTTPService {
	return &APIService{s}
}

func (s *APIService) Dashboard(c *gin.Context) {
	// bucketNmae := c.Query("bucket")
	bucketNmae := "test_bucket"

	var req request.BaseFileRequest = request.BaseFileRequest{
		BucketName: bucketNmae,
	}

	folders, err := s.GetSubFolders(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	folderPath := "/" + bucketNmae

	// Render the dashboard page with the files
	c.HTML(200, "dashboard.html", gin.H{
		"Title":   "Dashboard",
		"Folders": folders,
		"Path":    folderPath,
	})
}

func (s *APIService) NewUser(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.BindJSON(&req); err != nil {
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
		Name        string `json:"bucket_name"`
		Description string `json:"description"`
		OwnerID     string `json:"owner_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("Request: ", req)

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

func (s *APIService) AllBuckets(c *gin.Context) {
	owner_, ok := c.Get("owner")
	if !ok {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	clientType, _ := c.Get("clientType")

	buckets, err := s.GetBuckets(owner_.(model.OwnerModel).ID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/buckets.html", gin.H{"buckets": buckets})
	default:
		c.JSON(200, gin.H{"buckets": buckets})
	}
}

func (s *APIService) ViewBucket(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	bucketName := c.Query("bucket")
	var req request.BaseFileRequest
	req.BucketName = bucketName

	bucket, err := s.GetSubFolders(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/buckets.html", gin.H{"bucket": bucket})
	default:
		c.JSON(200, gin.H{"bucket": bucket})
	}
}

func (s *APIService) RemoveBucket(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	bucketName := c.Query("bucket")

	err := s.DeleteBucket(bucketName)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	switch clientType {
	case "portal":
		c.HTML(200, "partials/buckets.html", gin.H{"message": "Bucket deleted successfully"})
	default:
		c.JSON(200, gin.H{"message": "Bucket deleted successfully"})
	}
}

func (s *APIService) FileUpload(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	bucketName := c.Param("bucket")
	folderPath := c.PostForm("folder")

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
	bucketName := c.Param("bucket")

	var req request.FileRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.BucketName = bucketName

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

	bucketName := c.Param("bucket")

	var req request.RenameFileRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.BucketName = bucketName

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

	if err := c.BindJSON(&req); err != nil {
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
	filepath := c.Query("filepath")

	fileData, err := s.ServeFile(filepath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.File(fileData)
}

func (s *APIService) FileDelete(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	bucketName := c.Param("bucket")

	var req request.FileRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.BucketName = bucketName

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

func (s *APIService) FolderContent(c *gin.Context) {
	var req request.BaseFileRequest = request.BaseFileRequest{
		FolderPath: c.Query("folder"),
	}

	files, folders, err := s.GetFolderContent(req.FolderPath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Update folder path
	folderPath := "/" + req.FolderPath

	// Render with updated folder context
	c.HTML(200, "dashboard.html", gin.H{
		"Files":   files,
		"Folders": folders,
		"Path":    folderPath,
	})
}

func (s *APIService) FolderFiles(c *gin.Context) {
	// clientType, _ := c.Get("clientType")

	var req request.BaseFileRequest = request.BaseFileRequest{
		BucketName: c.Param("bucket"),
		FolderPath: c.Query("folder"),
	}

	files, err := s.GetFilesInFolder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// switch clientType {
	// case "portal":
	// 	c.HTML(200, "partials/files.html", gin.H{"files": files})
	// default:
	// 	c.JSON(200, gin.H{"files": files})
	// }
	c.JSON(200, gin.H{"files": files})
}

func (s *APIService) FolderSubFolders(c *gin.Context) {
	// clientType, _ := c.Get("clientType")

	var req request.BaseFileRequest = request.BaseFileRequest{
		BucketName: c.Param("bucket"),
		FolderPath: c.Query("folder"),
	}

	folders, err := s.GetSubFolders(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// switch clientType {
	// case "portal":
	// 	c.HTML(200, "partials/folders.html", gin.H{"folders": folders})
	// default:
	// 	c.JSON(200, gin.H{"folders": folders})
	// }
	c.JSON(200, gin.H{"folders": folders})
}

func (s *APIService) FolderRename(c *gin.Context) {
	clientType, _ := c.Get("clientType")

	var req request.RenameFolderRequest

	if err := c.BindJSON(&req); err != nil {
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

	if err := c.BindJSON(&req); err != nil {
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

	if err := c.BindJSON(&req); err != nil {
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

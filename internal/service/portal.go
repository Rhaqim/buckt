package service

// type portalService struct {
// 	domain.StorageFileService
// }

// func NewPortalService(s domain.StorageFileService) domain.StorageHTTPService {
// 	return &portalService{s}
// }

// func (s *portalService) FetchFiles(c *gin.Context) {
// 	bucketName := c.Param("bucket_name")

// 	files, err := s.GetFiles(bucketName)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": "Failed to retrieve files"})
// 		return
// 	}

// 	// Render the partial template to update only the file list section
// 	c.HTML(200, "partials/files.html", gin.H{
// 		"Files": files,
// 	})
// }

// func (s *portalService) NewUser(c *gin.Context) {
// 	var req struct {
// 		Name  string `form:"name"`
// 		Email string `form:"email"`
// 	}

// 	if err := c.Bind(&req); err != nil {
// 		c.JSON(400, gin.H{"error": err.Error()})
// 		return
// 	}

// 	err := s.CreateOwner(req.Name, req.Email)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(200, gin.H{"message": "User created successfully"})
// }

// func (s *portalService) NewBucket(c *gin.Context) {
// 	var req struct {
// 		Name        string `form:"bucket_name"`
// 		Description string `form:"description"`
// 		OwnerID     string // Example owner ID, adjust as needed
// 	}

// 	if err := c.Bind(&req); err != nil {
// 		c.JSON(400, gin.H{"error": err.Error()})
// 		return
// 	}

// 	err := s.CreateBucket(req.Name, req.Description, "example_owner_id") // Replace with appropriate owner ID
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(200, gin.H{"message": "Bucket created successfully"})
// }

// func (s *portalService) Upload(c *gin.Context) {
// 	bucketName := c.PostForm("bucket_name")
// 	folderPath := c.PostForm("folder_path")

// 	file, err := c.FormFile("file")
// 	if err != nil {
// 		c.JSON(400, gin.H{"error": "Failed to retrieve file", "message": err.Error()})
// 		return
// 	}

// 	err = s.UploadFile(file, bucketName, folderPath)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": "Failed to upload file", "message": err.Error()})
// 		return
// 	}

// 	c.JSON(200, gin.H{"message": "File uploaded successfully"})
// }

// func (s *portalService) Download(c *gin.Context) {
// 	filename := c.Param("filename")
// 	fileData, err := s.DownloadFile(filename)
// 	if err != nil {
// 		c.JSON(404, gin.H{"error": "File not found"})
// 		return
// 	}

// 	c.Header("Content-Disposition", "attachment; filename="+filename)
// 	c.Data(200, "application/octet-stream", fileData)
// }

// func (s *portalService) Delete(c *gin.Context) {
// 	filename := c.Param("filename")
// 	err := s.DeleteFile(filename)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": "Failed to delete file"})
// 		return
// 	}

// 	c.JSON(200, gin.H{"message": "File deleted successfully"})
// }

// func (s *portalService) ServeFile(c *gin.Context) {
// 	filename := c.Param("filename")
// 	fileData, err := s.DownloadFile(filename)
// 	if err != nil {
// 		c.JSON(404, gin.H{"error": "File not found"})
// 		return
// 	}

// 	c.Data(200, "application/octet-stream", fileData)
// }

// func (s *portalService) FetchFilesInFolder(c *gin.Context) {
// 	bucketName := c.Query("bucket_name")
// 	folderPath := c.Param("folder_path")

// 	files, err := s.GetFilesInFolder(bucketName, folderPath)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": "Failed to retrieve files"})
// 		return
// 	}

// 	// Render the partial template to update only the file list section
// 	c.HTML(200, "partials/files.html", gin.H{
// 		"Files": files,
// 	})
// }

// func (s *portalService) FetchSubFolders(c *gin.Context) {
// 	bucketName := c.Query("bucket_name")
// 	folderPath := c.Param("folder_path")

// 	subfolders, err := s.GetSubFolders(bucketName, folderPath)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": "Failed to retrieve subfolders"})
// 		return
// 	}

// 	// Render the partial template to update only the folder list section
// 	c.HTML(200, "partials/folders.html", gin.H{
// 		"Subfolders": subfolders,
// 	})
// }

package app

import (
	"fmt"
	"io"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web/domain"
	"github.com/Rhaqim/buckt/pkg/fileutil"
	"github.com/Rhaqim/buckt/pkg/metrics"
	"github.com/Rhaqim/buckt/pkg/response"
	"github.com/gin-gonic/gin"
)

type APIService struct {
	client *buckt.Client
}

func NewAPIService(client *buckt.Client) domain.APIService {
	return &APIService{
		client: client,
	}
}

// Metrics reports what buckt gained in 1.6.0: backend operation counts (with
// Cloudflare R2 billing class), total storage bytes, and cache hit/miss stats.
// Backend op counts are only present when the app configured buckt.WithMetrics.
func (svc *APIService) Metrics(c *gin.Context) {
	storage, err := svc.client.StorageBytes(c.Request.Context())
	if err != nil {
		abort500(c, "failed to read storage bytes", err)
		return
	}
	hits, misses := svc.client.CacheStats()

	type opStat struct {
		Count   int64  `json:"count"`
		Errors  int64  `json:"errors"`
		Bytes   int64  `json:"bytes"`
		R2Class string `json:"r2_class,omitempty"`
	}
	// metricsOn reflects whether the app configured buckt.WithMetrics — not
	// whether any ops have been recorded yet. A freshly started app has metrics
	// enabled but an empty backends map until the first backend call, and the UI
	// must distinguish "off" from "on but idle".
	snap, metricsOn := svc.client.MetricsSnapshot()
	backends := map[string]map[string]opStat{}
	if metricsOn {
		for backend, ops := range snap {
			m := make(map[string]opStat, len(ops))
			for op, s := range ops {
				m[op] = opStat{Count: s.Count, Errors: s.Errors, Bytes: s.Bytes, R2Class: metrics.R2Class(op)}
			}
			backends[backend] = m
		}
	}

	c.JSON(200, gin.H{
		"storage_bytes":   storage,
		"cache":           gin.H{"hits": hits, "misses": misses},
		"backends":        backends,
		"metrics_enabled": metricsOn,
	})
}

// CreateFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).CreateFolder of APIService.FolderService.
func (svc *APIService) CreateFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	var req struct {
		ParentID    string `json:"parent_id"`
		FolderName  string `json:"folder_name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid request", err.Error()))
		return
	}

	// create the folder
	new_folder_id, err := svc.client.NewFolder(user_id, req.ParentID, req.FolderName, req.Description)
	if err != nil {
		abort500(c, "failed to create folder", err)
		return
	}

	c.JSON(200, response.Success("folder created, ID: "+new_folder_id))
}

// GetFolderContent implements domain.APIService.
func (svc *APIService) GetFolderContent(c *gin.Context) {
	user_id := c.GetString("owner_id")

	// get the folder_id from the request
	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	// get the folder content
	folderContent, err := svc.client.GetFolderWithContent(user_id, folderID)
	if err != nil {
		abort500(c, "failed to get folder content", err)
		return
	}

	c.JSON(200, response.Success(folderContent))
}

// GetFilesInFolder implements domain.APIService.
func (svc *APIService) GetFilesInFolder(c *gin.Context) {
	// get the parent_id from the request
	parentID, ok := requireParam(c, "parent_id")
	if !ok {
		return
	}

	// get the files in the folder
	files, err := svc.client.ListFiles(parentID)
	if err != nil {
		abort500(c, "failed to get files", err)
		return
	}

	c.JSON(200, response.Success(files))
}

// GetSubFolders implements domain.APIService.
func (svc *APIService) GetSubFolders(c *gin.Context) {
	parentID, ok := requireParam(c, "parent_id")
	if !ok {
		return
	}

	folders, err := svc.client.ListFolders(parentID)
	if err != nil {
		abort500(c, "failed to get sub-folders", err)
		return
	}

	c.JSON(200, response.Success(folders))
}

// GetDescendants implements domain.APIService.
func (svc *APIService) GetDescendants(c *gin.Context) {
	user_id := c.GetString("owner_id")

	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	folder, err := svc.client.GetFolderWithContent(user_id, folderID)
	if err != nil {
		abort500(c, "failed to get descendants", err)
		return
	}

	c.JSON(200, response.Success(folder))
}

// DeleteFolder implements domain.APIService.
func (svc *APIService) DeleteFolder(c *gin.Context) {
	// get the folder_id from the request
	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	// ge tthe folder with content
	_, err := svc.client.DeleteFolder(folderID)
	if err != nil {
		abort500(c, "failed to delete folder", err)
		return
	}

	c.JSON(200, response.Success("folder deleted"))
}

// DeleteFolderPermanently implements domain.APIService.
func (svc *APIService) DeleteFolderPermanently(c *gin.Context) {
	user_id := c.GetString("owner_id")

	// get the folder_id from the request
	folderID, ok := requireParam(c, "folder_id")
	if !ok {
		return
	}

	// ge tthe folder with content
	_, err := svc.client.DeleteFolderPermanently(user_id, folderID)
	if err != nil {
		abort500(c, "failed to delete folder", err)
		return
	}

	c.JSON(200, response.Success("folder deleted"))
}

// MoveFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).MoveFolder of APIService.FolderService.
func (svc *APIService) MoveFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")
	var req struct {
		FolderID    string `json:"folder_id"`
		NewParentID string `json:"new_parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid request", err.Error()))
		return
	}

	// move the folder
	if err := svc.client.MoveFolder(user_id, req.FolderID, req.NewParentID); err != nil {
		abort500(c, "failed to move folder", err)
		return
	}

	c.JSON(200, response.Success("folder moved"))
}

// RenameFolder implements domain.APIService.
// Subtle: this method shadows the method (FolderService).RenameFolder of APIService.FolderService.
func (svc *APIService) RenameFolder(c *gin.Context) {
	user_id := c.GetString("owner_id")

	var req struct {
		FolderID string `json:"folder_id"`
		Name     string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(400, response.Error("invalid request", err.Error()))
		return
	}

	// rename the folder
	if err := svc.client.RenameFolder(user_id, req.FolderID, req.Name); err != nil {
		abort500(c, "failed to rename folder", err)
		return
	}

	c.JSON(200, response.Success("folder renamed"))
}

// UploadFile implements domain.APIService.
func (svc *APIService) UploadFile(c *gin.Context) {
	// get the user_id from the context
	user_id := c.GetString("owner_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(400, response.Error("file is required", err.Error()))
		return
	}

	// get the parent_id from the request
	parentID, ok := requireForm(c, "parent_id")
	if !ok {
		return
	}

	// Read file from request
	fileName, fileByte, err := fileutil.ProcessFileWithLimit(file, svc.client.MaxFileSize())
	if err != nil {
		abort500(c, "failed to process file", err)
		return
	}

	fileID, err := svc.client.UploadFile(user_id, parentID, fileName, file.Header.Get("Content-Type"), fileByte)
	if err != nil {
		abort500(c, "failed to create file", err)
		return
	}

	url := svc.constructURL(fileID)

	c.JSON(200, response.Success(url))
}

// DownloadFile implements domain.APIService.
func (svc *APIService) DownloadFile(c *gin.Context) {
	// get the file_id from the request
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	file, err := svc.client.GetFile(fileID)
	if err != nil {
		abort500(c, "failed to get file", err)
		return
	}

	// Ensure file data is available
	if file.Data == nil {
		c.AbortWithStatusJSON(500, response.Error("file data is empty", ""))
		return
	}

	// Set headers
	c.Header("Cache-Control", "public, max-age=86400")
	writeFileHeaders(c, file.Name, file.ContentType, file.Size, "attachment")

	// Send file data
	c.Data(200, file.ContentType, file.Data)
}

// ServeFile implements domain.APIService.
func (svc *APIService) ServeFile(c *gin.Context) {
	// get the file_id from the request
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	file, err := svc.client.GetFile(fileID)
	if err != nil {
		abort500(c, "failed to get file", err)
		return
	}

	// Ensure file data is available
	if file.Data == nil {
		c.AbortWithStatusJSON(500, response.Error("file data is empty", ""))
		return
	}

	// Only render inline for a small allowlist of media types. Types that can
	// execute script in this origin (text/html, image/svg+xml, ...) are forced
	// to download, closing a stored-XSS vector where an uploaded HTML/SVG file
	// would otherwise run in the buckt origin when opened via /serve.
	disposition := fileutil.DispositionFor(file.ContentType)

	// Set headers
	c.Header("Cache-Control", "public, max-age=86400")
	writeFileHeaders(c, file.Name, file.ContentType, file.Size, disposition)
	// Defense in depth: even if a type slips through as inline, this CSP blocks
	// script execution and sub-resource loads from the served document.
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox; img-src 'self'; media-src 'self'")

	// Send file data
	c.Data(200, file.ContentType, file.Data)
}

// ServeDerivative serves a generated image variant (e.g. "thumbnail"). If the
// variant doesn't exist — not generated, not an image, or derivatives aren't
// configured — it falls back to the original file, so an <img> tag pointing
// here always resolves.
func (svc *APIService) ServeDerivative(c *gin.Context) {
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}
	name, ok := requireParam(c, "name")
	if !ok {
		return
	}

	data, contentType, err := svc.client.GetDerivative(fileID, name)
	if err != nil {
		file, ferr := svc.client.GetFile(fileID)
		if ferr != nil {
			abort500(c, "failed to get file", ferr)
			return
		}
		data, contentType = file.Data, file.ContentType
	}

	// Variants are images, so serve inline (DispositionFor forces a download for
	// anything script-executable, mirroring ServeFile's XSS hardening).
	c.Header("Cache-Control", "public, max-age=86400")
	writeFileHeaders(c, name, contentType, int64(len(data)), fileutil.DispositionFor(contentType))
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox; img-src 'self'; media-src 'self'")
	c.Data(200, contentType, data)
}

func (svc *APIService) StreamFile(c *gin.Context) {
	// get the file_id from the request
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	file, stream, err := svc.client.GetFileStream(fileID)
	if err != nil {
		abort500(c, "failed to get file", err)
		return
	}
	defer func() { _ = stream.Close() }()

	// Set headers
	writeFileHeaders(c, file.Name, file.ContentType, file.Size, "attachment")

	// If file is small, use io.Copy
	if file.Size < 10*1024*1024 { // 10 MB threshold
		if _, err := io.Copy(c.Writer, stream); err != nil {
			abort500(c, "failed to stream file", err)
		}
		return
	}

	// Stream file in chunks (efficient for large files)
	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 64*1024) // 64 KB buffer
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					return false // Stop streaming if writing fails
				}
			}
			if err == io.EOF {
				return false // Stop streaming when the file ends
			}
			if err != nil {
				return false // Stop streaming on any error
			}
		}
	})

	// // Set headers to enable streaming
	// c.Header("Accept-Ranges", "bytes")
	// c.Header("Content-Type", file.ContentType)

	// // Get file size
	// fileSize := file.Size
	// rangeHeader := c.GetHeader("Range")

	// // Default range (entire file)
	// start, end := int64(0), fileSize-1

	// if rangeHeader != "" {
	// 	var parseErr error
	// 	start, end, parseErr = parseRange(rangeHeader, fileSize)
	// 	if parseErr != nil {
	// 		c.AbortWithStatusJSON(http.StatusRequestedRangeNotSatisfiable, response.Error("invalid range", ""))
	// 		return
	// 	}

	// 	// Set partial content headers
	// 	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	// 	c.Status(http.StatusPartialContent)

	// 	// Manually discard bytes up to `start`
	// 	discarded, err := io.CopyN(io.Discard, stream, start)
	// 	if err != nil || discarded != start {
	// 		c.AbortWithStatusJSON(http.StatusInternalServerError, response.WrapError("failed to discard bytes", err))
	// 		return
	// 	}
	// }

	// // Stream only the requested range
	// bytesToSend := end - start + 1
	// io.CopyN(c.Writer, stream, bytesToSend)
}

// DeleteFile implements domain.APIService.
// Subtle: this method shadows the method (FileService).DeleteFile of APIService.FileService.
func (svc *APIService) DeleteFile(c *gin.Context) {
	// get the file_id from the request
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	// delete the file
	_, err := svc.client.DeleteFile(fileID)
	if err != nil {
		abort500(c, "failed to delete file", err)
		return
	}

	c.JSON(200, response.Success("file deleted"))
}

func (svc *APIService) DeleteFilePermanently(c *gin.Context) {
	// get the file_id from the request
	fileID, ok := requireParam(c, "file_id")
	if !ok {
		return
	}

	// delete the file
	_, err := svc.client.DeleteFilePermanently(fileID)
	if err != nil {
		abort500(c, "failed to delete file", err)
		return
	}

	c.JSON(200, response.Success("file deleted"))
}

/* Helper functions */

func (f *APIService) constructURL(s string) string {
	return fmt.Sprintf("/serve/%s", s)
}

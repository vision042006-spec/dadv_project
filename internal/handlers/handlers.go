package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dadv-project/internal/config"
	"dadv-project/internal/db"
	"dadv-project/internal/middleware"
	"dadv-project/internal/queue"
)

type Handler struct {
	db    *db.DB
	queue *queue.Queue
	cfg   *config.Config
}

func New(db *db.DB, q *queue.Queue, cfg *config.Config) *Handler {
	return &Handler{
		db:    db,
		queue: q,
		cfg:   cfg,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		api.POST("/upload", h.UploadFile)
		api.GET("/job-status/:job_id", h.GetJobStatus)
		api.GET("/files", h.ListFiles)
		api.GET("/stats/file-types/:job_id", h.GetFileTypeStats)
		api.GET("/stats/size-distribution/:job_id", h.GetSizeDistribution)
		api.GET("/stats/ownership/:job_id", h.GetOwnershipStats)
		api.GET("/stats/temporal/:job_id", h.GetTemporalStats)
		api.GET("/stats/aggregate/:job_id", h.GetAggregateStats)
		api.GET("/anomalies/:job_id", h.GetAnomalies)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}

func (h *Handler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.error(c, http.StatusBadRequest, "no file uploaded")
		return
	}
	defer file.Close()

	if header.Size > h.cfg.Security.MaxFileSize {
		h.error(c, http.StatusBadRequest, "file too large")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !h.isAllowedFileType(contentType) {
		h.error(c, http.StatusBadRequest, "file type not allowed")
		return
	}

	safeName := sanitizeFilename(header.Filename)
	jobID := uuid.New().String()
	filePath := fmt.Sprintf("./data/uploads/%s_%s", jobID, safeName)

	if err := h.saveFile(file, filePath); err != nil {
		h.error(c, http.StatusInternalServerError, "failed to save file")
		return
	}

	userID := c.GetInt64("user_id")

	_, err = h.db.CreateFile(c.Request.Context(), userID, jobID, safeName, header.Size, contentType)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to create file record")
		return
	}

	jobPayload := &queue.JobPayload{
		JobID:      jobID,
		FilePath:  filePath,
		FileName:  safeName,
		FileSize:  header.Size,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.queue.Enqueue(c.Request.Context(), jobPayload); err != nil {
		h.error(c, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": jobID,
		"status": "pending",
		"message": "File uploaded and queued for processing",
	})
}

func (h *Handler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	userID := c.GetInt64("user_id")

	status, err := h.queue.GetStatus(c.Request.Context(), jobID)
	if err != nil {
		_, dbStatus, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
		if err != nil {
			h.error(c, http.StatusNotFound, "job not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"job_id": jobID,
			"status": dbStatus,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"status": status.Status,
		"created": status.Created,
		"started": status.Started,
		"completed": status.Completed,
		"error": status.ErrorMessage,
		"row_count": status.RowCount,
	})
}

func (h *Handler) ListFiles(c *gin.Context) {
	userID := c.GetInt64("user_id")
	files, err := h.db.ListFiles(c.Request.Context(), userID)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to list files")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"total": len(files),
	})
}

func (h *Handler) GetFileTypeStats(c *gin.Context) {
	jobID := c.Param("job_id")
	userID := c.GetInt64("user_id")

	fileID, _, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
	if err != nil {
		h.error(c, http.StatusNotFound, "job not found")
		return
	}

	stats, err := h.db.GetFileTypeStats(c.Request.Context(), fileID)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) GetSizeDistribution(c *gin.Context) {
	jobID := c.Param("job_id")
	userID := c.GetInt64("user_id")

	fileID, _, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
	if err != nil {
		h.error(c, http.StatusNotFound, "job not found")
		return
	}

	dist, err := h.db.GetSizeDistribution(c.Request.Context(), fileID)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to get distribution")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dist})
}

func (h *Handler) GetOwnershipStats(c *gin.Context) {
	jobID := c.Param("job_id")
	userID := c.GetInt64("user_id")

	fileID, _, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
	if err != nil {
		h.error(c, http.StatusNotFound, "job not found")
		return
	}

	stats, err := h.db.GetOwnershipStats(c.Request.Context(), fileID)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) GetTemporalStats(c *gin.Context) {
	jobID := c.Param("job_id")
	userID := c.GetInt64("user_id")

	fileID, _, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
	if err != nil {
		h.error(c, http.StatusNotFound, "job not found")
		return
	}

	stats, err := h.db.GetTemporalStats(c.Request.Context(), fileID)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) GetAggregateStats(c *gin.Context) {
	jobID := c.Param("job_id")
	userID := c.GetInt64("user_id")

	fileID, _, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
	if err != nil {
		h.error(c, http.StatusNotFound, "job not found")
		return
	}

	stats, err := h.db.GetAggregateStats(c.Request.Context(), fileID)
	if err != nil {
		h.error(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) GetAnomalies(c *gin.Context) {
	jobID := c.Param("job_id")
	userID := c.GetInt64("user_id")

	fileID, _, err := h.db.GetFileByJobID(c.Request.Context(), userID, jobID)
	if err != nil {
		h.error(c, http.StatusNotFound, "job not found")
		return
	}

	anomalies, err := h.db.GetAnomalies(c.Request.Context(), fileID)
	if err != nil {
		empty := []map[string]interface{}{}
		c.JSON(http.StatusOK, gin.H{"data": empty})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": anomalies})
}

func (h *Handler) error(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}

func (h *Handler) isAllowedFileType(contentType string) bool {
	for _, t := range h.cfg.Security.AllowedFileTypes {
		if t == contentType {
			return true
		}
	}
	return false
}

func (h *Handler) saveFile(file io.Reader, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	return err
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".csv" && ext != ".json" && ext != ".xlsx" && ext != ".xls" {
		ext = ".dat"
	}

	return name
}
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// KnowledgeHandler processes HTTP requests related to knowledge resources
type KnowledgeHandler struct {
	kgService interfaces.KnowledgeService
	kbService interfaces.KnowledgeBaseService
}

// NewKnowledgeHandler creates a new knowledge handler instance
func NewKnowledgeHandler(
	kgService interfaces.KnowledgeService,
	kbService interfaces.KnowledgeBaseService,
) *KnowledgeHandler {
	return &KnowledgeHandler{kgService: kgService, kbService: kbService}
}

// validateKnowledgeBaseAccess validates access permissions to a knowledge base
// Returns the knowledge base, the knowledge base ID, and any errors encountered
func (h *KnowledgeHandler) validateKnowledgeBaseAccess(c *gin.Context) (*types.KnowledgeBase, string, error) {
	ctx := c.Request.Context()

	// Get knowledge base ID from URL path parameter
	kbID := secutils.SanitizeForLog(c.Param("id"))
	if kbID == "" {
		logger.Error(ctx, "Knowledge base ID is empty")
		return nil, "", errors.NewBadRequestError("Knowledge base ID cannot be empty")
	}

	// Get knowledge base details
	kb, err := h.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		return nil, kbID, errors.NewInternalServerError(err.Error())
	}

	// Verify tenant permissions
	if kb.TenantID != c.GetUint64(types.TenantIDContextKey.String()) {
		logger.Warnf(
			ctx,
			"Permission denied to access this knowledge base, tenant ID mismatch, "+
				"requested tenant ID: %d, knowledge base tenant ID: %d",
			c.GetUint64(types.TenantIDContextKey.String()),
			kb.TenantID,
		)
		return nil, kbID, errors.NewForbiddenError("Permission denied to access this knowledge base")
	}

	return kb, kbID, nil
}

// handleDuplicateKnowledgeError handles cases where duplicate knowledge is detected
// Returns true if the error was a duplicate error and was handled, false otherwise
func (h *KnowledgeHandler) handleDuplicateKnowledgeError(c *gin.Context,
	err error, knowledge *types.Knowledge, duplicateType string,
) bool {
	if dupErr, ok := err.(*types.DuplicateKnowledgeError); ok {
		ctx := c.Request.Context()
		logger.Warnf(ctx, "Detected duplicate %s: %s", duplicateType, secutils.SanitizeForLog(dupErr.Error()))
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": dupErr.Error(),
			"data":    knowledge, // knowledge contains the existing document
			"code":    fmt.Sprintf("duplicate_%s", duplicateType),
		})
		return true
	}
	return false
}

// CreateKnowledgeFromFile godoc
// @Summary      파일로 지식 생성
// @Description  파일을 업로드하고 지식 항목을 생성합니다
// @Tags         지식 관리
// @Accept       multipart/form-data
// @Produce      json
// @Param        id                path      string  true   "지식베이스 ID"
// @Param        file              formData  file    true   "업로드된 파일"
// @Param        fileName          formData  string  false  "사용자 정의 파일명"
// @Param        metadata          formData  string  false  "메타데이터 JSON"
// @Param        enable_multimodel formData  bool    false  "멀티모달 처리 활성화"
// @Success      200               {object}  map[string]interface{}  "생성된 지식"
// @Failure      400               {object}  errors.AppError         "잘못된 요청 파라미터"
// @Failure      409               {object}  map[string]interface{}  "파일 중복"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/knowledge/file [post]
func (h *KnowledgeHandler) CreateKnowledgeFromFile(c *gin.Context) {
	ctx := c.Request.Context()
	logger.Info(ctx, "Start creating knowledge from file")

	// Validate access to the knowledge base
	_, kbID, err := h.validateKnowledgeBaseAccess(c)
	if err != nil {
		c.Error(err)
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		logger.Error(ctx, "File upload failed", err)
		c.Error(errors.NewBadRequestError("File upload failed").WithDetails(err.Error()))
		return
	}

	// Get custom filename if provided (for folder uploads with path)
	customFileName := c.PostForm("fileName")
	customFileName = secutils.SanitizeForLog(customFileName)
	displayFileName := file.Filename
	displayFileName = secutils.SanitizeForLog(displayFileName)
	if customFileName != "" {
		displayFileName = customFileName
		logger.Infof(ctx, "Using custom filename: %s (original: %s)", customFileName, displayFileName)
	}

	logger.Infof(ctx, "File upload successful, filename: %s, size: %.2f KB", displayFileName, float64(file.Size)/1024)
	logger.Infof(ctx, "Creating knowledge, knowledge base ID: %s, filename: %s", kbID, displayFileName)

	// Parse metadata if provided
	var metadata map[string]string
	metadataStr := c.PostForm("metadata")
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			logger.Error(ctx, "Failed to parse metadata", err)
			c.Error(errors.NewBadRequestError("Invalid metadata format").WithDetails(err.Error()))
			return
		}
		logger.Infof(ctx, "Received file metadata: %s", secutils.SanitizeForLog(fmt.Sprintf("%v", metadata)))
	}

	enableMultimodelForm := c.PostForm("enable_multimodel")
	var enableMultimodel *bool
	if enableMultimodelForm != "" {
		parseBool, err := strconv.ParseBool(enableMultimodelForm)
		if err != nil {
			logger.Error(ctx, "Failed to parse enable_multimodel", err)
			c.Error(errors.NewBadRequestError("Invalid enable_multimodel format").WithDetails(err.Error()))
			return
		}
		enableMultimodel = &parseBool
	}

	// Create knowledge entry from the file
	knowledge, err := h.kgService.CreateKnowledgeFromFile(ctx, kbID, file, metadata, enableMultimodel, customFileName)
	// Check for duplicate knowledge error
	if err != nil {
		if h.handleDuplicateKnowledgeError(c, err, knowledge, "file") {
			return
		}
		if appErr, ok := errors.IsAppError(err); ok {
			c.Error(appErr)
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Knowledge created successfully, ID: %s, title: %s",
		secutils.SanitizeForLog(knowledge.ID),
		secutils.SanitizeForLog(knowledge.Title),
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    knowledge,
	})
}

// CreateKnowledgeFromURL godoc
// @Summary      URL로 지식 생성
// @Description  지정된 URL에서 내용을 가져와 지식 항목을 생성합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "지식베이스 ID"
// @Param        request  body      object{url=string,enable_multimodel=bool,title=string}  true  "URL 요청"
// @Success      201      {object}  map[string]interface{}  "생성된 지식"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Failure      409      {object}  map[string]interface{}  "URL 중복"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/knowledge/url [post]
func (h *KnowledgeHandler) CreateKnowledgeFromURL(c *gin.Context) {
	ctx := c.Request.Context()
	logger.Info(ctx, "Start creating knowledge from URL")

	// Validate access to the knowledge base
	_, kbID, err := h.validateKnowledgeBaseAccess(c)
	if err != nil {
		c.Error(err)
		return
	}

	// Parse URL from request body
	var req struct {
		URL              string `json:"url" binding:"required"`
		EnableMultimodel *bool  `json:"enable_multimodel"`
		Title            string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse URL request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Received URL request: %s", secutils.SanitizeForLog(req.URL))
	logger.Infof(
		ctx,
		"Creating knowledge from URL, knowledge base ID: %s, URL: %s",
		secutils.SanitizeForLog(kbID),
		secutils.SanitizeForLog(req.URL),
	)

	// Create knowledge entry from the URL
	knowledge, err := h.kgService.CreateKnowledgeFromURL(ctx, kbID, req.URL, req.EnableMultimodel, req.Title)
	// Check for duplicate knowledge error
	if err != nil {
		if h.handleDuplicateKnowledgeError(c, err, knowledge, "url") {
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Knowledge created successfully from URL, ID: %s, title: %s",
		secutils.SanitizeForLog(knowledge.ID),
		secutils.SanitizeForLog(knowledge.Title),
	)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    knowledge,
	})
}

// CreateManualKnowledge godoc
// @Summary      수동으로 지식 생성
// @Description  Markdown 형식의 지식 내용을 수동으로 입력합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "지식베이스 ID"
// @Param        request  body      types.ManualKnowledgePayload true  "수동 지식 내용"
// @Success      200      {object}  map[string]interface{}       "생성된 지식"
// @Failure      400      {object}  errors.AppError              "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/knowledge/manual [post]
func (h *KnowledgeHandler) CreateManualKnowledge(c *gin.Context) {
	ctx := c.Request.Context()
	logger.Info(ctx, "Start creating manual knowledge")

	_, kbID, err := h.validateKnowledgeBaseAccess(c)
	if err != nil {
		c.Error(err)
		return
	}

	var req types.ManualKnowledgePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse manual knowledge request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	knowledge, err := h.kgService.CreateKnowledgeFromManual(ctx, kbID, &req)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok {
			c.Error(appErr)
			return
		}
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"kb_id": kbID,
		})
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Manual knowledge created successfully, knowledge ID: %s",
		secutils.SanitizeForLog(knowledge.ID))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    knowledge,
	})
}

// GetKnowledge godoc
// @Summary      지식 상세 조회
// @Description  ID로 지식 항목 상세 정보를 조회합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "지식 ID"
// @Success      200  {object}  map[string]interface{}  "지식 상세 정보"
// @Failure      400  {object}  errors.AppError         "잘못된 요청 파라미터"
// @Failure      404  {object}  errors.AppError         "지식이 존재하지 않습니다"
// @Security     Bearer
// @Router       /knowledge/{id} [get]
func (h *KnowledgeHandler) GetKnowledge(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving knowledge")

	// Get knowledge ID from URL path parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Knowledge ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Retrieving knowledge, ID: %s", secutils.SanitizeForLog(id))
	knowledge, err := h.kgService.GetKnowledgeByID(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Knowledge retrieved successfully, ID: %s, title: %s",
		secutils.SanitizeForLog(knowledge.ID),
		secutils.SanitizeForLog(knowledge.Title),
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    knowledge,
	})
}

// ListKnowledge godoc
// @Summary      지식 목록 조회
// @Description  지식베이스의 지식 목록 조회，페이지네이션 및 필터 지원
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id         path      string  true   "지식베이스 ID"
// @Param        page       query     int     false  "페이지 번호"
// @Param        page_size  query     int     false  "페이지당 항목 수"
// @Param        tag_id     query     string  false  "태그 ID 필터"
// @Param        keyword    query     string  false  "키워드 검색"
// @Param        file_type  query     string  false  "파일 유형 필터"
// @Success      200        {object}  map[string]interface{}  "지식 목록"
// @Failure      400        {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/knowledge [get]
func (h *KnowledgeHandler) ListKnowledge(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving knowledge list")

	// Get knowledge base ID from URL path parameter
	kbID := secutils.SanitizeForLog(c.Param("id"))
	if kbID == "" {
		logger.Error(ctx, "Knowledge base ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge base ID cannot be empty"))
		return
	}

	// Parse pagination parameters from query string
	var pagination types.Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		logger.Error(ctx, "Failed to parse pagination parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	tagID := c.Query("tag_id")
	keyword := c.Query("keyword")
	fileType := c.Query("file_type")

	logger.Infof(
		ctx,
		"Retrieving knowledge list under knowledge base, knowledge base ID: %s, tag_id: %s, keyword: %s, file_type: %s, page: %d, page size: %d",
		secutils.SanitizeForLog(kbID),
		secutils.SanitizeForLog(tagID),
		secutils.SanitizeForLog(keyword),
		secutils.SanitizeForLog(fileType),
		pagination.Page,
		pagination.PageSize,
	)

	// Retrieve paginated knowledge entries
	result, err := h.kgService.ListPagedKnowledgeByKnowledgeBaseID(ctx, kbID, &pagination, tagID, keyword, fileType)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Knowledge list retrieved successfully, knowledge base ID: %s, total: %d",
		secutils.SanitizeForLog(kbID),
		result.Total,
	)
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      result.Data,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

// DeleteKnowledge godoc
// @Summary      지식 삭제
// @Description  ID로 지식 항목을 삭제합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "지식 ID"
// @Success      200  {object}  map[string]interface{}  "삭제 성공"
// @Failure      400  {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/{id} [delete]
func (h *KnowledgeHandler) DeleteKnowledge(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start deleting knowledge")

	// Get knowledge ID from URL path parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Knowledge ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Deleting knowledge, ID: %s", secutils.SanitizeForLog(id))
	err := h.kgService.DeleteKnowledge(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Knowledge deleted successfully, ID: %s", secutils.SanitizeForLog(id))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Deleted successfully",
	})
}

// DownloadKnowledgeFile godoc
// @Summary      지식 파일 다운로드
// @Description  지식 항목에 연결된 원본 파일을 다운로드합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      application/octet-stream
// @Param        id   path      string  true  "지식 ID"
// @Success      200  {file}    file    "파일 내용"
// @Failure      400  {object}  errors.AppError  "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/{id}/download [get]
func (h *KnowledgeHandler) DownloadKnowledgeFile(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start downloading knowledge file")

	// Get knowledge ID from URL path parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Knowledge ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Retrieving knowledge file, ID: %s", secutils.SanitizeForLog(id))

	// Get file content and filename
	file, filename, err := h.kgService.GetKnowledgeFile(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError("Failed to retrieve file").WithDetails(err.Error()))
		return
	}
	defer file.Close()

	logger.Infof(
		ctx,
		"Knowledge file retrieved successfully, ID: %s, filename: %s",
		secutils.SanitizeForLog(id),
		secutils.SanitizeForLog(filename),
	)

	// Set response headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	// Stream file content to response
	c.Stream(func(w io.Writer) bool {
		if _, err := io.Copy(w, file); err != nil {
			logger.Errorf(ctx, "Failed to send file: %v", err)
			return false
		}
		logger.Debug(ctx, "File sending completed")
		return false
	})
}

// GetKnowledgeBatchRequest defines parameters for batch knowledge retrieval
type GetKnowledgeBatchRequest struct {
	IDs []string `form:"ids" binding:"required"` // List of knowledge IDs
}

// GetKnowledgeBatch godoc
// @Summary      지식 일괄 조회
// @Description  ID 목록으로 지식 항목을 일괄 조회합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        ids  query     []string  true  "지식 ID 목록"
// @Success      200  {object}  map[string]interface{}  "지식 목록"
// @Failure      400  {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/batch [get]
func (h *KnowledgeHandler) GetKnowledgeBatch(c *gin.Context) {
	ctx := c.Request.Context()

	// Get tenant ID from context
	tenantID, ok := c.Get(types.TenantIDContextKey.String())
	if !ok {
		logger.Error(ctx, "Failed to get tenant ID")
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	// Parse request parameters from query string
	var req GetKnowledgeBatchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Batch retrieving knowledge, tenant ID: %d, number of knowledge IDs: %d",
		tenantID, len(req.IDs),
	)

	// Retrieve knowledge entries in batch
	knowledges, err := h.kgService.GetKnowledgeBatch(ctx, tenantID.(uint64), req.IDs)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError("Failed to retrieve knowledge list").WithDetails(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Batch knowledge retrieval successful, requested count: %d, returned count: %d",
		len(req.IDs), len(knowledges),
	)

	// Return results
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    knowledges,
	})
}

// UpdateKnowledge godoc
// @Summary      지식 업데이트
// @Description  지식 항목 정보를 업데이트합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string          true  "지식 ID"
// @Param        request  body      types.Knowledge true  "지식 정보"
// @Success      200      {object}  map[string]interface{}  "업데이트 성공"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/{id} [put]
func (h *KnowledgeHandler) UpdateKnowledge(c *gin.Context) {
	ctx := c.Request.Context()
	// Get knowledge ID from URL path parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Knowledge ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge ID cannot be empty"))
		return
	}

	var knowledge types.Knowledge
	if err := c.ShouldBindJSON(&knowledge); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	if err := h.kgService.UpdateKnowledge(ctx, &knowledge); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Knowledge updated successfully, knowledge ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Knowledge chunk updated successfully",
	})
}

// UpdateManualKnowledge godoc
// @Summary      수동 지식 업데이트
// @Description  수동으로 입력된 Markdown 지식 내용을 업데이트합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "지식 ID"
// @Param        request  body      types.ManualKnowledgePayload true  "수동 지식 내용"
// @Success      200      {object}  map[string]interface{}       "업데이트된 지식"
// @Failure      400      {object}  errors.AppError              "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/manual/{id} [put]
func (h *KnowledgeHandler) UpdateManualKnowledge(c *gin.Context) {
	ctx := c.Request.Context()
	logger.Info(ctx, "Start updating manual knowledge")

	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Knowledge ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge ID cannot be empty"))
		return
	}

	var req types.ManualKnowledgePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse manual knowledge update request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	knowledge, err := h.kgService.UpdateManualKnowledge(ctx, id, &req)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok {
			c.Error(appErr)
			return
		}
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"knowledge_id": id,
		})
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Manual knowledge updated successfully, knowledge ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    knowledge,
	})
}

type knowledgeTagBatchRequest struct {
	Updates map[string]*string `json:"updates" binding:"required,min=1"`
}

// UpdateKnowledgeTagBatch godoc
// @Summary      지식 태그 일괄 업데이트
// @Description  지식 항목의 태그를 일괄 업데이트합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        request  body      object  true  "태그 업데이트 요청"
// @Success      200      {object}  map[string]interface{}  "업데이트 성공"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/tags [put]
func (h *KnowledgeHandler) UpdateKnowledgeTagBatch(c *gin.Context) {
	ctx := c.Request.Context()
	var req knowledgeTagBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse knowledge tag batch request", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}
	if err := h.kgService.UpdateKnowledgeTagBatch(ctx, req.Updates); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// UpdateImageInfo godoc
// @Summary      이미지 정보 업데이트
// @Description  지식 청크의 이미지 정보를 업데이트합니다
// @Tags         지식 관리
// @Accept       json
// @Produce      json
// @Param        id        path      string  true  "지식 ID"
// @Param        chunk_id  path      string  true  "청크 ID"
// @Param        request   body      object{image_info=string}  true  "이미지 정보"
// @Success      200       {object}  map[string]interface{}     "업데이트 성공"
// @Failure      400       {object}  errors.AppError            "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge/image/{id}/{chunk_id} [put]
func (h *KnowledgeHandler) UpdateImageInfo(c *gin.Context) {
	ctx := c.Request.Context()
	logger.Info(ctx, "Start updating image info")

	// Get knowledge ID from URL path parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Knowledge ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge ID cannot be empty"))
		return
	}
	chunkID := secutils.SanitizeForLog(c.Param("chunk_id"))
	if chunkID == "" {
		logger.Error(ctx, "Chunk ID is empty")
		c.Error(errors.NewBadRequestError("Chunk ID cannot be empty"))
		return
	}

	var request struct {
		ImageInfo string `json:"image_info"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Update chunk properties
	logger.Infof(ctx, "Updating knowledge chunk, knowledge ID: %s, chunk ID: %s", id, chunkID)
	err := h.kgService.UpdateImageInfo(ctx, id, chunkID, secutils.SanitizeForLog(request.ImageInfo))
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Knowledge chunk updated successfully, knowledge ID: %s, chunk ID: %s", id, chunkID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Knowledge chunk image updated successfully",
	})
}

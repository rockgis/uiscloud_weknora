package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/uiscloud_weknora/internal/utils"
)

// FAQHandler handles FAQ knowledge base operations.
type FAQHandler struct {
	knowledgeService interfaces.KnowledgeService
}

// NewFAQHandler creates a new FAQ handler
func NewFAQHandler(knowledgeService interfaces.KnowledgeService) *FAQHandler {
	return &FAQHandler{knowledgeService: knowledgeService}
}

// ListEntries godoc
// @Summary      FAQ 항목 목록 조회
// @Description  지식베이스의 FAQ 항목 목록을 조회합니다. 페이지네이션과 필터를 지원합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id         path      string  true   "지식베이스 ID"
// @Param        page       query     int     false  "페이지 번호"
// @Param        page_size  query     int     false  "페이지당 항목 수"
// @Param        tag_id     query     string  false  "태그 ID 필터"
// @Param        keyword    query     string  false  "키워드 검색"
// @Success      200        {object}  map[string]interface{}  "FAQ 목록"
// @Failure      400        {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries [get]
func (h *FAQHandler) ListEntries(c *gin.Context) {
	ctx := c.Request.Context()
	var page types.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		logger.Error(ctx, "Failed to bind pagination query", err)
		c.Error(errors.NewBadRequestError("페이지 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	tagID := secutils.SanitizeForLog(c.Query("tag_id"))
	keyword := secutils.SanitizeForLog(c.Query("keyword"))

	result, err := h.knowledgeService.ListFAQEntries(ctx, secutils.SanitizeForLog(c.Param("id")), &page, tagID, keyword)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// UpsertEntries godoc
// @Summary      FAQ 항목 일괄 업데이트/삽입
// @Description  FAQ 항목을 비동기로 일괄 업데이트하거나 삽입합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "지식베이스 ID"
// @Param        request  body      types.FAQBatchUpsertPayload  true  "일괄 작업 요청"
// @Success      200      {object}  map[string]interface{}    "작업 ID"
// @Failure      400      {object}  errors.AppError           "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries [post]
func (h *FAQHandler) UpsertEntries(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.FAQBatchUpsertPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ upsert payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	taskID, err := h.knowledgeService.UpsertFAQEntries(ctx, secutils.SanitizeForLog(c.Param("id")), &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task_id": taskID,
		},
	})
}

// CreateEntry godoc
// @Summary      단일 FAQ 항목 생성
// @Description  단일 FAQ 항목을 동기적으로 생성합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string                true  "지식베이스 ID"
// @Param        request  body      types.FAQEntryPayload true  "FAQ 항목"
// @Success      200      {object}  map[string]interface{}  "생성된 FAQ 항목"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entry [post]
func (h *FAQHandler) CreateEntry(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.FAQEntryPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	entry, err := h.knowledgeService.CreateFAQEntry(ctx, secutils.SanitizeForLog(c.Param("id")), &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}

// UpdateEntry godoc
// @Summary      FAQ 항목 업데이트
// @Description  지정된 FAQ 항목을 업데이트합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id        path      string                true  "지식베이스 ID"
// @Param        entry_id  path      string                true  "FAQ 항목 ID"
// @Param        request   body      types.FAQEntryPayload true  "FAQ 항목"
// @Success      200       {object}  map[string]interface{}  "업데이트 성공"
// @Failure      400       {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries/{entry_id} [put]
func (h *FAQHandler) UpdateEntry(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.FAQEntryPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	if err := h.knowledgeService.UpdateFAQEntry(ctx,
		secutils.SanitizeForLog(c.Param("id")), secutils.SanitizeForLog(c.Param("entry_id")), &req); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// UpdateEntryTagBatch godoc
// @Summary      FAQ 태그 일괄 업데이트
// @Description  FAQ 항목의 태그를 일괄 업데이트합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "지식베이스 ID"
// @Param        request  body      object  true  "태그 업데이트 요청"
// @Success      200      {object}  map[string]interface{}  "업데이트 성공"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries/tags [put]
func (h *FAQHandler) UpdateEntryTagBatch(c *gin.Context) {
	ctx := c.Request.Context()
	var req faqEntryTagBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry tag batch payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}
	if err := h.knowledgeService.UpdateFAQEntryTagBatch(ctx,
		secutils.SanitizeForLog(c.Param("id")), req.Updates); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// UpdateEntryFieldsBatch godoc
// @Summary      FAQ 필드 일괄 업데이트
// @Description  FAQ 항목의 여러 필드를 일괄 업데이트합니다 (is_enabled, is_recommended, tag_id)
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string                        true  "지식베이스 ID"
// @Param        request  body      types.FAQEntryFieldsBatchUpdate  true  "필드 업데이트 요청"
// @Success      200      {object}  map[string]interface{}        "업데이트 성공"
// @Failure      400      {object}  errors.AppError               "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries/fields [put]
func (h *FAQHandler) UpdateEntryFieldsBatch(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.FAQEntryFieldsBatchUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry fields batch payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}
	if err := h.knowledgeService.UpdateFAQEntryFieldsBatch(ctx,
		secutils.SanitizeForLog(c.Param("id")), &req); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// faqDeleteRequest is a request for deleting FAQ entries in batch
type faqDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,dive,required"`
}

// faqEntryTagBatchRequest is a request for updating tags for FAQ entries in batch
type faqEntryTagBatchRequest struct {
	Updates map[string]*string `json:"updates" binding:"required,min=1"`
}

// DeleteEntries godoc
// @Summary      FAQ 항목 일괄 삭제
// @Description  지정된 FAQ 항목을 일괄 삭제합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "지식베이스 ID"
// @Param        request  body      object{ids=[]string}  true  "삭제할 FAQ ID 목록"
// @Success      200      {object}  map[string]interface{}  "삭제 성공"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries [delete]
func (h *FAQHandler) DeleteEntries(c *gin.Context) {
	ctx := c.Request.Context()
	var req faqDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(ctx, "Failed to bind FAQ delete payload: %s", secutils.SanitizeForLog(err.Error()))
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	if err := h.knowledgeService.DeleteFAQEntries(ctx,
		secutils.SanitizeForLog(c.Param("id")),
		secutils.SanitizeForLogArray(req.IDs)); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// SearchFAQ godoc
// @Summary      FAQ 검색
// @Description  하이브리드 검색으로 FAQ를 검색합니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string                true  "지식베이스 ID"
// @Param        request  body      types.FAQSearchRequest  true  "검색 요청"
// @Success      200      {object}  map[string]interface{}  "검색 결과"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/search [post]
func (h *FAQHandler) SearchFAQ(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.FAQSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ search payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}
	req.QueryText = secutils.SanitizeForLog(req.QueryText)
	if req.MatchCount <= 0 {
		req.MatchCount = 10
	}
	if req.MatchCount > 200 {
		req.MatchCount = 200
	}
	entries, err := h.knowledgeService.SearchFAQEntries(ctx, secutils.SanitizeForLog(c.Param("id")), &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entries,
	})
}

// ExportEntries godoc
// @Summary      FAQ 항목 내보내기
// @Description  모든 FAQ 항목을 CSV 파일로 내보냅니다
// @Tags         FAQ 관리
// @Accept       json
// @Produce      text/csv
// @Param        id   path      string  true  "지식베이스 ID"
// @Success      200  {file}    file    "CSV 파일"
// @Failure      400  {object}  errors.AppError  "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/faq/entries/export [get]
func (h *FAQHandler) ExportEntries(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	csvData, err := h.knowledgeService.ExportFAQEntries(ctx, kbID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	// Set response headers for CSV download
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=faq_export.csv")
	// Add BOM for Excel compatibility with UTF-8
	bom := []byte{0xEF, 0xBB, 0xBF}
	c.Data(http.StatusOK, "text/csv; charset=utf-8", append(bom, csvData...))
}

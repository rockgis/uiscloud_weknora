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

// TagHandler handles knowledge base tag operations.
type TagHandler struct {
	tagService interfaces.KnowledgeTagService
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(tagService interfaces.KnowledgeTagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

// ListTags godoc
// @Summary      태그 목록 조회
// @Description  지식베이스의 모든 태그와 통계 정보를 조회합니다
// @Tags         태그 관리
// @Accept       json
// @Produce      json
// @Param        id         path      string  true   "지식베이스 ID"
// @Param        page       query     int     false  "페이지 번호"
// @Param        page_size  query     int     false  "페이지당 항목 수"
// @Param        keyword    query     string  false  "키워드 검색"
// @Success      200        {object}  map[string]interface{}  "태그 목록"
// @Failure      400        {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/tags [get]
func (h *TagHandler) ListTags(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var page types.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		logger.Error(ctx, "Failed to bind pagination query", err)
		c.Error(errors.NewBadRequestError("페이지 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	keyword := secutils.SanitizeForLog(c.Query("keyword"))

	tags, err := h.tagService.ListTags(ctx, kbID, &page, keyword)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tags,
	})
}

type createTagRequest struct {
	Name      string `json:"name"       binding:"required"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// CreateTag godoc
// @Summary      태그 생성
// @Description  지식베이스에 새 태그를 생성합니다
// @Tags         태그 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "지식베이스 ID"
// @Param        request  body      object{name=string,color=string,sort_order=int}  true  "태그 정보"
// @Success      200      {object}  map[string]interface{}  "생성된 태그"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/tags [post]
func (h *TagHandler) CreateTag(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req createTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind create tag payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	tag, err := h.tagService.CreateTag(ctx, kbID,
		secutils.SanitizeForLog(req.Name), secutils.SanitizeForLog(req.Color), req.SortOrder)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"kb_id": kbID,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

type updateTagRequest struct {
	Name      *string `json:"name"`
	Color     *string `json:"color"`
	SortOrder *int    `json:"sort_order"`
}

// UpdateTag godoc
// @Summary      태그 업데이트
// @Description  태그 정보를 업데이트합니다
// @Tags         태그 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "지식베이스 ID"
// @Param        tag_id   path      string  true  "태그 ID"
// @Param        request  body      object  true  "태그 업데이트 정보"
// @Success      200      {object}  map[string]interface{}  "업데이트된 태그"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/tags/{tag_id} [put]
func (h *TagHandler) UpdateTag(c *gin.Context) {
	ctx := c.Request.Context()

	tagID := secutils.SanitizeForLog(c.Param("tag_id"))
	var req updateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind update tag payload", err)
		c.Error(errors.NewBadRequestError("요청 파라미터가 올바르지 않습니다").WithDetails(err.Error()))
		return
	}

	tag, err := h.tagService.UpdateTag(ctx, tagID, req.Name, req.Color, req.SortOrder)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tag_id": tagID,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

// DeleteTag godoc
// @Summary      태그 삭제
// @Description  태그를 삭제합니다. force=true를 사용하면 참조된 태그를 강제 삭제합니다
// @Tags         태그 관리
// @Accept       json
// @Produce      json
// @Param        id      path      string  true   "지식베이스 ID"
// @Param        tag_id  path      string  true   "태그 ID"
// @Param        force   query     bool    false  "강제 삭제"
// @Success      200     {object}  map[string]interface{}  "삭제 성공"
// @Failure      400     {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /knowledge-bases/{id}/tags/{tag_id} [delete]
func (h *TagHandler) DeleteTag(c *gin.Context) {
	ctx := c.Request.Context()
	tagID := secutils.SanitizeForLog(c.Param("tag_id"))

	force := c.Query("force") == "true"

	if err := h.tagService.DeleteTag(ctx, tagID, force); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tag_id": tagID,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// NOTE: TagHandler currently exposes CRUD for tags and statistics.
// Knowledge / Chunk tagging is handled via dedicated knowledge and FAQ APIs.

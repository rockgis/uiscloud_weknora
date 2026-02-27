package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// ModelHandler handles HTTP requests for model-related operations
// It implements the necessary methods to create, retrieve, update, and delete models
type ModelHandler struct {
	service interfaces.ModelService
}

// NewModelHandler creates a new instance of ModelHandler
// It requires a model service implementation that handles business logic
// Parameters:
//   - service: An implementation of the ModelService interface
//
// Returns a pointer to the newly created ModelHandler
func NewModelHandler(service interfaces.ModelService) *ModelHandler {
	return &ModelHandler{service: service}
}

// hideSensitiveInfo hides sensitive information (APIKey, BaseURL) for builtin models
// Returns a copy of the model with sensitive fields cleared if it's a builtin model
func hideSensitiveInfo(model *types.Model) *types.Model {
	if !model.IsBuiltin {
		return model
	}

	// Create a copy with sensitive information hidden
	return &types.Model{
		ID:          model.ID,
		TenantID:    model.TenantID,
		Name:        model.Name,
		Type:        model.Type,
		Source:      model.Source,
		Description: model.Description,
		Parameters: types.ModelParameters{
			// Hide APIKey and BaseURL for builtin models
			BaseURL: "",
			APIKey:  "",
			// Keep other parameters like embedding dimensions
			EmbeddingParameters: model.Parameters.EmbeddingParameters,
			ParameterSize:       model.Parameters.ParameterSize,
		},
		IsBuiltin: model.IsBuiltin,
		Status:    model.Status,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

// CreateModelRequest defines the structure for model creation requests
// Contains all fields required to create a new model in the system
type CreateModelRequest struct {
	Name        string                `json:"name"        binding:"required"`
	Type        types.ModelType       `json:"type"        binding:"required"`
	Source      types.ModelSource     `json:"source"      binding:"required"`
	Description string                `json:"description"`
	Parameters  types.ModelParameters `json:"parameters"  binding:"required"`
}

// CreateModel godoc
// @Summary      모델 생성
// @Description  새 모델 설정을 생성합니다
// @Tags         모델 관리
// @Accept       json
// @Produce      json
// @Param        request  body      CreateModelRequest  true  "모델 정보"
// @Success      201      {object}  map[string]interface{}  "생성된 모델"
// @Failure      400      {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /models [post]
func (h *ModelHandler) CreateModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start creating model")

	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		logger.Error(ctx, "Tenant ID is empty")
		c.Error(errors.NewBadRequestError("Tenant ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Creating model, Tenant ID: %d, Model name: %s, Model type: %s",
		tenantID, secutils.SanitizeForLog(req.Name), secutils.SanitizeForLog(string(req.Type)))

	model := &types.Model{
		TenantID:    tenantID,
		Name:        secutils.SanitizeForLog(req.Name),
		Type:        types.ModelType(secutils.SanitizeForLog(string(req.Type))),
		Source:      req.Source,
		Description: secutils.SanitizeForLog(req.Description),
		Parameters:  req.Parameters,
	}

	if err := h.service.CreateModel(ctx, model); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(
		ctx,
		"Model created successfully, ID: %s, Name: %s",
		secutils.SanitizeForLog(model.ID),
		secutils.SanitizeForLog(model.Name),
	)

	// Hide sensitive information for builtin models (though newly created models are unlikely to be builtin)
	responseModel := hideSensitiveInfo(model)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    responseModel,
	})
}

// GetModel godoc
// @Summary      모델 상세 조회
// @Description  ID로 모델 상세 정보를 조회합니다
// @Tags         모델 관리
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "모델 ID"
// @Success      200  {object}  map[string]interface{}  "모델 상세 정보"
// @Failure      404  {object}  errors.AppError         "모델이 존재하지 않습니다"
// @Security     Bearer
// @Router       /models/{id} [get]
func (h *ModelHandler) GetModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving model")

	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Model ID is empty")
		c.Error(errors.NewBadRequestError("Model ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Retrieving model, ID: %s", id)
	model, err := h.service.GetModelByID(ctx, id)
	if err != nil {
		if err == service.ErrModelNotFound {
			logger.Warnf(ctx, "Model not found, ID: %s", id)
			c.Error(errors.NewNotFoundError("Model not found"))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Retrieved model successfully, ID: %s, Name: %s", model.ID, model.Name)

	// Hide sensitive information for builtin models
	responseModel := hideSensitiveInfo(model)
	if model.IsBuiltin {
		logger.Infof(ctx, "Builtin model detected, hiding sensitive information for model: %s", model.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseModel,
	})
}

// ListModels godoc
// @Summary      모델 목록 조회
// @Description  현재 테넌트의 모든 모델을 조회합니다
// @Tags         모델 관리
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "모델 목록"
// @Failure      400  {object}  errors.AppError         "잘못된 요청 파라미터"
// @Security     Bearer
// @Router       /models [get]
func (h *ModelHandler) ListModels(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving model list")

	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		logger.Error(ctx, "Tenant ID is empty")
		c.Error(errors.NewBadRequestError("Tenant ID cannot be empty"))
		return
	}

	models, err := h.service.ListModels(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Retrieved model list successfully, Tenant ID: %d, Total: %d models", tenantID, len(models))

	// Hide sensitive information for builtin models in the list
	responseModels := make([]*types.Model, len(models))
	for i, model := range models {
		responseModels[i] = hideSensitiveInfo(model)
		if model.IsBuiltin {
			logger.Infof(ctx, "Builtin model detected in list, hiding sensitive information for model: %s", model.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseModels,
	})
}

// UpdateModelRequest defines the structure for model update requests
// Contains fields that can be updated for an existing model
type UpdateModelRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Parameters  types.ModelParameters `json:"parameters"`
	Source      types.ModelSource     `json:"source"`
	Type        types.ModelType       `json:"type"`
}

// UpdateModel godoc
// @Summary      모델 업데이트
// @Description  모델 설정 정보를 업데이트합니다
// @Tags         모델 관리
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "모델 ID"
// @Param        request  body      UpdateModelRequest  true  "업데이트 정보"
// @Success      200      {object}  map[string]interface{}  "업데이트된 모델"
// @Failure      404      {object}  errors.AppError         "모델이 존재하지 않습니다"
// @Security     Bearer
// @Router       /models/{id} [put]
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start updating model")

	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Model ID is empty")
		c.Error(errors.NewBadRequestError("Model ID cannot be empty"))
		return
	}

	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Retrieving model information, ID: %s", id)
	model, err := h.service.GetModelByID(ctx, id)
	if err != nil {
		if err == service.ErrModelNotFound {
			logger.Warnf(ctx, "Model not found, ID: %s", id)
			c.Error(errors.NewNotFoundError("Model not found"))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Update model fields if they are provided in the request
	if req.Name != "" {
		model.Name = req.Name
	}
	model.Description = req.Description
	if req.Parameters != (types.ModelParameters{}) {
		model.Parameters = req.Parameters
	}
	model.Source = req.Source
	model.Type = req.Type

	logger.Infof(ctx, "Updating model, ID: %s, Name: %s", id, model.Name)
	if err := h.service.UpdateModel(ctx, model); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Model updated successfully, ID: %s", id)

	// Hide sensitive information for builtin models (though builtin models cannot be updated)
	responseModel := hideSensitiveInfo(model)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseModel,
	})
}

// DeleteModel godoc
// @Summary      모델 삭제
// @Description  지정된 모델을 삭제합니다
// @Tags         모델 관리
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "모델 ID"
// @Success      200  {object}  map[string]interface{}  "삭제 성공"
// @Failure      404  {object}  errors.AppError         "모델이 존재하지 않습니다"
// @Security     Bearer
// @Router       /models/{id} [delete]
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start deleting model")

	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Model ID is empty")
		c.Error(errors.NewBadRequestError("Model ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Deleting model, ID: %s", id)
	if err := h.service.DeleteModel(ctx, id); err != nil {
		if err == service.ErrModelNotFound {
			logger.Warnf(ctx, "Model not found, ID: %s", id)
			c.Error(errors.NewNotFoundError("Model not found"))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Model deleted successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Model deleted",
	})
}

package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/docreader/client"
	"github.com/Tencent/WeKnora/docreader/proto"
	"github.com/Tencent/WeKnora/internal/application/service/retriever"
	"github.com/Tencent/WeKnora/internal/config"
	werrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/tracing"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
)

// Error definitions for knowledge service operations
var (
	// ErrInvalidFileType is returned when an unsupported file type is provided
	ErrInvalidFileType = errors.New("unsupported file type")
	// ErrInvalidURL is returned when an invalid URL is provided
	ErrInvalidURL = errors.New("invalid URL")
	// ErrChunkNotFound is returned when a requested chunk cannot be found
	ErrChunkNotFound = errors.New("chunk not found")
	// ErrDuplicateFile is returned when trying to add a file that already exists
	ErrDuplicateFile = errors.New("file already exists")
	// ErrDuplicateURL is returned when trying to add a URL that already exists
	ErrDuplicateURL = errors.New("URL already exists")
	// ErrImageNotParse is returned when trying to update image information without enabling multimodel
	ErrImageNotParse = errors.New("image not parse without enable multimodel")
)

// knowledgeService implements the knowledge service interface
type knowledgeService struct {
	config          *config.Config
	retrieveEngine  interfaces.RetrieveEngineRegistry
	repo            interfaces.KnowledgeRepository
	kbService       interfaces.KnowledgeBaseService
	tenantRepo      interfaces.TenantRepository
	docReaderClient *client.Client
	chunkService    interfaces.ChunkService
	chunkRepo       interfaces.ChunkRepository
	tagRepo         interfaces.KnowledgeTagRepository
	tagService      interfaces.KnowledgeTagService
	fileSvc         interfaces.FileService
	modelService    interfaces.ModelService
	task            *asynq.Client
	graphEngine     interfaces.RetrieveGraphRepository
	redisClient     *redis.Client
}

const (
	manualContentMaxLength = 200000
	manualFileExtension    = ".md"
	faqImportBatchSize     = 50
)

// NewKnowledgeService creates a new knowledge service instance
func NewKnowledgeService(
	config *config.Config,
	repo interfaces.KnowledgeRepository,
	docReaderClient *client.Client,
	kbService interfaces.KnowledgeBaseService,
	tenantRepo interfaces.TenantRepository,
	chunkService interfaces.ChunkService,
	chunkRepo interfaces.ChunkRepository,
	tagRepo interfaces.KnowledgeTagRepository,
	tagService interfaces.KnowledgeTagService,
	fileSvc interfaces.FileService,
	modelService interfaces.ModelService,
	task *asynq.Client,
	graphEngine interfaces.RetrieveGraphRepository,
	retrieveEngine interfaces.RetrieveEngineRegistry,
	redisClient *redis.Client,
) (interfaces.KnowledgeService, error) {
	return &knowledgeService{
		config:          config,
		repo:            repo,
		kbService:       kbService,
		tenantRepo:      tenantRepo,
		docReaderClient: docReaderClient,
		chunkService:    chunkService,
		chunkRepo:       chunkRepo,
		tagRepo:         tagRepo,
		tagService:      tagService,
		fileSvc:         fileSvc,
		modelService:    modelService,
		task:            task,
		graphEngine:     graphEngine,
		retrieveEngine:  retrieveEngine,
		redisClient:     redisClient,
	}, nil
}

// GetRepository gets the knowledge repository
// Parameters:
//   - ctx: Context with authentication and request information
//
// Returns:
//   - interfaces.KnowledgeRepository: Knowledge repository
func (s *knowledgeService) GetRepository() interfaces.KnowledgeRepository {
	return s.repo
}

// isKnowledgeDeleting checks if a knowledge entry is being deleted.
// This is used to prevent async tasks from conflicting with deletion operations.
func (s *knowledgeService) isKnowledgeDeleting(ctx context.Context, tenantID uint64, knowledgeID string) bool {
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
	if err != nil {
		// If we can't find the knowledge, assume it's deleted
		logger.Warnf(ctx, "Failed to check knowledge deletion status (assuming deleted): %v", err)
		return true
	}
	if knowledge == nil {
		return true
	}
	return knowledge.ParseStatus == types.ParseStatusDeleting
}

// CreateKnowledgeFromFile creates a knowledge entry from an uploaded file
func (s *knowledgeService) CreateKnowledgeFromFile(ctx context.Context,
	kbID string, file *multipart.FileHeader, metadata map[string]string, enableMultimodel *bool, customFileName string,
) (*types.Knowledge, error) {
	logger.Info(ctx, "Start creating knowledge from file")

	// Use custom filename if provided, otherwise use original filename
	fileName := file.Filename
	if customFileName != "" {
		fileName = customFileName
		logger.Infof(ctx, "Using custom filename: %s (original: %s)", customFileName, file.Filename)
	}

	logger.Infof(ctx, "Knowledge base ID: %s, file: %s", kbID, fileName)

	// Get knowledge base configuration
	logger.Info(ctx, "Getting knowledge base configuration")
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return nil, err
	}

	if !IsImageType(getFileType(fileName)) {
		logger.Info(ctx, "Non-image file with multimodal enabled, skipping COS/VLM validation")
	} else {
		switch kb.StorageConfig.Provider {
		case "cos":
			if kb.StorageConfig.SecretID == "" || kb.StorageConfig.SecretKey == "" ||
				kb.StorageConfig.Region == "" || kb.StorageConfig.BucketName == "" ||
				kb.StorageConfig.AppID == "" {
				logger.Error(ctx, "COS configuration incomplete for image multimodal processing")
				return nil, werrors.NewBadRequestError("이미지 파일 업로드에는 완전한 객체 스토리지 설정이 필요합니다. 시스템 설정 페이지에서 설정을 완료해 주세요")
			}
		case "minio":
			if kb.StorageConfig.BucketName == "" {
				logger.Error(ctx, "MinIO configuration incomplete for image multimodal processing")
				return nil, werrors.NewBadRequestError("이미지 파일 업로드에는 완전한 객체 스토리지 설정이 필요합니다. 시스템 설정 페이지에서 설정을 완료해 주세요")
			}
		}

		if !kb.VLMConfig.Enabled || kb.VLMConfig.ModelID == "" {
			logger.Error(ctx, "VLM model is not configured")
			return nil, werrors.NewBadRequestError("이미지 파일 업로드에는 VLM 모델 설정이 필요합니다")
		}

		logger.Info(ctx, "Image multimodal configuration validation passed")
	}

	// Validate file type
	logger.Infof(ctx, "Checking file type: %s", fileName)
	if !isValidFileType(fileName) {
		logger.Error(ctx, "Invalid file type")
		return nil, ErrInvalidFileType
	}

	// Calculate file hash for deduplication
	logger.Info(ctx, "Calculating file hash")
	hash, err := calculateFileHash(file)
	if err != nil {
		logger.Errorf(ctx, "Failed to calculate file hash: %v", err)
		return nil, err
	}

	// Check if file already exists
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	logger.Infof(ctx, "Checking if file exists, tenant ID: %d", tenantID)
	exists, existingKnowledge, err := s.repo.CheckKnowledgeExists(ctx, tenantID, kbID, &types.KnowledgeCheckParams{
		Type:     "file",
		FileName: fileName,
		FileSize: file.Size,
		FileHash: hash,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to check knowledge existence: %v", err)
		return nil, err
	}
	if exists {
		logger.Infof(ctx, "File already exists: %s", fileName)
		// Update creation time for existing knowledge
		if err := s.repo.UpdateKnowledgeColumn(ctx, existingKnowledge.ID, "created_at", time.Now()); err != nil {
			logger.Errorf(ctx, "Failed to update existing knowledge: %v", err)
			return nil, err
		}
		return existingKnowledge, types.NewDuplicateFileError(existingKnowledge)
	}

	// Check storage quota
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	if tenantInfo.StorageQuota > 0 && tenantInfo.StorageUsed >= tenantInfo.StorageQuota {
		logger.Error(ctx, "Storage quota exceeded")
		return nil, types.NewStorageQuotaExceededError()
	}

	// Convert metadata to JSON format if provided
	var metadataJSON types.JSON
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			logger.Errorf(ctx, "Failed to marshal metadata: %v", err)
			return nil, err
		}
		metadataJSON = types.JSON(metadataBytes)
	}

	safeFilename, isValid := secutils.ValidateInput(fileName)
	if !isValid {
		logger.Errorf(ctx, "Invalid filename: %s", fileName)
		return nil, werrors.NewValidationError("파일명에 허용되지 않는 문자가 포함되어 있습니다")
	}

	// Create knowledge record
	logger.Info(ctx, "Creating knowledge record")
	knowledge := &types.Knowledge{
		TenantID:         tenantID,
		KnowledgeBaseID:  kbID,
		Type:             "file",
		Title:            safeFilename,
		FileName:         safeFilename,
		FileType:         getFileType(safeFilename),
		FileSize:         file.Size,
		FileHash:         hash,
		ParseStatus:      "pending",
		EnableStatus:     "disabled",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		EmbeddingModelID: kb.EmbeddingModelID,
		Metadata:         metadataJSON,
	}
	// Save knowledge record to database
	logger.Info(ctx, "Saving knowledge record to database")
	if err := s.repo.CreateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "Failed to create knowledge record, ID: %s, error: %v", knowledge.ID, err)
		return nil, err
	}
	// Save the file to storage
	logger.Infof(ctx, "Saving file, knowledge ID: %s", knowledge.ID)
	filePath, err := s.fileSvc.SaveFile(ctx, file, knowledge.TenantID, knowledge.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to save file, knowledge ID: %s, error: %v", knowledge.ID, err)
		return nil, err
	}
	knowledge.FilePath = filePath

	// Update knowledge record with file path
	logger.Info(ctx, "Updating knowledge record with file path")
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "Failed to update knowledge with file path, ID: %s, error: %v", knowledge.ID, err)
		return nil, err
	}

	// Enqueue document processing task to Asynq
	logger.Info(ctx, "Enqueuing document processing task to Asynq")
	enableMultimodelValue := false
	if enableMultimodel != nil {
		enableMultimodelValue = *enableMultimodel
	} else {
		enableMultimodelValue = kb.IsMultimodalEnabled()
	}

	// Check question generation config
	enableQuestionGeneration := false
	questionCount := 3 // default
	if kb.QuestionGenerationConfig != nil && kb.QuestionGenerationConfig.Enabled {
		enableQuestionGeneration = true
		if kb.QuestionGenerationConfig.QuestionCount > 0 {
			questionCount = kb.QuestionGenerationConfig.QuestionCount
		}
	}

	taskPayload := types.DocumentProcessPayload{
		TenantID:                 tenantID,
		KnowledgeID:              knowledge.ID,
		KnowledgeBaseID:          kbID,
		FilePath:                 filePath,
		FileName:                 safeFilename,
		FileType:                 getFileType(safeFilename),
		EnableMultimodel:         enableMultimodelValue,
		EnableQuestionGeneration: enableQuestionGeneration,
		QuestionCount:            questionCount,
	}

	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal document process task payload: %v", err)
		return knowledge, nil
	}

	task := asynq.NewTask(types.TypeDocumentProcess, payloadBytes, asynq.Queue("default"))
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue document process task: %v", err)
		return knowledge, nil
	}
	logger.Infof(
		ctx,
		"Enqueued document process task: id=%s queue=%s knowledge_id=%s",
		info.ID,
		info.Queue,
		knowledge.ID,
	)

	logger.Infof(ctx, "Knowledge from file created successfully, ID: %s", knowledge.ID)
	return knowledge, nil
}

// CreateKnowledgeFromURL creates a knowledge entry from a URL source
func (s *knowledgeService) CreateKnowledgeFromURL(ctx context.Context,
	kbID string, url string, enableMultimodel *bool, title string,
) (*types.Knowledge, error) {
	logger.Info(ctx, "Start creating knowledge from URL")
	logger.Infof(ctx, "Knowledge base ID: %s, URL: %s", kbID, url)

	// Get knowledge base configuration
	logger.Info(ctx, "Getting knowledge base configuration")
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return nil, err
	}

	// Validate URL format and security
	logger.Info(ctx, "Validating URL")
	if !isValidURL(url) || !secutils.IsValidURL(url) {
		logger.Error(ctx, "Invalid or unsafe URL format")
		return nil, ErrInvalidURL
	}

	// Check if URL already exists in the knowledge base
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	logger.Infof(ctx, "Checking if URL exists, tenant ID: %d", tenantID)
	fileHash := calculateStr(url)
	exists, existingKnowledge, err := s.repo.CheckKnowledgeExists(ctx, tenantID, kbID, &types.KnowledgeCheckParams{
		Type:     "url",
		URL:      url,
		FileHash: fileHash,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to check knowledge existence: %v", err)
		return nil, err
	}
	if exists {
		logger.Infof(ctx, "URL already exists: %s", url)
		// Update creation time for existing knowledge
		existingKnowledge.CreatedAt = time.Now()
		existingKnowledge.UpdatedAt = time.Now()
		if err := s.repo.UpdateKnowledge(ctx, existingKnowledge); err != nil {
			logger.Errorf(ctx, "Failed to update existing knowledge: %v", err)
			return nil, err
		}
		return existingKnowledge, types.NewDuplicateURLError(existingKnowledge)
	}

	// Check storage quota
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	if tenantInfo.StorageQuota > 0 && tenantInfo.StorageUsed >= tenantInfo.StorageQuota {
		logger.Error(ctx, "Storage quota exceeded")
		return nil, types.NewStorageQuotaExceededError()
	}

	// Create knowledge record
	logger.Info(ctx, "Creating knowledge record")
	knowledge := &types.Knowledge{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		KnowledgeBaseID:  kbID,
		Type:             "url",
		Title:            title,
		Source:           url,
		FileHash:         fileHash,
		ParseStatus:      "pending",
		EnableStatus:     "disabled",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		EmbeddingModelID: kb.EmbeddingModelID,
	}

	// Save knowledge record
	logger.Infof(ctx, "Saving knowledge record to database, ID: %s", knowledge.ID)
	if err := s.repo.CreateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "Failed to create knowledge record: %v", err)
		return nil, err
	}

	// Enqueue URL processing task to Asynq
	logger.Info(ctx, "Enqueuing URL processing task to Asynq")
	enableMultimodelValue := false
	if enableMultimodel != nil {
		enableMultimodelValue = *enableMultimodel
	} else {
		enableMultimodelValue = kb.IsMultimodalEnabled()
	}

	// Check question generation config
	enableQuestionGeneration := false
	questionCount := 3 // default
	if kb.QuestionGenerationConfig != nil && kb.QuestionGenerationConfig.Enabled {
		enableQuestionGeneration = true
		if kb.QuestionGenerationConfig.QuestionCount > 0 {
			questionCount = kb.QuestionGenerationConfig.QuestionCount
		}
	}

	taskPayload := types.DocumentProcessPayload{
		TenantID:                 tenantID,
		KnowledgeID:              knowledge.ID,
		KnowledgeBaseID:          kbID,
		URL:                      url,
		EnableMultimodel:         enableMultimodelValue,
		EnableQuestionGeneration: enableQuestionGeneration,
		QuestionCount:            questionCount,
	}

	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal URL process task payload: %v", err)
		return knowledge, nil
	}

	task := asynq.NewTask(types.TypeDocumentProcess, payloadBytes, asynq.Queue("default"))
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue URL process task: %v", err)
		return knowledge, nil
	}
	logger.Infof(ctx, "Enqueued URL process task: id=%s queue=%s knowledge_id=%s", info.ID, info.Queue, knowledge.ID)

	logger.Infof(ctx, "Knowledge from URL created successfully, ID: %s", knowledge.ID)
	return knowledge, nil
}

// CreateKnowledgeFromPassage creates a knowledge entry from text passages
func (s *knowledgeService) CreateKnowledgeFromPassage(ctx context.Context,
	kbID string, passage []string,
) (*types.Knowledge, error) {
	return s.createKnowledgeFromPassageInternal(ctx, kbID, passage, false)
}

// CreateKnowledgeFromPassageSync creates a knowledge entry from text passages and waits for indexing to complete.
func (s *knowledgeService) CreateKnowledgeFromPassageSync(ctx context.Context,
	kbID string, passage []string,
) (*types.Knowledge, error) {
	return s.createKnowledgeFromPassageInternal(ctx, kbID, passage, true)
}

// CreateKnowledgeFromManual creates or saves manual Markdown knowledge content.
func (s *knowledgeService) CreateKnowledgeFromManual(ctx context.Context,
	kbID string, payload *types.ManualKnowledgePayload,
) (*types.Knowledge, error) {
	logger.Info(ctx, "Start creating manual knowledge entry")

	if payload == nil {
		return nil, werrors.NewBadRequestError("요청 내용은 비워둘 수 없습니다")
	}

	cleanContent := secutils.CleanMarkdown(payload.Content)
	if strings.TrimSpace(cleanContent) == "" {
		return nil, werrors.NewValidationError("내용은 비워둘 수 없습니다")
	}
	if len([]rune(cleanContent)) > manualContentMaxLength {
		return nil, werrors.NewValidationError(fmt.Sprintf("내용 길이 제한 초과 (최대 %d자)", manualContentMaxLength))
	}

	safeTitle, ok := secutils.ValidateInput(payload.Title)
	if !ok {
		return nil, werrors.NewValidationError("제목에 허용되지 않는 문자가 포함되어 있거나 길이 제한을 초과했습니다")
	}

	status := strings.ToLower(strings.TrimSpace(payload.Status))
	if status == "" {
		status = types.ManualKnowledgeStatusDraft
	}
	if status != types.ManualKnowledgeStatusDraft && status != types.ManualKnowledgeStatusPublish {
		return nil, werrors.NewValidationError("상태는 draft 또는 publish만 지원됩니다")
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return nil, err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	now := time.Now()
	title := safeTitle
	if title == "" {
		title = fmt.Sprintf("Knowledge-%s", now.Format("20060102-150405"))
	}

	fileName := ensureManualFileName(title)
	meta := types.NewManualKnowledgeMetadata(cleanContent, status, 1)

	knowledge := &types.Knowledge{
		TenantID:         tenantID,
		KnowledgeBaseID:  kbID,
		Type:             types.KnowledgeTypeManual,
		Title:            title,
		Description:      "",
		Source:           types.KnowledgeTypeManual,
		ParseStatus:      types.ManualKnowledgeStatusDraft,
		EnableStatus:     "disabled",
		CreatedAt:        now,
		UpdatedAt:        now,
		EmbeddingModelID: kb.EmbeddingModelID,
		FileName:         fileName,
		FileType:         types.KnowledgeTypeManual,
	}
	if err := knowledge.SetManualMetadata(meta); err != nil {
		logger.Errorf(ctx, "Failed to set manual metadata: %v", err)
		return nil, err
	}
	knowledge.EnsureManualDefaults()

	if status == types.ManualKnowledgeStatusPublish {
		knowledge.ParseStatus = "pending"
	}

	if err := s.repo.CreateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "Failed to create manual knowledge record: %v", err)
		return nil, err
	}

	if status == types.ManualKnowledgeStatusPublish {
		logger.Infof(ctx, "Manual knowledge created, scheduling indexing, ID: %s", knowledge.ID)
		s.triggerManualProcessing(ctx, kb, knowledge, cleanContent, false)
	}

	return knowledge, nil
}

// createKnowledgeFromPassageInternal consolidates the common logic for creating knowledge from passages.
// When syncMode is true, chunk processing is performed synchronously; otherwise, it's processed asynchronously.
func (s *knowledgeService) createKnowledgeFromPassageInternal(ctx context.Context,
	kbID string, passage []string, syncMode bool,
) (*types.Knowledge, error) {
	if syncMode {
		logger.Info(ctx, "Start creating knowledge from passage (sync)")
	} else {
		logger.Info(ctx, "Start creating knowledge from passage")
	}
	logger.Infof(ctx, "Knowledge base ID: %s, passage count: %d", kbID, len(passage))

	safePassages := make([]string, 0, len(passage))
	for i, p := range passage {
		safePassage, isValid := secutils.ValidateInput(p)
		if !isValid {
			logger.Errorf(ctx, "Invalid passage content at index %d", i)
			return nil, werrors.NewValidationError(fmt.Sprintf("단락 %d 에 허용되지 않는 내용이 포함되어 있습니다", i+1))
		}
		safePassages = append(safePassages, safePassage)
	}

	// Get knowledge base configuration
	logger.Info(ctx, "Getting knowledge base configuration")
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return nil, err
	}

	// Create knowledge record
	if syncMode {
		logger.Info(ctx, "Creating knowledge record (sync)")
	} else {
		logger.Info(ctx, "Creating knowledge record")
	}
	knowledge := &types.Knowledge{
		ID:               uuid.New().String(),
		TenantID:         ctx.Value(types.TenantIDContextKey).(uint64),
		KnowledgeBaseID:  kbID,
		Type:             "passage",
		ParseStatus:      "pending",
		EnableStatus:     "disabled",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		EmbeddingModelID: kb.EmbeddingModelID,
	}

	// Save knowledge record
	logger.Infof(ctx, "Saving knowledge record to database, ID: %s", knowledge.ID)
	if err := s.repo.CreateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "Failed to create knowledge record: %v", err)
		return nil, err
	}

	// Process passages
	if syncMode {
		logger.Info(ctx, "Processing passage synchronously")
		s.processDocumentFromPassage(ctx, kb, knowledge, safePassages)
		logger.Infof(ctx, "Knowledge from passage created successfully (sync), ID: %s", knowledge.ID)
	} else {
		// Enqueue passage processing task to Asynq
		logger.Info(ctx, "Enqueuing passage processing task to Asynq")
		tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

		// Check question generation config
		enableQuestionGeneration := false
		questionCount := 3 // default
		if kb.QuestionGenerationConfig != nil && kb.QuestionGenerationConfig.Enabled {
			enableQuestionGeneration = true
			if kb.QuestionGenerationConfig.QuestionCount > 0 {
				questionCount = kb.QuestionGenerationConfig.QuestionCount
			}
		}

		taskPayload := types.DocumentProcessPayload{
			TenantID:                 tenantID,
			KnowledgeID:              knowledge.ID,
			KnowledgeBaseID:          kbID,
			Passages:                 safePassages,
			EnableMultimodel:         false,
			EnableQuestionGeneration: enableQuestionGeneration,
			QuestionCount:            questionCount,
		}

		payloadBytes, err := json.Marshal(taskPayload)
		if err != nil {
			logger.Errorf(ctx, "Failed to marshal passage process task payload: %v", err)
			return knowledge, nil
		}

		task := asynq.NewTask(types.TypeDocumentProcess, payloadBytes, asynq.Queue("default"))
		info, err := s.task.Enqueue(task)
		if err != nil {
			logger.Errorf(ctx, "Failed to enqueue passage process task: %v", err)
			return knowledge, nil
		}
		logger.Infof(ctx, "Enqueued passage process task: id=%s queue=%s knowledge_id=%s", info.ID, info.Queue, knowledge.ID)
		logger.Infof(ctx, "Knowledge from passage created successfully, ID: %s", knowledge.ID)
	}
	return knowledge, nil
}

// GetKnowledgeByID retrieves a knowledge entry by its ID
func (s *knowledgeService) GetKnowledgeByID(ctx context.Context, id string) (*types.Knowledge, error) {
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"knowledge_id": id,
			"tenant_id":    tenantID,
		})
		return nil, err
	}

	logger.Infof(ctx, "Knowledge retrieved successfully, ID: %s, type: %s", knowledge.ID, knowledge.Type)
	return knowledge, nil
}

// ListKnowledgeByKnowledgeBaseID returns all knowledge entries in a knowledge base
func (s *knowledgeService) ListKnowledgeByKnowledgeBaseID(ctx context.Context,
	kbID string,
) ([]*types.Knowledge, error) {
	return s.repo.ListKnowledgeByKnowledgeBaseID(ctx, ctx.Value(types.TenantIDContextKey).(uint64), kbID)
}

// ListPagedKnowledgeByKnowledgeBaseID returns paginated knowledge entries in a knowledge base
func (s *knowledgeService) ListPagedKnowledgeByKnowledgeBaseID(ctx context.Context,
	kbID string, page *types.Pagination, tagID string, keyword string, fileType string,
) (*types.PageResult, error) {
	knowledges, total, err := s.repo.ListPagedKnowledgeByKnowledgeBaseID(ctx,
		ctx.Value(types.TenantIDContextKey).(uint64), kbID, page, tagID, keyword, fileType)
	if err != nil {
		return nil, err
	}

	return types.NewPageResult(total, page, knowledges), nil
}

// DeleteKnowledge deletes a knowledge entry and all related resources
func (s *knowledgeService) DeleteKnowledge(ctx context.Context, id string) error {
	// Get the knowledge entry
	knowledge, err := s.repo.GetKnowledgeByID(ctx, ctx.Value(types.TenantIDContextKey).(uint64), id)
	if err != nil {
		return err
	}

	// Mark as deleting first to prevent async task conflicts
	// This ensures that any running async tasks will detect the deletion and abort
	originalStatus := knowledge.ParseStatus
	knowledge.ParseStatus = types.ParseStatusDeleting
	knowledge.UpdatedAt = time.Now()
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge failed to mark as deleting")
		// Continue with deletion even if marking fails
	} else {
		logger.Infof(ctx, "Marked knowledge %s as deleting (previous status: %s)", id, originalStatus)
	}

	wg := errgroup.Group{}
	// Delete knowledge embeddings from vector store
	wg.Go(func() error {
		tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
		retrieveEngine, err := retriever.NewCompositeRetrieveEngine(
			s.retrieveEngine,
			tenantInfo.GetEffectiveEngines(),
		)
		if err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete knowledge embedding failed")
			return err
		}
		embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, knowledge.EmbeddingModelID)
		if err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete knowledge embedding failed")
			return err
		}
		if err := retrieveEngine.DeleteByKnowledgeIDList(ctx, []string{knowledge.ID}, embeddingModel.GetDimensions(), knowledge.Type); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete knowledge embedding failed")
			return err
		}
		return nil
	})

	// Delete all chunks associated with this knowledge
	wg.Go(func() error {
		if err := s.chunkService.DeleteChunksByKnowledgeID(ctx, knowledge.ID); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete chunks failed")
			return err
		}
		return nil
	})

	// Delete the physical file if it exists
	wg.Go(func() error {
		if knowledge.FilePath != "" {
			if err := s.fileSvc.DeleteFile(ctx, knowledge.FilePath); err != nil {
				logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete file failed")
			}
		}
		tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
		tenantInfo.StorageUsed -= knowledge.StorageSize
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, -knowledge.StorageSize); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge update tenant storage used failed")
		}
		return nil
	})

	// Delete the knowledge graph
	wg.Go(func() error {
		namespace := types.NameSpace{KnowledgeBase: knowledge.KnowledgeBaseID, Knowledge: knowledge.ID}
		if err := s.graphEngine.DelGraph(ctx, []types.NameSpace{namespace}); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete knowledge graph failed")
			return err
		}
		return nil
	})

	if err = wg.Wait(); err != nil {
		return err
	}
	// Delete the knowledge entry itself from the database
	return s.repo.DeleteKnowledge(ctx, ctx.Value(types.TenantIDContextKey).(uint64), id)
}

// DeleteKnowledgeList deletes a knowledge entry and all related resources
func (s *knowledgeService) DeleteKnowledgeList(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	// 1. Get the knowledge entry
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	knowledgeList, err := s.repo.GetKnowledgeBatch(ctx, tenantInfo.ID, ids)
	if err != nil {
		return err
	}

	// Mark all as deleting first to prevent async task conflicts
	for _, knowledge := range knowledgeList {
		knowledge.ParseStatus = types.ParseStatusDeleting
		knowledge.UpdatedAt = time.Now()
		if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
			logger.GetLogger(ctx).WithField("error", err).WithField("knowledge_id", knowledge.ID).
				Errorf("DeleteKnowledgeList failed to mark as deleting")
			// Continue with deletion even if marking fails
		}
	}
	logger.Infof(ctx, "Marked %d knowledge entries as deleting", len(knowledgeList))

	wg := errgroup.Group{}
	// 2. Delete knowledge embeddings from vector store
	wg.Go(func() error {
		tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
		retrieveEngine, err := retriever.NewCompositeRetrieveEngine(
			s.retrieveEngine,
			tenantInfo.GetEffectiveEngines(),
		)
		if err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete knowledge embedding failed")
			return err
		}
		// Group by EmbeddingModelID and Type
		type groupKey struct {
			EmbeddingModelID string
			Type             string
		}
		group := map[groupKey][]string{}
		for _, knowledge := range knowledgeList {
			key := groupKey{EmbeddingModelID: knowledge.EmbeddingModelID, Type: knowledge.Type}
			group[key] = append(group[key], knowledge.ID)
		}
		for key, knowledgeIDs := range group {
			embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, key.EmbeddingModelID)
			if err != nil {
				logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge get embedding model failed")
				return err
			}
			if err := retrieveEngine.DeleteByKnowledgeIDList(ctx, knowledgeIDs, embeddingModel.GetDimensions(), key.Type); err != nil {
				logger.GetLogger(ctx).
					WithField("error", err).
					Errorf("DeleteKnowledge delete knowledge embedding failed")
				return err
			}
		}
		return nil
	})

	// 3. Delete all chunks associated with this knowledge
	wg.Go(func() error {
		if err := s.chunkService.DeleteByKnowledgeList(ctx, ids); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete chunks failed")
			return err
		}
		return nil
	})

	// 4. Delete the physical file if it exists
	wg.Go(func() error {
		storageAdjust := int64(0)
		for _, knowledge := range knowledgeList {
			if knowledge.FilePath != "" {
				if err := s.fileSvc.DeleteFile(ctx, knowledge.FilePath); err != nil {
					logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete file failed")
				}
			}
			storageAdjust -= knowledge.StorageSize
		}
		tenantInfo.StorageUsed += storageAdjust
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, storageAdjust); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge update tenant storage used failed")
		}
		return nil
	})

	// Delete the knowledge graph
	wg.Go(func() error {
		namespaces := []types.NameSpace{}
		for _, knowledge := range knowledgeList {
			namespaces = append(
				namespaces,
				types.NameSpace{KnowledgeBase: knowledge.KnowledgeBaseID, Knowledge: knowledge.ID},
			)
		}
		if err := s.graphEngine.DelGraph(ctx, namespaces); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Errorf("DeleteKnowledge delete knowledge graph failed")
			return err
		}
		return nil
	})

	if err = wg.Wait(); err != nil {
		return err
	}
	// 5. Delete the knowledge entry itself from the database
	return s.repo.DeleteKnowledgeList(ctx, tenantInfo.ID, ids)
}

func (s *knowledgeService) cloneKnowledge(
	ctx context.Context,
	src *types.Knowledge,
	targetKB *types.KnowledgeBase,
) (err error) {
	if src.ParseStatus != "completed" {
		logger.GetLogger(ctx).WithField("knowledge_id", src.ID).Errorf("MoveKnowledge parse status is not completed")
		return nil
	}
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	dst := &types.Knowledge{
		ID:               uuid.New().String(),
		TenantID:         targetKB.TenantID,
		KnowledgeBaseID:  targetKB.ID,
		Type:             src.Type,
		Title:            src.Title,
		Description:      src.Description,
		Source:           src.Source,
		ParseStatus:      "processing",
		EnableStatus:     "disabled",
		EmbeddingModelID: targetKB.EmbeddingModelID,
		FileName:         src.FileName,
		FileType:         src.FileType,
		FileSize:         src.FileSize,
		FileHash:         src.FileHash,
		FilePath:         src.FilePath,
		StorageSize:      src.StorageSize,
		Metadata:         src.Metadata,
	}
	defer func() {
		if err != nil {
			dst.ParseStatus = "failed"
			dst.ErrorMessage = err.Error()
			_ = s.repo.UpdateKnowledge(ctx, dst)
			logger.GetLogger(ctx).WithField("error", err).Errorf("MoveKnowledge failed to move knowledge")
		} else {
			dst.ParseStatus = "completed"
			dst.EnableStatus = "enabled"
			_ = s.repo.UpdateKnowledge(ctx, dst)
			logger.GetLogger(ctx).WithField("knowledge_id", dst.ID).Infof("MoveKnowledge move knowledge successfully")
		}
	}()

	if err = s.repo.CreateKnowledge(ctx, dst); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("MoveKnowledge create knowledge failed")
		return
	}
	tenantInfo.StorageUsed += dst.StorageSize
	if err = s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, dst.StorageSize); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("MoveKnowledge update tenant storage used failed")
		return
	}
	if err = s.CloneChunk(ctx, src, dst); err != nil {
		logger.GetLogger(ctx).WithField("knowledge_id", dst.ID).
			WithField("error", err).Errorf("MoveKnowledge move chunks failed")
		return
	}
	return
}

// processDocumentFromPassage handles asynchronous processing of text passages
func (s *knowledgeService) processDocumentFromPassage(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, passage []string,
) {
	// Update status to processing
	knowledge.ParseStatus = "processing"
	knowledge.UpdatedAt = time.Now()
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return
	}

	// Convert passages to chunks
	chunks := make([]*proto.Chunk, 0, len(passage))
	start, end := 0, 0
	for i, p := range passage {
		if p == "" {
			continue
		}
		end += len([]rune(p))
		chunk := &proto.Chunk{
			Content: p,
			Seq:     int32(i),
			Start:   int32(start),
			End:     int32(end),
		}
		start = end
		chunks = append(chunks, chunk)
	}
	// Process and store chunks
	s.processChunks(ctx, kb, knowledge, chunks)
}

// ProcessChunksOptions contains options for processing chunks
type ProcessChunksOptions struct {
	EnableQuestionGeneration bool
	QuestionCount            int
}

// processChunks processes chunks and creates embeddings for knowledge content
func (s *knowledgeService) processChunks(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, chunks []*proto.Chunk,
	opts ...ProcessChunksOptions,
) {
	// Get options
	var options ProcessChunksOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	ctx, span := tracing.ContextWithSpan(ctx, "knowledgeService.processChunks")
	defer span.End()
	span.SetAttributes(
		attribute.Int("tenant_id", int(knowledge.TenantID)),
		attribute.String("knowledge_base_id", knowledge.KnowledgeBaseID),
		attribute.String("knowledge_id", knowledge.ID),
		attribute.String("embedding_model_id", kb.EmbeddingModelID),
		attribute.Int("chunk_count", len(chunks)),
	)

	// Check if knowledge is being deleted before processing
	if s.isKnowledgeDeleting(ctx, knowledge.TenantID, knowledge.ID) {
		logger.Infof(ctx, "Knowledge is being deleted, aborting chunk processing: %s", knowledge.ID)
		span.AddEvent("aborted: knowledge is being deleted")
		return
	}

	// Get embedding model for vectorization
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("processChunks get embedding model failed")
		span.RecordError(err)
		return
	}

	logger.Infof(ctx, "Cleaning up existing chunks and index data for knowledge: %s", knowledge.ID)

	if err := s.chunkService.DeleteChunksByKnowledgeID(ctx, knowledge.ID); err != nil {
		logger.Warnf(ctx, "Failed to delete existing chunks (may not exist): %v", err)
	}

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err == nil {
		if err := retrieveEngine.DeleteByKnowledgeIDList(ctx, []string{knowledge.ID}, embeddingModel.GetDimensions(), knowledge.Type); err != nil {
			logger.Warnf(ctx, "Failed to delete existing index data (may not exist): %v", err)
		} else {
			logger.Infof(ctx, "Successfully deleted existing index data for knowledge: %s", knowledge.ID)
		}
	}

	namespace := types.NameSpace{KnowledgeBase: knowledge.KnowledgeBaseID, Knowledge: knowledge.ID}
	if err := s.graphEngine.DelGraph(ctx, []types.NameSpace{namespace}); err != nil {
		logger.Warnf(ctx, "Failed to delete existing graph data (may not exist): %v", err)
	}

	logger.Infof(ctx, "Cleanup completed, starting to process new chunks")

	logger.Infof(ctx, "[DocReader] ========== 파싱 결과 개요 ==========")
	logger.Infof(ctx, "[DocReader] 지식 ID: %s, 지식베이스 ID: %s", knowledge.ID, knowledge.KnowledgeBaseID)
	logger.Infof(ctx, "[DocReader] 총 청크 수: %d", len(chunks))

	totalImages := 0
	chunksWithImages := 0
	for _, chunkData := range chunks {
		if len(chunkData.Images) > 0 {
			chunksWithImages++
			totalImages += len(chunkData.Images)
		}
	}
	logger.Infof(ctx, "[DocReader] 이미지 포함 청크 수: %d, 총 이미지 수: %d", chunksWithImages, totalImages)

	for idx, chunkData := range chunks {
		contentPreview := chunkData.Content
		if len(contentPreview) > 200 {
			contentPreview = contentPreview[:200] + "..."
		}
		logger.Infof(ctx, "[DocReader] 청크 #%d (seq=%d): 내용 길이=%d, 이미지 수=%d, 범위=[%d-%d]",
			idx, chunkData.Seq, len(chunkData.Content), len(chunkData.Images), chunkData.Start, chunkData.End)
		logger.Debugf(ctx, "[DocReader] 청크 #%d 내용 미리보기: %s", idx, contentPreview)

		for imgIdx, img := range chunkData.Images {
			logger.Infof(ctx, "[DocReader]   이미지 #%d: URL=%s", imgIdx, img.Url)
			logger.Infof(ctx, "[DocReader]   이미지 #%d: 원본URL=%s", imgIdx, img.OriginalUrl)
			if img.Caption != "" {
				captionPreview := img.Caption
				if len(captionPreview) > 100 {
					captionPreview = captionPreview[:100] + "..."
				}
				logger.Infof(ctx, "[DocReader]   이미지 #%d: 설명=%s", imgIdx, captionPreview)
			}
			if img.OcrText != "" {
				ocrPreview := img.OcrText
				if len(ocrPreview) > 100 {
					ocrPreview = ocrPreview[:100] + "..."
				}
				logger.Infof(ctx, "[DocReader]   이미지 #%d: OCR텍스트=%s", imgIdx, ocrPreview)
			}
			logger.Infof(ctx, "[DocReader]   이미지 #%d: 위치=[%d-%d]", imgIdx, img.Start, img.End)
		}
	}
	logger.Infof(ctx, "[DocReader] ========== 파싱 결과 개요 끝 ==========")

	// Create chunk objects from proto chunks
	maxSeq := 0

	imageChunkCount := 0
	for _, chunkData := range chunks {
		if len(chunkData.Images) > 0 {
			imageChunkCount += len(chunkData.Images) * 2
		}
		if int(chunkData.Seq) > maxSeq {
			maxSeq = int(chunkData.Seq)
		}
	}

	insertChunks := make([]*types.Chunk, 0, len(chunks)+imageChunkCount)

	for _, chunkData := range chunks {
		if strings.TrimSpace(chunkData.Content) == "" {
			continue
		}

		textChunk := &types.Chunk{
			ID:              uuid.New().String(),
			TenantID:        knowledge.TenantID,
			KnowledgeID:     knowledge.ID,
			KnowledgeBaseID: knowledge.KnowledgeBaseID,
			Content:         chunkData.Content,
			ChunkIndex:      int(chunkData.Seq),
			IsEnabled:       true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			StartAt:         int(chunkData.Start),
			EndAt:           int(chunkData.End),
			ChunkType:       types.ChunkTypeText,
		}
		var chunkImages []types.ImageInfo
		insertChunks = append(insertChunks, textChunk)

		if len(chunkData.Images) > 0 {
			logger.GetLogger(ctx).Infof("Processing %d images in chunk #%d", len(chunkData.Images), chunkData.Seq)

			for i, img := range chunkData.Images {
				imageInfo := types.ImageInfo{
					URL:         img.Url,
					OriginalURL: img.OriginalUrl,
					StartPos:    int(img.Start),
					EndPos:      int(img.End),
					OCRText:     img.OcrText,
					Caption:     img.Caption,
				}
				chunkImages = append(chunkImages, imageInfo)

				imageInfoJSON, err := json.Marshal([]types.ImageInfo{imageInfo})
				if err != nil {
					logger.GetLogger(ctx).WithField("error", err).Errorf("Failed to marshal image info to JSON")
					continue
				}

				if img.OcrText != "" {
					ocrChunk := &types.Chunk{
						ID:              uuid.New().String(),
						TenantID:        knowledge.TenantID,
						KnowledgeID:     knowledge.ID,
						KnowledgeBaseID: knowledge.KnowledgeBaseID,
						Content:         img.OcrText,
						ChunkIndex:      maxSeq + i*100 + 1,
						IsEnabled:       true,
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
						StartAt:         int(img.Start),
						EndAt:           int(img.End),
						ChunkType:       types.ChunkTypeImageOCR,
						ParentChunkID:   textChunk.ID,
						ImageInfo:       string(imageInfoJSON),
					}
					insertChunks = append(insertChunks, ocrChunk)
					logger.GetLogger(ctx).Infof("Created OCR chunk for image %d in chunk #%d", i, chunkData.Seq)
				}

				if img.Caption != "" {
					captionChunk := &types.Chunk{
						ID:              uuid.New().String(),
						TenantID:        knowledge.TenantID,
						KnowledgeID:     knowledge.ID,
						KnowledgeBaseID: knowledge.KnowledgeBaseID,
						Content:         img.Caption,
						ChunkIndex:      maxSeq + i*100 + 2,
						IsEnabled:       true,
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
						StartAt:         int(img.Start),
						EndAt:           int(img.End),
						ChunkType:       types.ChunkTypeImageCaption,
						ParentChunkID:   textChunk.ID,
						ImageInfo:       string(imageInfoJSON),
					}
					insertChunks = append(insertChunks, captionChunk)
					logger.GetLogger(ctx).Infof("Created caption chunk for image %d in chunk #%d", i, chunkData.Seq)
				}
			}

			imageInfoJSON, err := json.Marshal(chunkImages)
			if err != nil {
				logger.GetLogger(ctx).WithField("error", err).Errorf("Failed to marshal image info to JSON")
				continue
			}
			textChunk.ImageInfo = string(imageInfoJSON)
		}
	}

	// Sort chunks by index for proper ordering
	sort.Slice(insertChunks, func(i, j int) bool {
		return insertChunks[i].ChunkIndex < insertChunks[j].ChunkIndex
	})

	textChunks := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range insertChunks {
		if chunk.ChunkType == types.ChunkTypeText {
			textChunks = append(textChunks, chunk)
		}
	}

	for i, chunk := range textChunks {
		if i > 0 {
			textChunks[i-1].NextChunkID = chunk.ID
		}
		if i < len(textChunks)-1 {
			textChunks[i+1].PreChunkID = chunk.ID
		}
	}

	// Create index information for each chunk (without generated questions for now)
	indexInfoList := make([]*types.IndexInfo, 0, len(insertChunks))
	for _, chunk := range insertChunks {
		// Add original chunk content to index
		indexInfoList = append(indexInfoList, &types.IndexInfo{
			Content:         chunk.Content,
			SourceID:        chunk.ID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         chunk.ID,
			KnowledgeID:     knowledge.ID,
			KnowledgeBaseID: knowledge.KnowledgeBaseID,
		})
	}

	// Initialize retrieval engine

	// Calculate storage size required for embeddings
	span.AddEvent("estimate storage size")
	totalStorageSize := retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfoList)
	if tenantInfo.StorageQuota > 0 {
		// Re-fetch tenant storage information
		tenantInfo, err = s.tenantRepo.GetTenantByID(ctx, tenantInfo.ID)
		if err != nil {
			knowledge.ParseStatus = types.ParseStatusFailed
			knowledge.ErrorMessage = err.Error()
			knowledge.UpdatedAt = time.Now()
			s.repo.UpdateKnowledge(ctx, knowledge)
			span.RecordError(err)
			return
		}
		// Check if there's enough storage quota available
		if tenantInfo.StorageUsed+totalStorageSize > tenantInfo.StorageQuota {
			knowledge.ParseStatus = types.ParseStatusFailed
			knowledge.ErrorMessage = "저장 공간이 부족합니다"
			knowledge.UpdatedAt = time.Now()
			s.repo.UpdateKnowledge(ctx, knowledge)
			span.RecordError(errors.New("storage quota exceeded"))
			return
		}
	}

	// Check again if knowledge is being deleted before writing to database
	if s.isKnowledgeDeleting(ctx, knowledge.TenantID, knowledge.ID) {
		logger.Infof(ctx, "Knowledge is being deleted, aborting before saving chunks: %s", knowledge.ID)
		span.AddEvent("aborted: knowledge is being deleted before saving")
		return
	}

	// Save chunks to database
	span.AddEvent("create chunks")
	if err := s.chunkService.CreateChunks(ctx, insertChunks); err != nil {
		knowledge.ParseStatus = types.ParseStatusFailed
		knowledge.ErrorMessage = err.Error()
		knowledge.UpdatedAt = time.Now()
		s.repo.UpdateKnowledge(ctx, knowledge)
		span.RecordError(err)
		return
	}

	// Check again before batch indexing (this is a heavy operation)
	if s.isKnowledgeDeleting(ctx, knowledge.TenantID, knowledge.ID) {
		logger.Infof(ctx, "Knowledge is being deleted, cleaning up and aborting before indexing: %s", knowledge.ID)
		// Clean up the chunks we just created
		if err := s.chunkService.DeleteChunksByKnowledgeID(ctx, knowledge.ID); err != nil {
			logger.Warnf(ctx, "Failed to cleanup chunks after deletion detected: %v", err)
		}
		span.AddEvent("aborted: knowledge is being deleted before indexing")
		return
	}

	span.AddEvent("batch index")
	err = retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoList)
	if err != nil {
		knowledge.ParseStatus = types.ParseStatusFailed
		knowledge.ErrorMessage = err.Error()
		knowledge.UpdatedAt = time.Now()
		s.repo.UpdateKnowledge(ctx, knowledge)

		// delete failed chunks
		if err := s.chunkService.DeleteChunksByKnowledgeID(ctx, knowledge.ID); err != nil {
			logger.Errorf(ctx, "Delete chunks failed: %v", err)
		}

		// delete index
		if err := retrieveEngine.DeleteByKnowledgeIDList(
			ctx, []string{knowledge.ID}, embeddingModel.GetDimensions(), kb.Type,
		); err != nil {
			logger.Errorf(ctx, "Delete index failed: %v", err)
		}
		span.RecordError(err)
		return
	}
	logger.GetLogger(ctx).Infof("processChunks batch index successfully, with %d index", len(indexInfoList))

	logger.Infof(ctx, "processChunks create relationship rag task")
	if kb.ExtractConfig != nil && kb.ExtractConfig.Enabled {
		for _, chunk := range textChunks {
			err := NewChunkExtractTask(ctx, s.task, chunk.TenantID, chunk.ID, kb.SummaryModelID)
			if err != nil {
				logger.GetLogger(ctx).WithField("error", err).Errorf("processChunks create chunk extract task failed")
				span.RecordError(err)
			}
		}
	}

	// Final check before marking as completed - if deleted during processing, don't update status
	if s.isKnowledgeDeleting(ctx, knowledge.TenantID, knowledge.ID) {
		logger.Infof(ctx, "Knowledge was deleted during processing, skipping completion update: %s", knowledge.ID)
		// Clean up the data we just created since the knowledge is being deleted
		if err := s.chunkService.DeleteChunksByKnowledgeID(ctx, knowledge.ID); err != nil {
			logger.Warnf(ctx, "Failed to cleanup chunks after deletion detected: %v", err)
		}
		if err := retrieveEngine.DeleteByKnowledgeIDList(ctx, []string{knowledge.ID}, embeddingModel.GetDimensions(), kb.Type); err != nil {
			logger.Warnf(ctx, "Failed to cleanup index after deletion detected: %v", err)
		}
		span.AddEvent("aborted: knowledge was deleted during processing")
		return
	}

	// Update knowledge status to completed
	knowledge.ParseStatus = types.ParseStatusCompleted
	knowledge.EnableStatus = "enabled"
	knowledge.StorageSize = totalStorageSize
	now := time.Now()
	knowledge.ProcessedAt = &now
	knowledge.UpdatedAt = now

	// Set summary status based on whether summary generation will be triggered
	if len(textChunks) > 0 {
		knowledge.SummaryStatus = types.SummaryStatusPending
	} else {
		knowledge.SummaryStatus = types.SummaryStatusNone
	}

	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("processChunks update knowledge failed")
	}

	// Enqueue question generation task if enabled (async, non-blocking)
	if options.EnableQuestionGeneration && len(textChunks) > 0 {
		questionCount := options.QuestionCount
		if questionCount <= 0 {
			questionCount = 3
		}
		if questionCount > 10 {
			questionCount = 10
		}
		s.enqueueQuestionGenerationTask(ctx, knowledge.KnowledgeBaseID, knowledge.ID, questionCount)
	}

	// Enqueue summary generation task (async, non-blocking)
	if len(textChunks) > 0 {
		s.enqueueSummaryGenerationTask(ctx, knowledge.KnowledgeBaseID, knowledge.ID)
	}

	// Update tenant's storage usage
	tenantInfo.StorageUsed += totalStorageSize
	if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, totalStorageSize); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("processChunks update tenant storage used failed")
	}
	logger.GetLogger(ctx).Infof("processChunks successfully")
}

// GetSummary generates a summary for knowledge content using an AI model
func (s *knowledgeService) getSummary(ctx context.Context,
	summaryModel chat.Chat, knowledge *types.Knowledge, chunks []*types.Chunk,
) (string, error) {
	// Get knowledge info from the first chunk
	if len(chunks) == 0 {
		return "", fmt.Errorf("no chunks provided for summary generation")
	}

	// concat chunk contents
	chunkContents := ""
	allImageInfos := make([]*types.ImageInfo, 0)

	// then, sort chunks by StartAt
	sortedChunks := make([]*types.Chunk, len(chunks))
	copy(sortedChunks, chunks)
	sort.Slice(sortedChunks, func(i, j int) bool {
		return sortedChunks[i].StartAt < sortedChunks[j].StartAt
	})

	// concat chunk contents and collect image infos
	for _, chunk := range sortedChunks {
		if chunk.EndAt > 4096 {
			break
		}
		chunkContents = string([]rune(chunkContents)[:chunk.StartAt]) + chunk.Content
		if chunk.ImageInfo != "" {
			var images []*types.ImageInfo
			if err := json.Unmarshal([]byte(chunk.ImageInfo), &images); err == nil {
				allImageInfos = append(allImageInfos, images...)
			}
		}
	}
	// remove markdown image syntax
	re := regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	chunkContents = re.ReplaceAllString(chunkContents, "")
	// collect all image infos
	if len(allImageInfos) > 0 {
		// add image infos to chunk contents
		var imageAnnotations string
		for _, img := range allImageInfos {
			if img.Caption != "" {
				imageAnnotations += fmt.Sprintf("\n[이미지 설명: %s]", img.Caption)
			}
			if img.OCRText != "" {
				imageAnnotations += fmt.Sprintf("\n[이미지 텍스트: %s]", img.OCRText)
			}
		}

		// concat chunk contents and image annotations
		chunkContents = chunkContents + imageAnnotations
	}

	if len(chunkContents) < 300 {
		return chunkContents, nil
	}

	// Prepare content with metadata for summary generation
	contentWithMetadata := chunkContents

	// Add knowledge metadata if available
	if knowledge != nil {
		metadataIntro := fmt.Sprintf("문서 유형: %s\n파일명: %s\n", knowledge.FileType, knowledge.FileName)

		// Add additional metadata if available
		if knowledge.Type != "" {
			metadataIntro += fmt.Sprintf("지식 유형: %s\n", knowledge.Type)
		}

		// Prepend metadata to content
		contentWithMetadata = metadataIntro + "\n내용:\n" + contentWithMetadata
	}

	// Generate summary using AI model
	thinking := false
	summary, err := summaryModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: s.config.Conversation.GenerateSummaryPrompt,
		},
		{
			Role:    "user",
			Content: contentWithMetadata,
		},
	}, &chat.ChatOptions{
		Temperature: 0.3,
		MaxTokens:   1024,
		Thinking:    &thinking,
	})
	if err != nil {
		logger.GetLogger(ctx).WithField("error", err).Errorf("GetSummary failed")
		return "", err
	}
	logger.GetLogger(ctx).WithField("summary", summary.Content).Infof("GetSummary success")
	return summary.Content, nil
}

// enqueueQuestionGenerationTask enqueues an async task for question generation
func (s *knowledgeService) enqueueQuestionGenerationTask(ctx context.Context,
	kbID, knowledgeID string, questionCount int,
) {
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	payload := types.QuestionGenerationPayload{
		TenantID:        tenantID,
		KnowledgeBaseID: kbID,
		KnowledgeID:     knowledgeID,
		QuestionCount:   questionCount,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal question generation payload: %v", err)
		return
	}

	task := asynq.NewTask(types.TypeQuestionGeneration, payloadBytes, asynq.Queue("low"), asynq.MaxRetry(3))
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue question generation task: %v", err)
		return
	}
	logger.Infof(ctx, "Enqueued question generation task: %s for knowledge: %s", info.ID, knowledgeID)
}

// enqueueSummaryGenerationTask enqueues an async task for summary generation
func (s *knowledgeService) enqueueSummaryGenerationTask(ctx context.Context,
	kbID, knowledgeID string,
) {
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	payload := types.SummaryGenerationPayload{
		TenantID:        tenantID,
		KnowledgeBaseID: kbID,
		KnowledgeID:     knowledgeID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal summary generation payload: %v", err)
		return
	}

	task := asynq.NewTask(types.TypeSummaryGeneration, payloadBytes, asynq.Queue("low"), asynq.MaxRetry(3))
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue summary generation task: %v", err)
		return
	}
	logger.Infof(ctx, "Enqueued summary generation task: %s for knowledge: %s", info.ID, knowledgeID)
}

// ProcessSummaryGeneration handles async summary generation task
func (s *knowledgeService) ProcessSummaryGeneration(ctx context.Context, t *asynq.Task) error {
	var payload types.SummaryGenerationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "Failed to unmarshal summary generation payload: %v", err)
		return nil // Don't retry on unmarshal error
	}

	logger.Infof(ctx, "Processing summary generation for knowledge: %s", payload.KnowledgeID)

	// Set tenant context
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)

	// Get knowledge base
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return nil
	}

	// Get knowledge
	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge: %v", err)
		return nil
	}

	// Update summary status to processing
	knowledge.SummaryStatus = types.SummaryStatusProcessing
	knowledge.UpdatedAt = time.Now()
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		logger.Warnf(ctx, "Failed to update summary status to processing: %v", err)
	}

	// Helper function to mark summary as failed
	markSummaryFailed := func() {
		knowledge.SummaryStatus = types.SummaryStatusFailed
		knowledge.UpdatedAt = time.Now()
		if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
			logger.Warnf(ctx, "Failed to update summary status to failed: %v", err)
		}
	}

	// Get text chunks for this knowledge
	chunks, err := s.chunkService.ListChunksByKnowledgeID(ctx, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chunks: %v", err)
		markSummaryFailed()
		return nil
	}

	// Filter text chunks only
	textChunks := make([]*types.Chunk, 0)
	for _, chunk := range chunks {
		if chunk.ChunkType == types.ChunkTypeText {
			textChunks = append(textChunks, chunk)
		}
	}

	if len(textChunks) == 0 {
		logger.Infof(ctx, "No text chunks found for knowledge: %s", payload.KnowledgeID)
		// Mark as completed since there's nothing to summarize
		knowledge.SummaryStatus = types.SummaryStatusCompleted
		knowledge.UpdatedAt = time.Now()
		s.repo.UpdateKnowledge(ctx, knowledge)
		return nil
	}

	// Sort chunks by ChunkIndex for proper ordering
	sort.Slice(textChunks, func(i, j int) bool {
		return textChunks[i].ChunkIndex < textChunks[j].ChunkIndex
	})

	// Initialize chat model for summary
	chatModel, err := s.modelService.GetChatModel(ctx, kb.SummaryModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chat model: %v", err)
		markSummaryFailed()
		return fmt.Errorf("failed to get chat model: %w", err)
	}

	// Generate summary
	summary, err := s.getSummary(ctx, chatModel, knowledge, textChunks)
	if err != nil {
		logger.Errorf(ctx, "Failed to generate summary for knowledge %s: %v", payload.KnowledgeID, err)
		// Use first chunk content as fallback
		if len(textChunks) > 0 {
			summary = textChunks[0].Content
			if len(summary) > 500 {
				summary = summary[:500]
			}
		}
	}

	// Update knowledge description
	knowledge.Description = summary
	knowledge.SummaryStatus = types.SummaryStatusCompleted
	knowledge.UpdatedAt = time.Now()
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "Failed to update knowledge description: %v", err)
		return fmt.Errorf("failed to update knowledge: %w", err)
	}

	// Create summary chunk and index it
	if strings.TrimSpace(summary) != "" {
		// Get max chunk index
		maxChunkIndex := 0
		for _, chunk := range chunks {
			if chunk.ChunkIndex > maxChunkIndex {
				maxChunkIndex = chunk.ChunkIndex
			}
		}

		summaryChunk := &types.Chunk{
			ID:              uuid.New().String(),
			TenantID:        knowledge.TenantID,
			KnowledgeID:     knowledge.ID,
			KnowledgeBaseID: knowledge.KnowledgeBaseID,
			Content:         fmt.Sprintf("# 문서명\n%s\n\n# 요약\n%s", knowledge.FileName, summary),
			ChunkIndex:      maxChunkIndex + 1,
			IsEnabled:       true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			StartAt:         0,
			EndAt:           0,
			ChunkType:       types.ChunkTypeSummary,
			ParentChunkID:   textChunks[0].ID,
		}

		// Save summary chunk
		if err := s.chunkService.CreateChunks(ctx, []*types.Chunk{summaryChunk}); err != nil {
			logger.Errorf(ctx, "Failed to create summary chunk: %v", err)
			return fmt.Errorf("failed to create summary chunk: %w", err)
		}

		// Index summary chunk
		tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
		if err != nil {
			logger.Errorf(ctx, "Failed to get tenant info: %v", err)
			return fmt.Errorf("failed to get tenant info: %w", err)
		}
		ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

		retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
		if err != nil {
			logger.Errorf(ctx, "Failed to init retrieve engine: %v", err)
			return fmt.Errorf("failed to init retrieve engine: %w", err)
		}

		embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
		if err != nil {
			logger.Errorf(ctx, "Failed to get embedding model: %v", err)
			return fmt.Errorf("failed to get embedding model: %w", err)
		}

		indexInfo := []*types.IndexInfo{{
			Content:         summaryChunk.Content,
			SourceID:        summaryChunk.ID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         summaryChunk.ID,
			KnowledgeID:     knowledge.ID,
			KnowledgeBaseID: knowledge.KnowledgeBaseID,
		}}

		if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfo); err != nil {
			logger.Errorf(ctx, "Failed to index summary chunk: %v", err)
			return fmt.Errorf("failed to index summary chunk: %w", err)
		}

		logger.Infof(ctx, "Successfully created and indexed summary chunk for knowledge: %s", payload.KnowledgeID)
	}

	logger.Infof(ctx, "Successfully generated summary for knowledge: %s", payload.KnowledgeID)
	return nil
}

// ProcessQuestionGeneration handles async question generation task
func (s *knowledgeService) ProcessQuestionGeneration(ctx context.Context, t *asynq.Task) error {
	ctx, span := tracing.ContextWithSpan(ctx, "knowledgeService.ProcessQuestionGeneration")
	defer span.End()

	var payload types.QuestionGenerationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "Failed to unmarshal question generation payload: %v", err)
		return nil // Don't retry on unmarshal error
	}

	logger.Infof(ctx, "Processing question generation for knowledge: %s", payload.KnowledgeID)

	// Set tenant context
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)

	// Get knowledge base
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		return nil
	}

	// Get knowledge
	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge: %v", err)
		return nil
	}

	// Get text chunks for this knowledge
	chunks, err := s.chunkService.ListChunksByKnowledgeID(ctx, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chunks: %v", err)
		return nil
	}

	// Filter text chunks only
	textChunks := make([]*types.Chunk, 0)
	for _, chunk := range chunks {
		if chunk.ChunkType == types.ChunkTypeText {
			textChunks = append(textChunks, chunk)
		}
	}

	if len(textChunks) == 0 {
		logger.Infof(ctx, "No text chunks found for knowledge: %s", payload.KnowledgeID)
		return nil
	}

	// Sort chunks by StartAt for context building
	sort.Slice(textChunks, func(i, j int) bool {
		return textChunks[i].StartAt < textChunks[j].StartAt
	})

	// Initialize chat model
	chatModel, err := s.modelService.GetChatModel(ctx, kb.SummaryModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chat model: %v", err)
		return fmt.Errorf("failed to get chat model: %w", err)
	}

	// Initialize embedding model and retrieval engine
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get embedding model: %v", err)
		return fmt.Errorf("failed to get embedding model: %w", err)
	}

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get tenant info: %v", err)
		return fmt.Errorf("failed to get tenant info: %w", err)
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		logger.Errorf(ctx, "Failed to init retrieve engine: %v", err)
		return fmt.Errorf("failed to init retrieve engine: %w", err)
	}

	questionCount := payload.QuestionCount
	if questionCount <= 0 {
		questionCount = 3
	}
	if questionCount > 10 {
		questionCount = 10
	}

	// Generate questions for each chunk with context
	var indexInfoList []*types.IndexInfo
	for i, chunk := range textChunks {
		// Build context from adjacent chunks
		var prevContent, nextContent string
		if i > 0 {
			prevContent = textChunks[i-1].Content
			// Limit context size
			if len(prevContent) > 500 {
				prevContent = prevContent[len(prevContent)-500:]
			}
		}
		if i < len(textChunks)-1 {
			nextContent = textChunks[i+1].Content
			// Limit context size
			if len(nextContent) > 500 {
				nextContent = nextContent[:500]
			}
		}

		questions, err := s.generateQuestionsWithContext(ctx, chatModel, chunk.Content, prevContent, nextContent, knowledge.Title, questionCount)
		if err != nil {
			logger.Warnf(ctx, "Failed to generate questions for chunk %s: %v", chunk.ID, err)
			continue
		}

		if len(questions) == 0 {
			continue
		}

		// Update chunk metadata with unique IDs for each question
		generatedQuestions := make([]types.GeneratedQuestion, len(questions))
		for j, question := range questions {
			questionID := fmt.Sprintf("q%d", time.Now().UnixNano()+int64(j))
			generatedQuestions[j] = types.GeneratedQuestion{
				ID:       questionID,
				Question: question,
			}
		}
		meta := &types.DocumentChunkMetadata{
			GeneratedQuestions: generatedQuestions,
		}
		if err := chunk.SetDocumentMetadata(meta); err != nil {
			logger.Warnf(ctx, "Failed to set document metadata for chunk %s: %v", chunk.ID, err)
			continue
		}

		// Update chunk in database
		if err := s.chunkService.UpdateChunk(ctx, chunk); err != nil {
			logger.Warnf(ctx, "Failed to update chunk %s: %v", chunk.ID, err)
			continue
		}

		// Create index entries for generated questions
		for _, gq := range generatedQuestions {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, gq.ID)
			indexInfoList = append(indexInfoList, &types.IndexInfo{
				Content:         gq.Question,
				SourceID:        sourceID,
				SourceType:      types.ChunkSourceType,
				ChunkID:         chunk.ID,
				KnowledgeID:     knowledge.ID,
				KnowledgeBaseID: knowledge.KnowledgeBaseID,
			})
		}
		logger.Debugf(ctx, "Generated %d questions for chunk %s", len(questions), chunk.ID)
	}

	// Index generated questions
	if len(indexInfoList) > 0 {
		if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoList); err != nil {
			logger.Errorf(ctx, "Failed to index generated questions: %v", err)
			return fmt.Errorf("failed to index questions: %w", err)
		}
		logger.Infof(ctx, "Successfully indexed %d generated questions for knowledge: %s", len(indexInfoList), payload.KnowledgeID)
	}

	return nil
}

// generateQuestionsWithContext generates questions for a chunk with surrounding context
func (s *knowledgeService) generateQuestionsWithContext(ctx context.Context,
	chatModel chat.Chat, content, prevContent, nextContent, docName string, questionCount int,
) ([]string, error) {
	if content == "" || questionCount <= 0 {
		return nil, nil
	}

	// Build prompt with context
	prompt := s.config.Conversation.GenerateQuestionsPrompt
	if prompt == "" {
		prompt = defaultQuestionGenerationPrompt
	}

	// Build context section
	var contextSection string
	if prevContent != "" || nextContent != "" {
		contextSection = "## 컨텍스트 정보 (참고용, 주요 내용 이해에 도움)\n"
		if prevContent != "" {
			contextSection += fmt.Sprintf("【앞 내용】%s\n", prevContent)
		}
		if nextContent != "" {
			contextSection += fmt.Sprintf("【뒤 내용】%s\n", nextContent)
		}
		contextSection += "\n"
	}

	// Replace placeholders
	prompt = strings.ReplaceAll(prompt, "{{.QuestionCount}}", fmt.Sprintf("%d", questionCount))
	prompt = strings.ReplaceAll(prompt, "{{.Content}}", content)
	prompt = strings.ReplaceAll(prompt, "{{.Context}}", contextSection)
	prompt = strings.ReplaceAll(prompt, "{{.DocName}}", docName)

	thinking := false
	response, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}, &chat.ChatOptions{
		Temperature: 0.7,
		MaxTokens:   512,
		Thinking:    &thinking,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate questions: %w", err)
	}

	// Parse response
	lines := strings.Split(response.Content, "\n")
	questions := make([]string, 0, questionCount)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "0123456789.-*) ")
		line = strings.TrimSpace(line)
		if line != "" && len(line) > 5 {
			questions = append(questions, line)
			if len(questions) >= questionCount {
				break
			}
		}
	}

	return questions, nil
}

// Default prompt for question generation with context support
const defaultQuestionGenerationPrompt = `당신은 전문 질문 생성 어시스턴트입니다. 주어진 【주요 내용】을 바탕으로 사용자가 물어볼 수 있는 관련 질문을 생성하는 것이 당신의 임무입니다.

{{.Context}}
## 주요 내용 (이 내용을 바탕으로 질문을 생성해 주세요)
문서명: {{.DocName}}
문서 내용:
{{.Content}}

## 핵심 요구사항
- 생성된 질문은 반드시 【주요 내용】과 직접 관련되어야 합니다
- 질문에 대명사나 지시어를 사용하지 말고 구체적인 명칭으로 대체하세요
- 질문은 완전하고 독립적이어야 하며, 맥락 없이도 이해될 수 있어야 합니다
- 질문은 사용자가 실제 상황에서 할 수 있는 자연스러운 질문이어야 합니다
- 질문은 다양하게, 내용의 여러 측면을 커버해야 합니다
- 각 질문은 간결하고 명확해야 하며, 30자 이내로 작성하세요
- 생성할 질문 수: {{.QuestionCount}} 개

## 질문 유형 제안
- 정의형: ~은 무엇인가요? ~이란?
- 원인형: 왜 ~인가요? ~의 이유는 무엇인가요?
- 방법형: 어떻게 ~하나요? ~하는 방법은?
- 비교형: ~과 ~의 차이는 무엇인가요?
- 적용형: ~은 어떤 상황에서 사용되나요?

## 출력 형식
질문 목록을 한 줄에 하나씩 바로 출력하고, 번호나 다른 접두어를 붙이지 마세요.`

// GetKnowledgeFile retrieves the physical file associated with a knowledge entry
func (s *knowledgeService) GetKnowledgeFile(ctx context.Context, id string) (io.ReadCloser, string, error) {
	// Get knowledge record
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, id)
	if err != nil {
		return nil, "", err
	}

	// Get the file from storage
	file, err := s.fileSvc.GetFile(ctx, knowledge.FilePath)
	if err != nil {
		return nil, "", err
	}

	return file, knowledge.FileName, nil
}

func (s *knowledgeService) UpdateKnowledge(ctx context.Context, knowledge *types.Knowledge) error {
	record, err := s.repo.GetKnowledgeByID(ctx, ctx.Value(types.TenantIDContextKey).(uint64), knowledge.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge record: %v", err)
		return err
	}
	// if need other fields update, please add here
	if knowledge.Title != "" {
		record.Title = knowledge.Title
	}

	// Update knowledge record in the repository
	if err := s.repo.UpdateKnowledge(ctx, record); err != nil {
		logger.Errorf(ctx, "Failed to update knowledge: %v", err)
		return err
	}
	logger.Infof(ctx, "Knowledge updated successfully, ID: %s", knowledge.ID)
	return nil
}

// UpdateManualKnowledge updates manual Markdown knowledge content.
func (s *knowledgeService) UpdateManualKnowledge(ctx context.Context,
	knowledgeID string, payload *types.ManualKnowledgePayload,
) (*types.Knowledge, error) {
	logger.Info(ctx, "Start updating manual knowledge entry")
	if payload == nil {
		return nil, werrors.NewBadRequestError("요청 내용은 비워둘 수 없습니다")
	}

	cleanContent := secutils.CleanMarkdown(payload.Content)
	if strings.TrimSpace(cleanContent) == "" {
		return nil, werrors.NewValidationError("내용은 비워둘 수 없습니다")
	}
	if len([]rune(cleanContent)) > manualContentMaxLength {
		return nil, werrors.NewValidationError(fmt.Sprintf("내용 길이 제한 초과 (최대 %d자)", manualContentMaxLength))
	}

	safeTitle, ok := secutils.ValidateInput(payload.Title)
	if !ok {
		return nil, werrors.NewValidationError("제목에 허용되지 않는 문자가 포함되어 있거나 길이 제한을 초과했습니다")
	}

	status := strings.ToLower(strings.TrimSpace(payload.Status))
	if status == "" {
		status = types.ManualKnowledgeStatusDraft
	}
	if status != types.ManualKnowledgeStatusDraft && status != types.ManualKnowledgeStatusPublish {
		return nil, werrors.NewValidationError("상태는 draft 또는 publish만 지원됩니다")
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	existing, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to load knowledge: %v", err)
		return nil, err
	}
	if !existing.IsManual() {
		return nil, werrors.NewBadRequestError("직접 입력한 지식만 온라인 편집을 지원합니다")
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, existing.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base for manual update: %v", err)
		return nil, err
	}

	var version int
	if meta, err := existing.ManualMetadata(); err == nil && meta != nil {
		version = meta.Version + 1
	} else {
		version = 1
	}

	meta := types.NewManualKnowledgeMetadata(cleanContent, status, version)
	if err := existing.SetManualMetadata(meta); err != nil {
		logger.Errorf(ctx, "Failed to set manual metadata during update: %v", err)
		return nil, err
	}

	if safeTitle != "" {
		existing.Title = safeTitle
	} else if existing.Title == "" {
		existing.Title = fmt.Sprintf("직접입력지식-%s", time.Now().Format("20060102-150405"))
	}
	existing.FileName = ensureManualFileName(existing.Title)
	existing.FileType = types.KnowledgeTypeManual
	existing.Type = types.KnowledgeTypeManual
	existing.Source = types.KnowledgeTypeManual
	existing.EnableStatus = "disabled"
	existing.UpdatedAt = time.Now()

	if err := s.cleanupKnowledgeResources(ctx, existing); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"knowledge_id": knowledgeID,
		})
		return nil, err
	}

	existing.EmbeddingModelID = kb.EmbeddingModelID

	if status == types.ManualKnowledgeStatusDraft {
		existing.ParseStatus = types.ManualKnowledgeStatusDraft
		existing.Description = ""
		existing.ProcessedAt = nil

		if err := s.repo.UpdateKnowledge(ctx, existing); err != nil {
			logger.Errorf(ctx, "Failed to persist manual draft: %v", err)
			return nil, err
		}
		return existing, nil
	}

	existing.ParseStatus = "pending"
	existing.Description = ""
	existing.ProcessedAt = nil

	if err := s.repo.UpdateKnowledge(ctx, existing); err != nil {
		logger.Errorf(ctx, "Failed to persist manual knowledge before indexing: %v", err)
		return nil, err
	}

	logger.Infof(ctx, "Manual knowledge updated, scheduling indexing, ID: %s", existing.ID)
	s.triggerManualProcessing(ctx, kb, existing, cleanContent, false)
	return existing, nil
}

// isValidFileType checks if a file type is supported
func isValidFileType(filename string) bool {
	switch strings.ToLower(getFileType(filename)) {
	case "pdf", "txt", "docx", "doc", "md", "markdown", "png", "jpg", "jpeg", "gif", "csv", "xlsx", "xls":
		return true
	default:
		return false
	}
}

// getFileType extracts the file extension from a filename
func getFileType(filename string) string {
	ext := strings.Split(filename, ".")
	if len(ext) < 2 {
		return "unknown"
	}
	return ext[len(ext)-1]
}

// isValidURL verifies if a URL is valid
func isValidURL(url string) bool {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return true
	}
	return false
}

// GetKnowledgeBatch retrieves multiple knowledge entries by their IDs
func (s *knowledgeService) GetKnowledgeBatch(ctx context.Context,
	tenantID uint64, ids []string,
) ([]*types.Knowledge, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repo.GetKnowledgeBatch(ctx, tenantID, ids)
}

// calculateFileHash calculates MD5 hash of a file
func calculateFileHash(file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	// Reset file pointer for subsequent operations
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func calculateStr(strList ...string) string {
	h := md5.New()
	input := strings.Join(strList, "")
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *knowledgeService) CloneKnowledgeBase(ctx context.Context, srcID, dstID string) error {
	srcKB, dstKB, err := s.kbService.CopyKnowledgeBase(ctx, srcID, dstID)
	if err != nil {
		logger.Errorf(ctx, "Failed to copy knowledge base: %v", err)
		return err
	}

	addKnowledge, err := s.repo.AminusB(ctx, srcKB.TenantID, srcKB.ID, dstKB.TenantID, dstKB.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge: %v", err)
		return err
	}

	delKnowledge, err := s.repo.AminusB(ctx, dstKB.TenantID, dstKB.ID, srcKB.TenantID, srcKB.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge: %v", err)
		return err
	}
	logger.Infof(ctx, "Knowledge after update to add: %d, delete: %d", len(addKnowledge), len(delKnowledge))

	batch := 10
	g, gctx := errgroup.WithContext(ctx)
	for ids := range slices.Chunk(delKnowledge, batch) {
		g.Go(func() error {
			err := s.DeleteKnowledgeList(gctx, ids)
			if err != nil {
				logger.Errorf(gctx, "delete partial knowledge %v: %v", ids, err)
				return err
			}
			return nil
		})
	}
	err = g.Wait()
	if err != nil {
		logger.Errorf(ctx, "delete total knowledge %d: %v", len(delKnowledge), err)
		return err
	}

	// Copy context out of auto-stop task
	g, gctx = errgroup.WithContext(ctx)
	g.SetLimit(batch)
	for _, knowledge := range addKnowledge {
		g.Go(func() error {
			srcKn, err := s.repo.GetKnowledgeByID(gctx, srcKB.TenantID, knowledge)
			if err != nil {
				logger.Errorf(gctx, "get knowledge %s: %v", knowledge, err)
				return err
			}
			err = s.cloneKnowledge(gctx, srcKn, dstKB)
			if err != nil {
				logger.Errorf(gctx, "clone knowledge %s: %v", knowledge, err)
				return err
			}
			return nil
		})
	}
	err = g.Wait()
	if err != nil {
		logger.Errorf(ctx, "add total knowledge %d: %v", len(addKnowledge), err)
		return err
	}
	return nil
}

func (s *knowledgeService) updateChunkVector(ctx context.Context, kbID string, chunks []*types.Chunk) error {
	// Get embedding model from knowledge base
	sourceKB, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return err
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, sourceKB.EmbeddingModelID)
	if err != nil {
		return err
	}

	// Initialize composite retrieve engine from tenant configuration
	indexInfo := make([]*types.IndexInfo, 0, len(chunks))
	ids := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.KnowledgeBaseID != kbID {
			logger.Warnf(ctx, "Knowledge base ID mismatch: %s != %s", chunk.KnowledgeBaseID, kbID)
			continue
		}
		indexInfo = append(indexInfo, &types.IndexInfo{
			Content:         chunk.Content,
			SourceID:        chunk.ID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         chunk.ID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
		})
		ids = append(ids, chunk.ID)
	}

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		return err
	}

	// Delete old vector representation of the chunk
	err = retrieveEngine.DeleteByChunkIDList(ctx, ids, embeddingModel.GetDimensions(), sourceKB.Type)
	if err != nil {
		return err
	}

	// Index updated chunk content with new vector representation
	err = retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfo)
	if err != nil {
		return err
	}
	return nil
}

func (s *knowledgeService) UpdateImageInfo(
	ctx context.Context,
	knowledgeID string,
	chunkID string,
	imageInfo string,
) error {
	var images []*types.ImageInfo
	if err := json.Unmarshal([]byte(imageInfo), &images); err != nil {
		logger.Errorf(ctx, "Failed to unmarshal image info: %v", err)
		return err
	}
	if len(images) != 1 {
		logger.Warnf(ctx, "Expected exactly one image info, got %d", len(images))
		return nil
	}
	image := images[0]

	// Retrieve all chunks with the given parent chunk ID
	chunk, err := s.chunkService.GetChunkByID(ctx, chunkID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chunk: %v", err)
		return err
	}
	chunk.ImageInfo = imageInfo
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	chunkChildren, err := s.chunkService.ListChunkByParentID(ctx, tenantID, chunkID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"parent_chunk_id": chunkID,
			"tenant_id":       tenantID,
		})
		return err
	}
	logger.Infof(ctx, "Found %d chunks with parent chunk ID: %s", len(chunkChildren), chunkID)

	// Iterate through each chunk and update its content based on the image information
	updateChunk := []*types.Chunk{chunk}
	var addChunk []*types.Chunk

	// Track whether we've found OCR and caption child chunks for this image
	hasOCRChunk := false
	hasCaptionChunk := false

	for i, child := range chunkChildren {
		// Skip chunks that are not image types
		var cImageInfo []*types.ImageInfo
		err = json.Unmarshal([]byte(child.ImageInfo), &cImageInfo)
		if err != nil {
			logger.Warnf(ctx, "Failed to unmarshal image %s info: %v", child.ID, err)
			continue
		}
		if len(cImageInfo) == 0 {
			continue
		}
		if cImageInfo[0].OriginalURL != image.OriginalURL {
			logger.Warnf(ctx, "Skipping chunk ID: %s, image URL mismatch: %s != %s",
				child.ID, cImageInfo[0].OriginalURL, image.OriginalURL)
			continue
		}

		// Mark that we've found chunks for this image
		switch child.ChunkType {
		case types.ChunkTypeImageCaption:
			hasCaptionChunk = true
			// Update caption if it has changed
			if image.Caption != cImageInfo[0].Caption {
				child.Content = image.Caption
				child.ImageInfo = imageInfo
				updateChunk = append(updateChunk, chunkChildren[i])
			}
		case types.ChunkTypeImageOCR:
			hasOCRChunk = true
			// Update OCR if it has changed
			if image.OCRText != cImageInfo[0].OCRText {
				child.Content = image.OCRText
				child.ImageInfo = imageInfo
				updateChunk = append(updateChunk, chunkChildren[i])
			}
		}
	}

	// Create a new caption chunk if it doesn't exist and we have caption data
	if !hasCaptionChunk && image.Caption != "" {
		captionChunk := &types.Chunk{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			Content:         image.Caption,
			ChunkType:       types.ChunkTypeImageCaption,
			ParentChunkID:   chunk.ID,
			ImageInfo:       imageInfo,
		}
		addChunk = append(addChunk, captionChunk)
		logger.Infof(ctx, "Created new caption chunk ID: %s for image URL: %s", captionChunk.ID, image.OriginalURL)
	}

	// Create a new OCR chunk if it doesn't exist and we have OCR data
	if !hasOCRChunk && image.OCRText != "" {
		ocrChunk := &types.Chunk{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			Content:         image.OCRText,
			ChunkType:       types.ChunkTypeImageOCR,
			ParentChunkID:   chunk.ID,
			ImageInfo:       imageInfo,
		}
		addChunk = append(addChunk, ocrChunk)
		logger.Infof(ctx, "Created new OCR chunk ID: %s for image URL: %s", ocrChunk.ID, image.OriginalURL)
	}
	logger.Infof(ctx, "Updated %d chunks out of %d total chunks", len(updateChunk), len(chunkChildren)+1)

	if len(addChunk) > 0 {
		err := s.chunkService.CreateChunks(ctx, addChunk)
		if err != nil {
			logger.ErrorWithFields(ctx, err, map[string]interface{}{
				"add_chunk_size": len(addChunk),
			})
			return err
		}
	}

	// Update the chunks
	for _, c := range updateChunk {
		err := s.chunkService.UpdateChunk(ctx, c)
		if err != nil {
			logger.ErrorWithFields(ctx, err, map[string]interface{}{
				"chunk_id":     c.ID,
				"knowledge_id": c.KnowledgeID,
			})
			return err
		}
	}

	// Update the chunk vector
	err = s.updateChunkVector(ctx, chunk.KnowledgeBaseID, append(updateChunk, addChunk...))
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"chunk_id":     chunk.ID,
			"knowledge_id": chunk.KnowledgeID,
		})
		return err
	}

	// Update the knowledge file hash
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge: %v", err)
		return err
	}
	fileHash := calculateStr(knowledgeID, knowledge.FileHash, imageInfo)
	knowledge.FileHash = fileHash
	err = s.repo.UpdateKnowledge(ctx, knowledge)
	if err != nil {
		logger.Warnf(ctx, "Failed to update knowledge file hash: %v", err)
	}

	logger.Infof(ctx, "Updated chunk successfully, chunk ID: %s, knowledge ID: %s", chunk.ID, chunk.KnowledgeID)
	return nil
}

// CloneChunk clone chunks from one knowledge to another
// This method transfers a chunk from a source knowledge document to a target knowledge document
// It handles the creation of new chunks in the target knowledge and updates the vector database accordingly
// Parameters:
//   - ctx: Context with authentication and request information
//   - src: Source knowledge document containing the chunk to move
//   - dst: Target knowledge document where the chunk will be moved
//
// Returns:
//   - error: Any error encountered during the move operation
//
// This method handles the chunk transfer logic, including creating new chunks in the target knowledge
// and updating the vector database representation of the moved chunks.
// It also ensures that the chunk's relationships (like pre and next chunk IDs) are maintained
// by mapping the source chunk IDs to the new target chunk IDs.
func (s *knowledgeService) CloneChunk(ctx context.Context, src, dst *types.Knowledge) error {
	chunkPage := 1
	chunkPageSize := 100
	srcTodst := map[string]string{}
	tagIDMapping := map[string]string{} // srcTagID -> dstTagID
	targetChunks := make([]*types.Chunk, 0, 10)
	chunkType := []types.ChunkType{
		types.ChunkTypeText, types.ChunkTypeSummary,
		types.ChunkTypeImageCaption, types.ChunkTypeImageOCR,
	}
	for {
		sourceChunks, _, err := s.chunkRepo.ListPagedChunksByKnowledgeID(ctx,
			src.TenantID,
			src.ID,
			&types.Pagination{
				Page:     chunkPage,
				PageSize: chunkPageSize,
			},
			chunkType,
			"",
			"",
		)
		chunkPage++
		if err != nil {
			return err
		}
		if len(sourceChunks) == 0 {
			break
		}
		now := time.Now()
		for _, sourceChunk := range sourceChunks {
			// Map TagID to target knowledge base
			targetTagID := ""
			if sourceChunk.TagID != "" {
				if mappedTagID, ok := tagIDMapping[sourceChunk.TagID]; ok {
					targetTagID = mappedTagID
				} else {
					// Try to find or create the tag in target knowledge base
					targetTagID = s.getOrCreateTagInTarget(ctx, src.TenantID, dst.TenantID, dst.KnowledgeBaseID, sourceChunk.TagID, tagIDMapping)
				}
			}

			targetChunk := &types.Chunk{
				ID:              uuid.New().String(),
				TenantID:        dst.TenantID,
				KnowledgeID:     dst.ID,
				KnowledgeBaseID: dst.KnowledgeBaseID,
				TagID:           targetTagID,
				Content:         sourceChunk.Content,
				ChunkIndex:      sourceChunk.ChunkIndex,
				IsEnabled:       sourceChunk.IsEnabled,
				Flags:           sourceChunk.Flags,
				Status:          sourceChunk.Status,
				StartAt:         sourceChunk.StartAt,
				EndAt:           sourceChunk.EndAt,
				PreChunkID:      sourceChunk.PreChunkID,
				NextChunkID:     sourceChunk.NextChunkID,
				ChunkType:       sourceChunk.ChunkType,
				ParentChunkID:   sourceChunk.ParentChunkID,
				Metadata:        sourceChunk.Metadata,
				ContentHash:     sourceChunk.ContentHash,
				ImageInfo:       sourceChunk.ImageInfo,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			targetChunks = append(targetChunks, targetChunk)
			srcTodst[sourceChunk.ID] = targetChunk.ID
		}
	}
	for _, targetChunk := range targetChunks {
		if val, ok := srcTodst[targetChunk.PreChunkID]; ok {
			targetChunk.PreChunkID = val
		} else {
			targetChunk.PreChunkID = ""
		}
		if val, ok := srcTodst[targetChunk.NextChunkID]; ok {
			targetChunk.NextChunkID = val
		} else {
			targetChunk.NextChunkID = ""
		}
		if val, ok := srcTodst[targetChunk.ParentChunkID]; ok {
			targetChunk.ParentChunkID = val
		} else {
			targetChunk.ParentChunkID = ""
		}
	}
	for chunks := range slices.Chunk(targetChunks, chunkPageSize) {
		err := s.chunkRepo.CreateChunks(ctx, chunks)
		if err != nil {
			return err
		}
	}

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		return err
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, dst.EmbeddingModelID)
	if err != nil {
		return err
	}
	if err := retrieveEngine.CopyIndices(ctx, src.KnowledgeBaseID, dst.KnowledgeBaseID,
		map[string]string{src.ID: dst.ID},
		srcTodst,
		embeddingModel.GetDimensions(),
		dst.Type,
	); err != nil {
		return err
	}
	return nil
}

// ListFAQEntries lists FAQ entries under a FAQ knowledge base.
func (s *knowledgeService) ListFAQEntries(ctx context.Context,
	kbID string, page *types.Pagination, tagID string, keyword string,
) (*types.PageResult, error) {
	if page == nil {
		page = &types.Pagination{}
	}
	keyword = strings.TrimSpace(keyword)
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return nil, err
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	faqKnowledge, err := s.findFAQKnowledge(ctx, tenantID, kb.ID)
	if err != nil {
		return nil, err
	}
	if faqKnowledge == nil {
		return types.NewPageResult(0, page, []*types.FAQEntry{}), nil
	}
	chunkType := []types.ChunkType{types.ChunkTypeFAQ}
	chunks, total, err := s.chunkRepo.ListPagedChunksByKnowledgeID(
		ctx, tenantID, faqKnowledge.ID, page, chunkType, tagID, keyword,
	)
	if err != nil {
		return nil, err
	}
	kb.EnsureDefaults()
	entries := make([]*types.FAQEntry, 0, len(chunks))
	for _, chunk := range chunks {
		entry, err := s.chunkToFAQEntry(chunk, kb)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return types.NewPageResult(total, page, entries), nil
}

// UpsertFAQEntries imports or appends FAQ entries asynchronously.
// Returns task ID for tracking import progress.
func (s *knowledgeService) UpsertFAQEntries(ctx context.Context,
	kbID string, payload *types.FAQBatchUpsertPayload,
) (string, error) {
	if payload == nil || len(payload.Entries) == 0 {
		return "", werrors.NewBadRequestError("FAQ 항목은 비워둘 수 없습니다")
	}
	if payload.Mode == "" {
		payload.Mode = types.FAQBatchModeAppend
	}
	if payload.Mode != types.FAQBatchModeAppend && payload.Mode != types.FAQBatchModeReplace {
		return "", werrors.NewBadRequestError("모드는 append 또는 replace만 지원됩니다")
	}

	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return "", err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	runningKnowledge, err := s.getRunningFAQImportTask(ctx, kbID, tenantID)
	if err != nil {
		logger.Errorf(ctx, "Failed to check running import task: %v", err)
	} else if runningKnowledge != nil {
		logger.Warnf(ctx, "Import task already running for KB %s: %s (status: %s)", kbID, runningKnowledge.ID, runningKnowledge.ParseStatus)
		return "", werrors.NewBadRequestError(fmt.Sprintf("이 지식베이스에 이미 진행 중인 가져오기 작업이 있습니다 (작업 ID: %s). 완료 후 다시 시도해 주세요", runningKnowledge.ID))
	}

	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return "", fmt.Errorf("failed to ensure FAQ knowledge: %w", err)
	}

	taskID := faqKnowledge.ID
	if err := s.updateFAQImportStatusWithRanges(ctx, taskID, types.FAQImportStatusPending,
		0, len(payload.Entries), 0, ""); err != nil {
		logger.Errorf(ctx, "Failed to initialize FAQ import task status: %v", err)
		return "", fmt.Errorf("failed to initialize task: %w", err)
	}

	logger.Infof(ctx, "FAQ import task initialized: %s, total entries: %d", taskID, len(payload.Entries))

	// Enqueue FAQ import task to Asynq
	logger.Info(ctx, "Enqueuing FAQ import task to Asynq")
	taskPayload := types.FAQImportPayload{
		TenantID:    tenantID,
		TaskID:      taskID,
		KBID:        kbID,
		KnowledgeID: faqKnowledge.ID,
		Entries:     payload.Entries,
		Mode:        payload.Mode,
	}

	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal FAQ import task payload: %v", err)
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(types.TypeFAQImport, payloadBytes, asynq.Queue("default"), asynq.MaxRetry(5))
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue FAQ import task: %v", err)
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}
	logger.Infof(ctx, "Enqueued FAQ import task: id=%s queue=%s task_id=%s", info.ID, info.Queue, taskID)

	return taskID, nil
}

func (s *knowledgeService) calculateAppendOperations(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload,
) ([]types.FAQEntryPayload, int, error) {
	if len(entries) == 0 {
		return []types.FAQEntryPayload{}, 0, nil
	}

	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list existing FAQ chunks: %w", err)
	}

	existingQuestions := make(map[string]bool)
	for _, chunk := range existingChunks {
		meta, err := chunk.FAQMetadata()
		if err != nil || meta == nil {
			continue
		}
		if meta.StandardQuestion != "" {
			existingQuestions[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestions[q] = true
			}
		}
	}

	batchQuestions := make(map[string]bool)
	entriesToProcess := make([]types.FAQEntryPayload, 0, len(entries))
	skippedCount := 0

	for _, entry := range entries {
		meta, err := sanitizeFAQEntryPayload(&entry)
		if err != nil {
			skippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry: %v", err)
			continue
		}

		if existingQuestions[meta.StandardQuestion] || batchQuestions[meta.StandardQuestion] {
			skippedCount++
			logger.Infof(ctx, "Skipping FAQ entry with duplicate standard question: %s", meta.StandardQuestion)
			continue
		}

		hasDuplicateSimilar := false
		for _, q := range meta.SimilarQuestions {
			if existingQuestions[q] || batchQuestions[q] {
				hasDuplicateSimilar = true
				logger.Infof(ctx, "Skipping FAQ entry with duplicate similar question: %s (standard: %s)", q, meta.StandardQuestion)
				break
			}
		}
		if hasDuplicateSimilar {
			skippedCount++
			continue
		}

		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}

		entriesToProcess = append(entriesToProcess, entry)
	}

	return entriesToProcess, skippedCount, nil
}

func (s *knowledgeService) calculateReplaceOperations(ctx context.Context,
	tenantID uint64, knowledgeID string, newEntries []types.FAQEntryPayload,
) ([]types.FAQEntryPayload, []*types.Chunk, int, error) {
	type entryWithHash struct {
		entry types.FAQEntryPayload
		hash  string
		meta  *types.FAQChunkMetadata
	}
	entriesWithHash := make([]entryWithHash, 0, len(newEntries))
	newHashSet := make(map[string]bool)
	batchQuestions := make(map[string]bool)
	batchSkippedCount := 0

	for _, entry := range newEntries {
		meta, err := sanitizeFAQEntryPayload(&entry)
		if err != nil {
			batchSkippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry in replace mode: %v", err)
			continue
		}

		if batchQuestions[meta.StandardQuestion] {
			batchSkippedCount++
			logger.Infof(ctx, "Skipping FAQ entry with duplicate standard question in batch: %s", meta.StandardQuestion)
			continue
		}

		hasDuplicateSimilar := false
		for _, q := range meta.SimilarQuestions {
			if batchQuestions[q] {
				hasDuplicateSimilar = true
				logger.Infof(ctx, "Skipping FAQ entry with duplicate similar question in batch: %s (standard: %s)", q, meta.StandardQuestion)
				break
			}
		}
		if hasDuplicateSimilar {
			batchSkippedCount++
			continue
		}

		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}

		hash := types.CalculateFAQContentHash(meta)
		if hash != "" {
			entriesWithHash = append(entriesWithHash, entryWithHash{entry: entry, hash: hash, meta: meta})
			newHashSet[hash] = true
		}
	}

	allExistingChunks, err := s.chunkRepo.ListAllFAQChunksByKnowledgeID(ctx, tenantID, knowledgeID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing chunks: %w", err)
	}

	existingHashMap := make(map[string]*types.Chunk)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash != "" && newHashSet[chunk.ContentHash] {
			existingHashMap[chunk.ContentHash] = chunk
		}
	}

	chunksToDelete := make([]*types.Chunk, 0)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash == "" {
			chunksToDelete = append(chunksToDelete, chunk)
		} else if !newHashSet[chunk.ContentHash] {
			chunksToDelete = append(chunksToDelete, chunk)
		}
	}

	entriesToProcess := make([]types.FAQEntryPayload, 0, len(entriesWithHash))
	skippedCount := batchSkippedCount

	for _, ewh := range entriesWithHash {
		if existingHashMap[ewh.hash] != nil {
			skippedCount++
			continue
		}

		entriesToProcess = append(entriesToProcess, ewh.entry)
	}

	return entriesToProcess, chunksToDelete, skippedCount, nil
}

func (s *knowledgeService) executeFAQImport(ctx context.Context, taskID string, kbID string,
	payload *types.FAQBatchUpsertPayload, tenantID uint64, processedCount int,
) (err error) {
	var kb *types.KnowledgeBase
	var embeddingModel embedding.Embedder
	totalEntries := len(payload.Entries) + processedCount

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			stack := string(buf[:n])
			logger.Errorf(ctx, "FAQ import task %s panicked: %v\n%s", taskID, r, stack)
			err = fmt.Errorf("panic during FAQ import: %v", r)
		}
	}()

	kb, err = s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}

	kb.EnsureDefaults()

	embeddingModel, err = s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return fmt.Errorf("failed to get embedding model: %w", err)
	}
	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return err
	}

	indexMode := types.FAQIndexModeQuestionOnly
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	var entriesToProcess []types.FAQEntryPayload
	var chunksToDelete []*types.Chunk
	var skippedCount int

	if payload.Mode == types.FAQBatchModeReplace {
		entriesToProcess, chunksToDelete, skippedCount, err = s.calculateReplaceOperations(
			ctx,
			tenantID,
			faqKnowledge.ID,
			payload.Entries,
		)
		if err != nil {
			return fmt.Errorf("failed to calculate replace operations: %w", err)
		}

		if len(chunksToDelete) > 0 {
			chunkIDsToDelete := make([]string, 0, len(chunksToDelete))
			for _, chunk := range chunksToDelete {
				chunkIDsToDelete = append(chunkIDsToDelete, chunk.ID)
			}
			if err := s.chunkRepo.DeleteChunks(ctx, tenantID, chunkIDsToDelete); err != nil {
				return fmt.Errorf("failed to delete chunks: %w", err)
			}
			if err := s.deleteFAQChunkVectors(ctx, kb, faqKnowledge, chunksToDelete); err != nil {
				return fmt.Errorf("failed to delete chunk vectors: %w", err)
			}
			logger.Infof(ctx, "FAQ import task %s: deleted %d chunks (including updates)", taskID, len(chunksToDelete))
		}
	} else {
		entriesToProcess, skippedCount, err = s.calculateAppendOperations(ctx, tenantID, kb.ID, payload.Entries)
		if err != nil {
			return fmt.Errorf("failed to calculate append operations: %w", err)
		}
	}

	logger.Infof(
		ctx,
		"FAQ import task %s: total entries: %d, to process: %d, skipped: %d",
		taskID,
		len(payload.Entries),
		len(entriesToProcess),
		skippedCount,
	)

	if len(entriesToProcess) == 0 {
		logger.Infof(ctx, "FAQ import task %s: no entries to process, all skipped", taskID)
		return nil
	}

	remainingEntries := len(entriesToProcess)
	totalStartTime := time.Now()
	actualProcessed := skippedCount + processedCount

	logger.Infof(
		ctx,
		"FAQ import task %s: starting batch processing, remaining entries: %d, total entries: %d, batch size: %d",
		taskID,
		remainingEntries,
		totalEntries,
		faqImportBatchSize,
	)

	for i := 0; i < remainingEntries; i += faqImportBatchSize {
		batchStartTime := time.Now()
		end := i + faqImportBatchSize
		if end > remainingEntries {
			end = remainingEntries
		}

		batch := entriesToProcess[i:end]
		logger.Infof(ctx, "FAQ import task %s: processing batch %d-%d (%d entries)", taskID, i+1, end, len(batch))

		buildStartTime := time.Now()
		chunks := make([]*types.Chunk, 0, len(batch))
		chunkIds := make([]string, 0, len(batch))
		for idx, entry := range batch {
			meta, err := sanitizeFAQEntryPayload(&entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to sanitize entry at index %d: %w", i+idx, err)
			}

			tagID, err := s.resolveTagID(ctx, kbID, &entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to resolve tag for entry at index %d: %w", i+idx, err)
			}

			isEnabled := true
			if entry.IsEnabled != nil {
				isEnabled = *entry.IsEnabled
			}
			chunk := &types.Chunk{
				ID:              uuid.New().String(),
				TenantID:        tenantID,
				KnowledgeID:     faqKnowledge.ID,
				KnowledgeBaseID: kb.ID,
				Content:         buildFAQChunkContent(meta, indexMode),
				// ChunkIndex:      0,
				IsEnabled: isEnabled,
				ChunkType: types.ChunkTypeFAQ,
				TagID:     tagID,
				Status:    int(types.ChunkStatusStored), // store but not indexed
			}
			if err := chunk.SetFAQMetadata(meta); err != nil {
				return fmt.Errorf("failed to set FAQ metadata: %w", err)
			}
			chunks = append(chunks, chunk)
			chunkIds = append(chunkIds, chunk.ID)
		}
		buildDuration := time.Since(buildStartTime)
		logger.Debugf(ctx, "FAQ import task %s: batch %d-%d built %d chunks in %v, chunk IDs: %v",
			taskID, i+1, end, len(chunks), buildDuration, chunkIds)
		createStartTime := time.Now()
		if err := s.chunkService.CreateChunks(ctx, chunks); err != nil {
			return fmt.Errorf("failed to create chunks: %w", err)
		}
		createDuration := time.Since(createStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d created %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			createDuration,
		)

		indexStartTime := time.Now()
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, chunks, embeddingModel, true, false); err != nil {
			return fmt.Errorf("failed to index chunks: %w", err)
		}
		indexDuration := time.Since(indexStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d indexed %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			indexDuration,
		)

		chunksToUpdate := make([]*types.Chunk, 0, len(chunks))
		for _, chunk := range chunks {
			chunk.Status = int(types.ChunkStatusIndexed) // indexed
			chunksToUpdate = append(chunksToUpdate, chunk)
		}
		if err := s.chunkService.UpdateChunks(ctx, chunksToUpdate); err != nil {
			return fmt.Errorf("failed to update chunks status: %w", err)
		}

		actualProcessed += len(batch)
		progress := int(float64(actualProcessed) / float64(totalEntries) * 100)
		if err := s.updateFAQImportStatus(ctx, taskID, types.FAQImportStatusProcessing, progress, totalEntries, actualProcessed, ""); err != nil {
			logger.Errorf(ctx, "Failed to update task progress: %v", err)
		}

		batchDuration := time.Since(batchStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d completed in %v (build: %v, create: %v, index: %v), total progress: %d/%d (%d%%)",
			taskID,
			i+1,
			end,
			batchDuration,
			buildDuration,
			createDuration,
			indexDuration,
			actualProcessed,
			totalEntries,
			progress,
		)
	}

	totalDuration := time.Since(totalStartTime)
	logger.Infof(
		ctx,
		"FAQ import task %s: all batches completed, processed: %d entries (skipped: %d) in %v, avg: %v per entry",
		taskID,
		actualProcessed,
		skippedCount,
		totalDuration,
		totalDuration/time.Duration(actualProcessed),
	)

	return nil
}

// CreateFAQEntry creates a single FAQ entry synchronously.
func (s *knowledgeService) CreateFAQEntry(ctx context.Context,
	kbID string, payload *types.FAQEntryPayload,
) (*types.FAQEntry, error) {
	if payload == nil {
		return nil, werrors.NewBadRequestError("요청 본문은 비워둘 수 없습니다")
	}

	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return nil, err
	}
	kb.EnsureDefaults()

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	meta, err := sanitizeFAQEntryPayload(payload)
	if err != nil {
		return nil, err
	}

	tagID, err := s.resolveTagID(ctx, kbID, payload)
	if err != nil {
		return nil, err
	}

	if err := s.checkFAQQuestionDuplicate(ctx, tenantID, kb.ID, "", meta); err != nil {
		return nil, err
	}

	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure FAQ knowledge: %w", err)
	}

	indexMode := types.FAQIndexModeQuestionOnly
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding model: %w", err)
	}

	isEnabled := true
	if payload.IsEnabled != nil {
		isEnabled = *payload.IsEnabled
	}
	flags := types.ChunkFlagRecommended
	if payload.IsRecommended != nil && !*payload.IsRecommended {
		flags = 0
	}

	chunk := &types.Chunk{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		KnowledgeID:     faqKnowledge.ID,
		KnowledgeBaseID: kb.ID,
		Content:         buildFAQChunkContent(meta, indexMode),
		IsEnabled:       isEnabled,
		Flags:           flags,
		ChunkType:       types.ChunkTypeFAQ,
		TagID:           tagID,
		Status:          int(types.ChunkStatusStored),
	}

	if err := chunk.SetFAQMetadata(meta); err != nil {
		return nil, fmt.Errorf("failed to set FAQ metadata: %w", err)
	}

	if err := s.chunkService.CreateChunks(ctx, []*types.Chunk{chunk}); err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	if err := s.indexFAQChunks(ctx, kb, faqKnowledge, []*types.Chunk{chunk}, embeddingModel, true, false); err != nil {
		_ = s.chunkService.DeleteChunk(ctx, chunk.ID)
		return nil, fmt.Errorf("failed to index chunk: %w", err)
	}

	chunk.Status = int(types.ChunkStatusIndexed)
	if err := s.chunkService.UpdateChunk(ctx, chunk); err != nil {
		return nil, fmt.Errorf("failed to update chunk status: %w", err)
	}

	entry, err := s.chunkToFAQEntry(chunk, kb)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// UpdateFAQEntry updates a single FAQ entry.
func (s *knowledgeService) UpdateFAQEntry(ctx context.Context,
	kbID string, entryID string, payload *types.FAQEntryPayload,
) error {
	if payload == nil {
		return werrors.NewBadRequestError("요청 본문은 비워둘 수 없습니다")
	}
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	kb.EnsureDefaults()
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	chunk, err := s.chunkRepo.GetChunkByID(ctx, tenantID, entryID)
	if err != nil {
		return err
	}
	if chunk.KnowledgeBaseID != kb.ID {
		return werrors.NewForbiddenError("이 FAQ 항목을 수정할 권한이 없습니다")
	}
	if chunk.ChunkType != types.ChunkTypeFAQ {
		return werrors.NewBadRequestError("FAQ 항목만 업데이트할 수 있습니다")
	}
	meta, err := sanitizeFAQEntryPayload(payload)
	if err != nil {
		return err
	}

	if err := s.checkFAQQuestionDuplicate(ctx, tenantID, kb.ID, entryID, meta); err != nil {
		return err
	}

	if existing, err := chunk.FAQMetadata(); err == nil && existing != nil {
		meta.Version = existing.Version + 1
	}
	if err := chunk.SetFAQMetadata(meta); err != nil {
		return err
	}
	indexMode := types.FAQIndexModeQuestionOnly
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}
	chunk.Content = buildFAQChunkContent(meta, indexMode)
	chunk.TagID = payload.TagID
	isEnabledUpdated := false
	if payload.IsEnabled != nil {
		oldEnabled := chunk.IsEnabled
		chunk.IsEnabled = *payload.IsEnabled
		isEnabledUpdated = (oldEnabled != chunk.IsEnabled)
	}
	if payload.IsRecommended != nil {
		if *payload.IsRecommended {
			chunk.Flags = chunk.Flags.SetFlag(types.ChunkFlagRecommended)
		} else {
			chunk.Flags = chunk.Flags.ClearFlag(types.ChunkFlagRecommended)
		}
	}
	chunk.UpdatedAt = time.Now()
	if err := s.chunkService.UpdateChunk(ctx, chunk); err != nil {
		return err
	}

	// Sync is_enabled status to retriever engines if it was updated
	if isEnabledUpdated {
		chunkStatusMap := map[string]bool{chunk.ID: chunk.IsEnabled}
		tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
		retrieveEngine, err := retriever.NewCompositeRetrieveEngine(
			s.retrieveEngine,
			tenantInfo.GetEffectiveEngines(),
		)
		if err != nil {
			return err
		}
		if err := retrieveEngine.BatchUpdateChunkEnabledStatus(ctx, chunkStatusMap); err != nil {
			return err
		}
	}

	faqKnowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, chunk.KnowledgeID)
	if err != nil {
		return err
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return err
	}
	return s.indexFAQChunks(ctx, kb, faqKnowledge, []*types.Chunk{chunk}, embeddingModel, false, true)
}

// UpdateFAQEntryStatus updates enable status for a FAQ entry.
func (s *knowledgeService) UpdateFAQEntryStatus(ctx context.Context,
	kbID string, entryID string, isEnabled bool,
) error {
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	chunk, err := s.chunkRepo.GetChunkByID(ctx, tenantID, entryID)
	if err != nil {
		return err
	}
	if chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
		return werrors.NewBadRequestError("FAQ 항목만 업데이트할 수 있습니다")
	}
	if chunk.IsEnabled == isEnabled {
		return nil
	}
	chunk.IsEnabled = isEnabled
	chunk.UpdatedAt = time.Now()
	if err := s.chunkService.UpdateChunk(ctx, chunk); err != nil {
		return err
	}

	// Sync update to retriever engines
	chunkStatusMap := map[string]bool{chunk.ID: isEnabled}
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		return err
	}
	if err := retrieveEngine.BatchUpdateChunkEnabledStatus(ctx, chunkStatusMap); err != nil {
		return err
	}

	return nil
}

// UpdateFAQEntryFieldsBatch updates multiple fields for FAQ entries in batch.
// This is the unified API for batch updating FAQ entry fields.
// Supports two modes:
// 1. By entry ID: use ByID field
// 2. By Tag: use ByTag field to apply the same update to all entries under a tag
func (s *knowledgeService) UpdateFAQEntryFieldsBatch(ctx context.Context,
	kbID string, req *types.FAQEntryFieldsBatchUpdate,
) error {
	if req == nil || (len(req.ByID) == 0 && len(req.ByTag) == 0) {
		return nil
	}
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	enabledUpdates := make(map[string]bool)

	// Handle ByTag updates first
	if len(req.ByTag) > 0 {
		for tagID, update := range req.ByTag {
			var setFlags, clearFlags types.ChunkFlags

			// Handle IsRecommended
			if update.IsRecommended != nil {
				if *update.IsRecommended {
					setFlags = types.ChunkFlagRecommended
				} else {
					clearFlags = types.ChunkFlagRecommended
				}
			}

			// Update all chunks with this tag
			affectedIDs, err := s.chunkRepo.UpdateChunkFieldsByTagID(
				ctx, tenantID, kb.ID, tagID,
				update.IsEnabled, setFlags, clearFlags,
			)
			if err != nil {
				return err
			}

			// Collect affected IDs for retriever sync
			if update.IsEnabled != nil && len(affectedIDs) > 0 {
				for _, id := range affectedIDs {
					enabledUpdates[id] = *update.IsEnabled
				}
			}
		}
	}

	// Handle ByID updates
	if len(req.ByID) > 0 {
		entryIDs := make([]string, 0, len(req.ByID))
		for entryID := range req.ByID {
			entryIDs = append(entryIDs, entryID)
		}
		chunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, entryIDs)
		if err != nil {
			return err
		}

		setFlags := make(map[string]types.ChunkFlags)
		clearFlags := make(map[string]types.ChunkFlags)
		chunksToUpdate := make([]*types.Chunk, 0)

		for _, chunk := range chunks {
			if chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
				continue
			}
			update, exists := req.ByID[chunk.ID]
			if !exists {
				continue
			}

			needUpdate := false

			// Handle IsEnabled
			if update.IsEnabled != nil && chunk.IsEnabled != *update.IsEnabled {
				chunk.IsEnabled = *update.IsEnabled
				enabledUpdates[chunk.ID] = *update.IsEnabled
				needUpdate = true
			}

			// Handle IsRecommended (via Flags)
			if update.IsRecommended != nil {
				currentRecommended := chunk.Flags.HasFlag(types.ChunkFlagRecommended)
				if currentRecommended != *update.IsRecommended {
					if *update.IsRecommended {
						setFlags[chunk.ID] = types.ChunkFlagRecommended
					} else {
						clearFlags[chunk.ID] = types.ChunkFlagRecommended
					}
				}
			}

			// Handle TagID
			if update.TagID != nil {
				newTagID := ""
				if *update.TagID != "" {
					newTagID = *update.TagID
				}
				if chunk.TagID != newTagID {
					chunk.TagID = newTagID
					needUpdate = true
				}
			}

			if needUpdate {
				chunk.UpdatedAt = time.Now()
				chunksToUpdate = append(chunksToUpdate, chunk)
			}
		}

		// Batch update chunks (for IsEnabled and TagID)
		if len(chunksToUpdate) > 0 {
			if err := s.chunkRepo.UpdateChunks(ctx, chunksToUpdate); err != nil {
				return err
			}
		}

		// Batch update flags (for IsRecommended)
		if len(setFlags) > 0 || len(clearFlags) > 0 {
			if err := s.chunkRepo.UpdateChunkFlagsBatch(ctx, tenantID, kb.ID, setFlags, clearFlags); err != nil {
				return err
			}
		}
	}

	// Sync enabled status to retriever engines
	if len(enabledUpdates) > 0 {
		tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
		retrieveEngine, err := retriever.NewCompositeRetrieveEngine(
			s.retrieveEngine,
			tenantInfo.GetEffectiveEngines(),
		)
		if err != nil {
			return err
		}
		if err := retrieveEngine.BatchUpdateChunkEnabledStatus(ctx, enabledUpdates); err != nil {
			return err
		}
	}

	return nil
}

// UpdateKnowledgeTag updates the tag assigned to a knowledge document.
func (s *knowledgeService) UpdateKnowledgeTag(ctx context.Context, knowledgeID string, tagID *string) error {
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
	if err != nil {
		return err
	}

	var resolvedTagID string
	if tagID != nil && *tagID != "" {
		tag, err := s.tagRepo.GetByID(ctx, tenantID, *tagID)
		if err != nil {
			return err
		}
		if tag.KnowledgeBaseID != knowledge.KnowledgeBaseID {
			return werrors.NewBadRequestError("태그가 현재 지식베이스에 속하지 않습니다")
		}
		resolvedTagID = tag.ID
	}

	knowledge.TagID = resolvedTagID
	return s.repo.UpdateKnowledge(ctx, knowledge)
}

// UpdateKnowledgeTagBatch updates tags for document knowledge items in batch.
func (s *knowledgeService) UpdateKnowledgeTagBatch(ctx context.Context, updates map[string]*string) error {
	if len(updates) == 0 {
		return nil
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	// Get all knowledge items in batch
	knowledgeIDs := make([]string, 0, len(updates))
	for knowledgeID := range updates {
		knowledgeIDs = append(knowledgeIDs, knowledgeID)
	}
	knowledgeList, err := s.repo.GetKnowledgeBatch(ctx, tenantID, knowledgeIDs)
	if err != nil {
		return err
	}

	// Build tag ID map for validation
	tagIDSet := make(map[string]bool)
	for _, tagID := range updates {
		if tagID != nil && *tagID != "" {
			tagIDSet[*tagID] = true
		}
	}

	// Validate all tags in batch
	tagMap := make(map[string]*types.KnowledgeTag)
	if len(tagIDSet) > 0 {
		tagIDs := make([]string, 0, len(tagIDSet))
		for tagID := range tagIDSet {
			tagIDs = append(tagIDs, tagID)
		}
		for _, tagID := range tagIDs {
			tag, err := s.tagRepo.GetByID(ctx, tenantID, tagID)
			if err != nil {
				return err
			}
			tagMap[tagID] = tag
		}
	}

	// Update knowledge items
	knowledgeToUpdate := make([]*types.Knowledge, 0)
	for _, knowledge := range knowledgeList {
		tagID, exists := updates[knowledge.ID]
		if !exists {
			continue
		}

		var resolvedTagID string
		if tagID != nil && *tagID != "" {
			tag, ok := tagMap[*tagID]
			if !ok {
				return werrors.NewBadRequestError(fmt.Sprintf("태그 %s 가 존재하지 않습니다", *tagID))
			}
			if tag.KnowledgeBaseID != knowledge.KnowledgeBaseID {
				return werrors.NewBadRequestError(fmt.Sprintf("태그 %s 가 지식베이스 %s 에 속하지 않습니다", *tagID, knowledge.KnowledgeBaseID))
			}
			resolvedTagID = tag.ID
		}

		knowledge.TagID = resolvedTagID
		knowledgeToUpdate = append(knowledgeToUpdate, knowledge)
	}

	if len(knowledgeToUpdate) > 0 {
		return s.repo.UpdateKnowledgeBatch(ctx, knowledgeToUpdate)
	}

	return nil
}

// UpdateFAQEntryTag updates the tag assigned to an FAQ entry.
func (s *knowledgeService) UpdateFAQEntryTag(ctx context.Context, kbID string, entryID string, tagID *string) error {
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	chunk, err := s.chunkRepo.GetChunkByID(ctx, tenantID, entryID)
	if err != nil {
		return err
	}
	if chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
		return werrors.NewBadRequestError("FAQ 항목 태그만 업데이트할 수 있습니다")
	}

	var resolvedTagID string
	if tagID != nil && *tagID != "" {
		tag, err := s.tagRepo.GetByID(ctx, tenantID, *tagID)
		if err != nil {
			return err
		}
		if tag.KnowledgeBaseID != kb.ID {
			return werrors.NewBadRequestError("태그가 현재 지식베이스에 속하지 않습니다")
		}
		resolvedTagID = tag.ID
	}

	chunk.TagID = resolvedTagID
	chunk.UpdatedAt = time.Now()
	return s.chunkRepo.UpdateChunk(ctx, chunk)
}

// UpdateFAQEntryTagBatch updates tags for FAQ entries in batch.
func (s *knowledgeService) UpdateFAQEntryTagBatch(ctx context.Context, kbID string, updates map[string]*string) error {
	if len(updates) == 0 {
		return nil
	}
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	// Get all chunks in batch
	entryIDs := make([]string, 0, len(updates))
	for entryID := range updates {
		entryIDs = append(entryIDs, entryID)
	}
	chunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, entryIDs)
	if err != nil {
		return err
	}

	// Build tag ID map for validation
	tagIDSet := make(map[string]bool)
	for _, tagID := range updates {
		if tagID != nil && *tagID != "" {
			tagIDSet[*tagID] = true
		}
	}

	// Validate all tags in batch
	tagMap := make(map[string]*types.KnowledgeTag)
	if len(tagIDSet) > 0 {
		tagIDs := make([]string, 0, len(tagIDSet))
		for tagID := range tagIDSet {
			tagIDs = append(tagIDs, tagID)
		}
		for _, tagID := range tagIDs {
			tag, err := s.tagRepo.GetByID(ctx, tenantID, tagID)
			if err != nil {
				return err
			}
			if tag.KnowledgeBaseID != kb.ID {
				return werrors.NewBadRequestError(fmt.Sprintf("태그 %s 가 현재 지식베이스에 속하지 않습니다", tagID))
			}
			tagMap[tagID] = tag
		}
	}

	// Update chunks
	chunksToUpdate := make([]*types.Chunk, 0)
	for _, chunk := range chunks {
		if chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
			continue
		}
		tagID, exists := updates[chunk.ID]
		if !exists {
			continue
		}

		var resolvedTagID string
		if tagID != nil && *tagID != "" {
			tag, ok := tagMap[*tagID]
			if !ok {
				return werrors.NewBadRequestError(fmt.Sprintf("태그 %s 가 존재하지 않습니다", *tagID))
			}
			resolvedTagID = tag.ID
		}

		chunk.TagID = resolvedTagID
		chunk.UpdatedAt = time.Now()
		chunksToUpdate = append(chunksToUpdate, chunk)
	}

	if len(chunksToUpdate) > 0 {
		return s.chunkRepo.UpdateChunks(ctx, chunksToUpdate)
	}

	return nil
}

// SearchFAQEntries searches FAQ entries using hybrid search.
func (s *knowledgeService) SearchFAQEntries(ctx context.Context,
	kbID string, req *types.FAQSearchRequest,
) ([]*types.FAQEntry, error) {
	// Validate FAQ knowledge base
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return nil, err
	}

	// Set default values
	if req.VectorThreshold <= 0 {
		req.VectorThreshold = 0.7
	}
	if req.MatchCount <= 0 {
		req.MatchCount = 10
	}
	if req.MatchCount > 50 {
		req.MatchCount = 50
	}

	// Prepare search parameters
	searchParams := types.SearchParams{
		QueryText:            secutils.SanitizeForLog(req.QueryText),
		VectorThreshold:      req.VectorThreshold,
		MatchCount:           req.MatchCount,
		DisableKeywordsMatch: true,
	}

	// Call HybridSearch
	searchResults, err := s.kbService.HybridSearch(ctx, kbID, searchParams)
	if err != nil {
		return nil, err
	}

	if len(searchResults) == 0 {
		return []*types.FAQEntry{}, nil
	}

	// Extract chunk IDs and build score/match type maps
	chunkIDs := make([]string, 0, len(searchResults))
	chunkScores := make(map[string]float64)
	chunkMatchTypes := make(map[string]types.MatchType)
	for _, result := range searchResults {
		// SearchResult.ID is the chunk ID
		chunkID := result.ID
		chunkIDs = append(chunkIDs, chunkID)
		chunkScores[chunkID] = result.Score
		chunkMatchTypes[chunkID] = result.MatchType
	}

	// Batch fetch chunks
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	chunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, chunkIDs)
	if err != nil {
		return nil, err
	}

	// Filter FAQ chunks and convert to FAQEntry
	kb.EnsureDefaults()
	entries := make([]*types.FAQEntry, 0, len(chunks))
	for _, chunk := range chunks {
		// Only process FAQ type chunks
		if chunk.ChunkType != types.ChunkTypeFAQ {
			continue
		}
		if !chunk.IsEnabled {
			continue
		}

		entry, err := s.chunkToFAQEntry(chunk, kb)
		if err != nil {
			logger.Warnf(ctx, "Failed to convert chunk to FAQ entry: %v", err)
			continue
		}

		// Preserve score and match type from search results
		// Note: Negative question filtering is now handled in HybridSearch
		if score, ok := chunkScores[chunk.ID]; ok {
			entry.Score = score
		}
		if matchType, ok := chunkMatchTypes[chunk.ID]; ok {
			entry.MatchType = matchType
		}

		entries = append(entries, entry)
	}

	slices.SortFunc(entries, func(a, b *types.FAQEntry) int {
		return int(b.Score - a.Score)
	})

	return entries, nil
}

// DeleteFAQEntries deletes FAQ entries in batch.
func (s *knowledgeService) DeleteFAQEntries(ctx context.Context,
	kbID string, entryIDs []string,
) error {
	if len(entryIDs) == 0 {
		return werrors.NewBadRequestError("삭제할 FAQ 항목을 선택해 주세요")
	}
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	var faqKnowledge *types.Knowledge
	chunksToRemove := make([]*types.Chunk, 0, len(entryIDs))
	for _, id := range entryIDs {
		if id == "" {
			continue
		}
		chunk, err := s.chunkRepo.GetChunkByID(ctx, tenantID, id)
		if err != nil {
			return err
		}
		if chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
			return werrors.NewBadRequestError("유효하지 않은 FAQ 항목이 포함되어 있습니다")
		}
		if err := s.chunkService.DeleteChunk(ctx, id); err != nil {
			return err
		}
		if faqKnowledge == nil {
			faqKnowledge, err = s.repo.GetKnowledgeByID(ctx, tenantID, chunk.KnowledgeID)
			if err != nil {
				return err
			}
		}
		chunksToRemove = append(chunksToRemove, chunk)
	}
	if len(chunksToRemove) > 0 && faqKnowledge != nil {
		if err := s.deleteFAQChunkVectors(ctx, kb, faqKnowledge, chunksToRemove); err != nil {
			return err
		}
	}
	return nil
}

// ExportFAQEntries exports all FAQ entries for a knowledge base as CSV data.
// The CSV format matches the import example format with 8 columns:
func (s *knowledgeService) ExportFAQEntries(ctx context.Context, kbID string) ([]byte, error) {
	kb, err := s.validateFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return nil, err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	faqKnowledge, err := s.findFAQKnowledge(ctx, tenantID, kb.ID)
	if err != nil {
		return nil, err
	}
	if faqKnowledge == nil {
		// Return empty CSV with headers only
		return s.buildFAQCSV(nil, nil), nil
	}

	// Get all FAQ chunks
	chunks, err := s.chunkRepo.ListAllFAQChunksForExport(ctx, tenantID, faqKnowledge.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list FAQ chunks: %w", err)
	}

	// Build tag map for tag_id -> tag_name conversion
	tagMap, err := s.buildTagMap(ctx, tenantID, kbID)
	if err != nil {
		return nil, fmt.Errorf("failed to build tag map: %w", err)
	}

	return s.buildFAQCSV(chunks, tagMap), nil
}

// buildTagMap builds a map from tag_id to tag_name for the given knowledge base.
func (s *knowledgeService) buildTagMap(ctx context.Context, tenantID uint64, kbID string) (map[string]string, error) {
	// Get all tags for this knowledge base (no pagination limit)
	page := &types.Pagination{Page: 1, PageSize: 10000}
	tags, _, err := s.tagRepo.ListByKB(ctx, tenantID, kbID, page, "")
	if err != nil {
		return nil, err
	}

	tagMap := make(map[string]string, len(tags))
	for _, tag := range tags {
		if tag != nil {
			tagMap[tag.ID] = tag.Name
		}
	}
	return tagMap, nil
}

// buildFAQCSV builds CSV content from FAQ chunks.
func (s *knowledgeService) buildFAQCSV(chunks []*types.Chunk, tagMap map[string]string) []byte {
	var buf strings.Builder

	// Write CSV header (matching import example format)
	headers := []string{
		"분류(필수)",
		"질문(필수)",
		"유사 질문(선택-여러 개는 ##으로 구분)",
		"반례 질문(선택-여러 개는 ##으로 구분)",
		"봇 답변(필수-여러 개는 ##으로 구분)",
		"전체 답변 여부(선택-기본값 FALSE)",
		"비활성화 여부(선택-기본값 FALSE)",
		"추천 금지 여부(선택-기본값 False, 추천 가능)",
	}
	buf.WriteString(strings.Join(headers, ","))
	buf.WriteString("\n")

	// Write data rows
	for _, chunk := range chunks {
		meta, err := chunk.FAQMetadata()
		if err != nil || meta == nil {
			continue
		}

		// Get tag name
		tagName := ""
		if chunk.TagID != "" && tagMap != nil {
			if name, ok := tagMap[chunk.TagID]; ok {
				tagName = name
			}
		}

		// Build row
		row := []string{
			escapeCSVField(tagName),
			escapeCSVField(meta.StandardQuestion),
			escapeCSVField(strings.Join(meta.SimilarQuestions, "##")),
			escapeCSVField(strings.Join(meta.NegativeQuestions, "##")),
			escapeCSVField(strings.Join(meta.Answers, "##")),
			boolToCSV(meta.AnswerStrategy == types.AnswerStrategyAll),
			boolToCSV(!chunk.IsEnabled),
			boolToCSV(!chunk.Flags.HasFlag(types.ChunkFlagRecommended)),
		}
		buf.WriteString(strings.Join(row, ","))
		buf.WriteString("\n")
	}

	return []byte(buf.String())
}

// escapeCSVField escapes a field for CSV format.
func escapeCSVField(field string) string {
	// If field contains comma, newline, or quote, wrap in quotes and escape internal quotes
	if strings.ContainsAny(field, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(field, "\"", "\"\"") + "\""
	}
	return field
}

// boolToCSV converts a boolean to CSV TRUE/FALSE string.
func boolToCSV(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

func (s *knowledgeService) validateFAQKnowledgeBase(ctx context.Context, kbID string) (*types.KnowledgeBase, error) {
	if kbID == "" {
		return nil, werrors.NewBadRequestError("지식베이스 ID는 비워둘 수 없습니다")
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	kb.EnsureDefaults()
	if kb.Type != types.KnowledgeBaseTypeFAQ {
		return nil, werrors.NewBadRequestError("FAQ 지식베이스만 이 작업을 지원합니다")
	}
	return kb, nil
}

func (s *knowledgeService) findFAQKnowledge(
	ctx context.Context,
	tenantID uint64,
	kbID string,
) (*types.Knowledge, error) {
	knowledges, err := s.repo.ListKnowledgeByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return nil, err
	}
	for _, knowledge := range knowledges {
		if knowledge.Type == types.KnowledgeTypeFAQ {
			return knowledge, nil
		}
	}
	return nil, nil
}

func (s *knowledgeService) ensureFAQKnowledge(
	ctx context.Context,
	tenantID uint64,
	kb *types.KnowledgeBase,
) (*types.Knowledge, error) {
	existing, err := s.findFAQKnowledge(ctx, tenantID, kb.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	knowledge := &types.Knowledge{
		TenantID:         tenantID,
		KnowledgeBaseID:  kb.ID,
		Type:             types.KnowledgeTypeFAQ,
		Title:            fmt.Sprintf("%s - FAQ", kb.Name),
		Description:      "FAQ 항목 컨테이너",
		Source:           types.KnowledgeTypeFAQ,
		ParseStatus:      "completed",
		EnableStatus:     "enabled",
		EmbeddingModelID: kb.EmbeddingModelID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := s.repo.CreateKnowledge(ctx, knowledge); err != nil {
		return nil, err
	}
	return knowledge, nil
}

func (s *knowledgeService) updateFAQImportStatus(
	ctx context.Context,
	knowledgeID string,
	status types.FAQImportTaskStatus,
	progress, total, processed int,
	errorMsg string,
) error {
	return s.updateFAQImportStatusWithRanges(ctx, knowledgeID, status, progress, total, processed, errorMsg)
}

func (s *knowledgeService) updateFAQImportStatusWithRanges(
	ctx context.Context,
	knowledgeID string,
	status types.FAQImportTaskStatus,
	progress, total, processed int,
	errorMsg string,
) error {
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
	if err != nil {
		return err
	}


	knowledge.ParseStatus = string(status)
	knowledge.UpdatedAt = time.Now()

	meta, err := types.ParseFAQImportMetadata(knowledge)
	if err != nil || meta == nil {
		meta = &types.FAQImportMetadata{}
	}

	knowledge.ErrorMessage = errorMsg
	if status == types.FAQImportStatusCompleted {
		knowledge.ErrorMessage = ""
	}

	meta.ImportProgress = progress
	meta.ImportTotal = total
	meta.ImportProcessed = processed
	metaJSON, err := meta.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal import metadata: %w", err)
	}
	knowledge.Metadata = metaJSON

	return s.repo.UpdateKnowledge(ctx, knowledge)
}

func (s *knowledgeService) getRunningFAQImportTask(
	ctx context.Context,
	kbID string,
	tenantID uint64,
) (*types.Knowledge, error) {
	faqKnowledge, err := s.findFAQKnowledge(ctx, tenantID, kbID)
	if err != nil {
		return nil, err
	}
	if faqKnowledge == nil {
		return nil, errors.New("FAQ knowledge not found")
	}
	if faqKnowledge.ParseStatus == "pending" || faqKnowledge.ParseStatus == "processing" {
		return faqKnowledge, nil
	}
	return nil, nil
}

func (s *knowledgeService) chunkToFAQEntry(chunk *types.Chunk, kb *types.KnowledgeBase) (*types.FAQEntry, error) {
	meta, err := chunk.FAQMetadata()
	if err != nil {
		return nil, err
	}
	if meta == nil {
		meta = &types.FAQChunkMetadata{StandardQuestion: chunk.Content}
	}
	answerStrategy := meta.AnswerStrategy
	if answerStrategy == "" {
		answerStrategy = types.AnswerStrategyAll
	}
	entry := &types.FAQEntry{
		ID:                chunk.ID,
		ChunkID:           chunk.ID,
		KnowledgeID:       chunk.KnowledgeID,
		KnowledgeBaseID:   chunk.KnowledgeBaseID,
		TagID:             chunk.TagID,
		IsEnabled:         chunk.IsEnabled,
		IsRecommended:     chunk.Flags.HasFlag(types.ChunkFlagRecommended),
		StandardQuestion:  meta.StandardQuestion,
		SimilarQuestions:  meta.SimilarQuestions,
		NegativeQuestions: meta.NegativeQuestions,
		Answers:           meta.Answers,
		AnswerStrategy:    answerStrategy,
		IndexMode:         kb.FAQConfig.IndexMode,
		UpdatedAt:         chunk.UpdatedAt,
		CreatedAt:         chunk.CreatedAt,
		ChunkType:         chunk.ChunkType,
	}
	return entry, nil
}

func buildFAQChunkContent(meta *types.FAQChunkMetadata, mode types.FAQIndexMode) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Q: %s\n", meta.StandardQuestion))
	if len(meta.SimilarQuestions) > 0 {
		builder.WriteString("Similar Questions:\n")
		for _, q := range meta.SimilarQuestions {
			builder.WriteString(fmt.Sprintf("- %s\n", q))
		}
	}
	if mode == types.FAQIndexModeQuestionAnswer && len(meta.Answers) > 0 {
		builder.WriteString("Answers:\n")
		for _, ans := range meta.Answers {
			builder.WriteString(fmt.Sprintf("- %s\n", ans))
		}
	}
	return builder.String()
}

func (s *knowledgeService) checkFAQQuestionDuplicate(
	ctx context.Context,
	tenantID uint64,
	kbID string,
	excludeChunkID string,
	meta *types.FAQChunkMetadata,
) error {
	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return fmt.Errorf("failed to list existing FAQ chunks: %w", err)
	}

	for _, chunk := range existingChunks {
		if chunk.ID == excludeChunkID {
			continue
		}

		existingMeta, err := chunk.FAQMetadata()
		if err != nil || existingMeta == nil {
			continue
		}

		if existingMeta.StandardQuestion == meta.StandardQuestion {
			return werrors.NewBadRequestError(fmt.Sprintf("표준 질문 「%s」 이 이미 존재합니다", meta.StandardQuestion))
		}

		for _, q := range existingMeta.SimilarQuestions {
			if q == meta.StandardQuestion {
				return werrors.NewBadRequestError(fmt.Sprintf("표준 질문 「%s」 이 기존 유사 질문과 중복됩니다", meta.StandardQuestion))
			}
		}

		for _, q := range meta.SimilarQuestions {
			if q == existingMeta.StandardQuestion {
				return werrors.NewBadRequestError(fmt.Sprintf("유사 질문 「%s」 이 기존 표준 질문과 중복됩니다", q))
			}
		}

		for _, q := range meta.SimilarQuestions {
			for _, existingQ := range existingMeta.SimilarQuestions {
				if q == existingQ {
					return werrors.NewBadRequestError(fmt.Sprintf("유사 질문 「%s」 이 이미 존재합니다", q))
				}
			}
		}
	}

	return nil
}

// resolveTagID resolves tag ID from payload, prioritizing tag_id over tag_name
func (s *knowledgeService) resolveTagID(ctx context.Context, kbID string, payload *types.FAQEntryPayload) (string, error) {
	if payload.TagID != "" {
		return payload.TagID, nil
	}

	if payload.TagName != "" && payload.TagName != "미분류" {
		tag, err := s.tagService.FindOrCreateTagByName(ctx, kbID, payload.TagName)
		if err != nil {
			return "", fmt.Errorf("failed to resolve tag by name '%s': %w", payload.TagName, err)
		}
		return tag.ID, nil
	}

	return "", nil
}

func sanitizeFAQEntryPayload(payload *types.FAQEntryPayload) (*types.FAQChunkMetadata, error) {
	answerStrategy := types.AnswerStrategyAll
	if payload.AnswerStrategy != nil && *payload.AnswerStrategy != "" {
		switch *payload.AnswerStrategy {
		case types.AnswerStrategyAll, types.AnswerStrategyRandom:
			answerStrategy = *payload.AnswerStrategy
		default:
			return nil, werrors.NewBadRequestError("answer_strategy는 'all' 또는 'random'이어야 합니다")
		}
	}
	meta := &types.FAQChunkMetadata{
		StandardQuestion:  strings.TrimSpace(payload.StandardQuestion),
		SimilarQuestions:  payload.SimilarQuestions,
		NegativeQuestions: payload.NegativeQuestions,
		Answers:           payload.Answers,
		AnswerStrategy:    answerStrategy,
		Version:           1,
		Source:            "faq",
	}
	meta.Normalize()
	if meta.StandardQuestion == "" {
		return nil, werrors.NewBadRequestError("표준 질문은 비워둘 수 없습니다")
	}
	if len(meta.Answers) == 0 {
		return nil, werrors.NewBadRequestError("최소 하나의 답변을 제공해야 합니다")
	}
	return meta, nil
}

func buildFAQIndexContent(meta *types.FAQChunkMetadata, mode types.FAQIndexMode) string {
	var builder strings.Builder
	builder.WriteString(meta.StandardQuestion)
	for _, q := range meta.SimilarQuestions {
		builder.WriteString("\n")
		builder.WriteString(q)
	}
	if mode == types.FAQIndexModeQuestionAnswer {
		for _, ans := range meta.Answers {
			builder.WriteString("\n")
			builder.WriteString(ans)
		}
	}
	return builder.String()
}

func (s *knowledgeService) buildFAQIndexInfoList(
	ctx context.Context,
	kb *types.KnowledgeBase,
	chunk *types.Chunk,
) ([]*types.IndexInfo, error) {
	indexMode := types.FAQIndexModeQuestionAnswer
	questionIndexMode := types.FAQQuestionIndexModeCombined
	if kb.FAQConfig != nil {
		if kb.FAQConfig.IndexMode != "" {
			indexMode = kb.FAQConfig.IndexMode
		}
		if kb.FAQConfig.QuestionIndexMode != "" {
			questionIndexMode = kb.FAQConfig.QuestionIndexMode
		}
	}

	meta, err := chunk.FAQMetadata()
	if err != nil {
		return nil, err
	}
	if meta == nil {
		meta = &types.FAQChunkMetadata{StandardQuestion: chunk.Content}
	}

	if questionIndexMode == types.FAQQuestionIndexModeCombined {
		content := buildFAQIndexContent(meta, indexMode)
		return []*types.IndexInfo{
			{
				Content:         content,
				SourceID:        chunk.ID,
				SourceType:      types.ChunkSourceType,
				ChunkID:         chunk.ID,
				KnowledgeID:     chunk.KnowledgeID,
				KnowledgeBaseID: chunk.KnowledgeBaseID,
				KnowledgeType:   types.KnowledgeTypeFAQ,
				IsEnabled:       chunk.IsEnabled,
			},
		}, nil
	}

	indexInfoList := make([]*types.IndexInfo, 0)

	standardContent := meta.StandardQuestion
	if indexMode == types.FAQIndexModeQuestionAnswer && len(meta.Answers) > 0 {
		var builder strings.Builder
		builder.WriteString(meta.StandardQuestion)
		for _, ans := range meta.Answers {
			builder.WriteString("\n")
			builder.WriteString(ans)
		}
		standardContent = builder.String()
	}
	indexInfoList = append(indexInfoList, &types.IndexInfo{
		Content:         standardContent,
		SourceID:        chunk.ID,
		SourceType:      types.ChunkSourceType,
		ChunkID:         chunk.ID,
		KnowledgeID:     chunk.KnowledgeID,
		KnowledgeBaseID: chunk.KnowledgeBaseID,
		KnowledgeType:   types.KnowledgeTypeFAQ,
		IsEnabled:       chunk.IsEnabled,
	})

	for i, similarQ := range meta.SimilarQuestions {
		similarContent := similarQ
		if indexMode == types.FAQIndexModeQuestionAnswer && len(meta.Answers) > 0 {
			var builder strings.Builder
			builder.WriteString(similarQ)
			for _, ans := range meta.Answers {
				builder.WriteString("\n")
				builder.WriteString(ans)
			}
			similarContent = builder.String()
		}
		sourceID := fmt.Sprintf("%s-%d", chunk.ID, i)
		indexInfoList = append(indexInfoList, &types.IndexInfo{
			Content:         similarContent,
			SourceID:        sourceID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         chunk.ID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			KnowledgeType:   types.KnowledgeTypeFAQ,
			IsEnabled:       chunk.IsEnabled,
		})
	}

	return indexInfoList, nil
}

func (s *knowledgeService) indexFAQChunks(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge,
	chunks []*types.Chunk, embeddingModel embedding.Embedder,
	adjustStorage bool, needDelete bool,
) error {
	if len(chunks) == 0 {
		return nil
	}
	indexStartTime := time.Now()
	logger.Debugf(ctx, "indexFAQChunks: starting to index %d chunks", len(chunks))

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		return err
	}

	buildIndexInfoStartTime := time.Now()
	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}
	buildIndexInfoDuration := time.Since(buildIndexInfoStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: built %d index info entries for %d chunks in %v",
		len(indexInfo),
		len(chunks),
		buildIndexInfoDuration,
	)

	var size int64
	if adjustStorage {
		estimateStartTime := time.Now()
		size = retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
		estimateDuration := time.Since(estimateStartTime)
		logger.Debugf(ctx, "indexFAQChunks: estimated storage size %d bytes in %v", size, estimateDuration)
		if tenantInfo.StorageQuota > 0 && tenantInfo.StorageUsed+size > tenantInfo.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}
	}

	var deleteDuration time.Duration
	if needDelete {
		deleteStartTime := time.Now()
		if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
			logger.Warnf(ctx, "Delete FAQ vectors failed: %v", err)
		}
		deleteDuration = time.Since(deleteStartTime)
		if deleteDuration > 100*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: deleted old vectors for %d chunks in %v", len(chunkIDs), deleteDuration)
		}
	}

	batchIndexStartTime := time.Now()
	if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfo); err != nil {
		return err
	}
	batchIndexDuration := time.Since(batchIndexStartTime)
	logger.Debugf(ctx, "indexFAQChunks: batch indexed %d index info entries in %v (avg: %v per entry)",
		len(indexInfo), batchIndexDuration, batchIndexDuration/time.Duration(len(indexInfo)))

	if adjustStorage && size > 0 {
		adjustStartTime := time.Now()
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, size); err == nil {
			tenantInfo.StorageUsed += size
		}
		knowledge.StorageSize += size
		adjustDuration := time.Since(adjustStartTime)
		if adjustDuration > 50*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: adjusted storage in %v", adjustDuration)
		}
	}

	updateStartTime := time.Now()
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	err = s.repo.UpdateKnowledge(ctx, knowledge)
	updateDuration := time.Since(updateStartTime)
	if updateDuration > 50*time.Millisecond {
		logger.Debugf(ctx, "indexFAQChunks: updated knowledge in %v", updateDuration)
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: completed indexing %d chunks in %v (build: %v, delete: %v, batchIndex: %v, update: %v)",
		len(chunks),
		totalDuration,
		buildIndexInfoDuration,
		deleteDuration,
		batchIndexDuration,
		updateDuration,
	)

	return err
}

func (s *knowledgeService) deleteFAQChunkVectors(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, chunks []*types.Chunk,
) error {
	if len(chunks) == 0 {
		return nil
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return err
	}
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		return err
	}

	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}

	size := retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
	if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
		return err
	}
	if size > 0 {
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, -size); err == nil {
			tenantInfo.StorageUsed -= size
			if tenantInfo.StorageUsed < 0 {
				tenantInfo.StorageUsed = 0
			}
		}
		if knowledge.StorageSize >= size {
			knowledge.StorageSize -= size
		} else {
			knowledge.StorageSize = 0
		}
	}
	knowledge.UpdatedAt = time.Now()
	return s.repo.UpdateKnowledge(ctx, knowledge)
}

func ensureManualFileName(title string) string {
	if title == "" {
		return fmt.Sprintf("manual-%s%s", time.Now().Format("20060102-150405"), manualFileExtension)
	}
	trimmed := strings.TrimSpace(title)
	if strings.HasSuffix(strings.ToLower(trimmed), manualFileExtension) {
		return trimmed
	}
	return trimmed + manualFileExtension
}

func (s *knowledgeService) triggerManualProcessing(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, content string, sync bool,
) {
	clean := strings.TrimSpace(content)
	if clean == "" {
		return
	}

	contentBytes := []byte(clean)
	fileName := ensureManualFileName(knowledge.Title)
	fileType := "md"

	enableMultimodel := kb.IsMultimodalEnabled() && kb.StorageConfig.Provider != ""

	var vlmConfig *proto.VLMConfig
	if enableMultimodel {
		cfg, cfgErr := s.getVLMProtoConfig(ctx, kb)
		if cfgErr != nil {
			logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
				WithField("error", cfgErr).Errorf("triggerManualProcessing build VLM config failed")
			knowledge.ParseStatus = "failed"
			knowledge.ErrorMessage = cfgErr.Error()
			knowledge.UpdatedAt = time.Now()
			s.repo.UpdateKnowledge(ctx, knowledge)
			return
		}
		if cfg == nil {
			logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
				Error("triggerManualProcessing enable multimodal but VLM config missing")
		}
		vlmConfig = cfg
	}

	resp, err := s.docReaderClient.ReadFromFile(ctx, &proto.ReadFromFileRequest{
		FileContent: contentBytes,
		FileName:    fileName,
		FileType:    fileType,
		ReadConfig: &proto.ReadConfig{
			ChunkSize:        int32(kb.ChunkingConfig.ChunkSize),
			ChunkOverlap:     int32(kb.ChunkingConfig.ChunkOverlap),
			Separators:       kb.ChunkingConfig.Separators,
			EnableMultimodal: enableMultimodel,
			StorageConfig: &proto.StorageConfig{
				Provider: proto.StorageProvider(
					proto.StorageProvider_value[strings.ToUpper(kb.StorageConfig.Provider)],
				),
				Region:          kb.StorageConfig.Region,
				BucketName:      kb.StorageConfig.BucketName,
				AccessKeyId:     kb.StorageConfig.SecretID,
				SecretAccessKey: kb.StorageConfig.SecretKey,
				AppId:           kb.StorageConfig.AppID,
				PathPrefix:      kb.StorageConfig.PathPrefix,
			},
			VlmConfig: vlmConfig,
		},
		RequestId: ctx.Value(types.RequestIDContextKey).(string),
	})
	if err != nil {
		logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
			WithField("error", err).Errorf("triggerManualProcessing read file failed")
		knowledge.ParseStatus = "failed"
		knowledge.ErrorMessage = err.Error()
		knowledge.UpdatedAt = time.Now()
		s.repo.UpdateKnowledge(ctx, knowledge)
		return
	}

	if sync {
		s.processChunks(ctx, kb, knowledge, resp.Chunks)
		return
	}

	newCtx := logger.CloneContext(ctx)
	go s.processChunks(newCtx, kb, knowledge, resp.Chunks)
}

func (s *knowledgeService) cleanupKnowledgeResources(ctx context.Context, knowledge *types.Knowledge) error {
	logger.GetLogger(ctx).Infof("Cleaning knowledge resources before manual update, knowledge ID: %s", knowledge.ID)

	var cleanupErr error

	if knowledge.ParseStatus == types.ManualKnowledgeStatusDraft && knowledge.StorageSize == 0 {
		// Draft without indexed data, skip cleanup.
		return nil
	}

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	if knowledge.EmbeddingModelID != "" {
		retrieveEngine, err := retriever.NewCompositeRetrieveEngine(
			s.retrieveEngine,
			tenantInfo.GetEffectiveEngines(),
		)
		if err != nil {
			logger.GetLogger(ctx).WithField("error", err).Error("Failed to init retrieve engine during cleanup")
			cleanupErr = errors.Join(cleanupErr, err)
		} else {
			embeddingModel, modelErr := s.modelService.GetEmbeddingModel(ctx, knowledge.EmbeddingModelID)
			if modelErr != nil {
				logger.GetLogger(ctx).WithField("error", modelErr).Error("Failed to get embedding model during cleanup")
				cleanupErr = errors.Join(cleanupErr, modelErr)
			} else {
				if err := retrieveEngine.DeleteByKnowledgeIDList(ctx, []string{knowledge.ID}, embeddingModel.GetDimensions(), knowledge.Type); err != nil {
					logger.GetLogger(ctx).WithField("error", err).Error("Failed to delete manual knowledge index")
					cleanupErr = errors.Join(cleanupErr, err)
				}
			}
		}
	}

	if err := s.chunkService.DeleteChunksByKnowledgeID(ctx, knowledge.ID); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Error("Failed to delete manual knowledge chunks")
		cleanupErr = errors.Join(cleanupErr, err)
	}

	namespace := types.NameSpace{KnowledgeBase: knowledge.KnowledgeBaseID, Knowledge: knowledge.ID}
	if err := s.graphEngine.DelGraph(ctx, []types.NameSpace{namespace}); err != nil {
		logger.GetLogger(ctx).WithField("error", err).Error("Failed to delete manual knowledge graph data")
		cleanupErr = errors.Join(cleanupErr, err)
	}

	if knowledge.StorageSize > 0 {
		tenantInfo.StorageUsed -= knowledge.StorageSize
		if tenantInfo.StorageUsed < 0 {
			tenantInfo.StorageUsed = 0
		}
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, -knowledge.StorageSize); err != nil {
			logger.GetLogger(ctx).WithField("error", err).Error("Failed to adjust storage usage during manual cleanup")
			cleanupErr = errors.Join(cleanupErr, err)
		}
		knowledge.StorageSize = 0
	}

	return cleanupErr
}

func (s *knowledgeService) getVLMProtoConfig(ctx context.Context, kb *types.KnowledgeBase) (*proto.VLMConfig, error) {
	if kb == nil {
		return nil, nil
	}
	if kb.VLMConfig.ModelName != "" && kb.VLMConfig.BaseURL != "" {
		return &proto.VLMConfig{
			ModelName:     kb.VLMConfig.ModelName,
			BaseUrl:       kb.VLMConfig.BaseURL,
			ApiKey:        kb.VLMConfig.APIKey,
			InterfaceType: kb.VLMConfig.InterfaceType,
		}, nil
	}

	if !kb.VLMConfig.Enabled || kb.VLMConfig.ModelID == "" {
		return nil, nil
	}

	model, err := s.modelService.GetModelByID(ctx, kb.VLMConfig.ModelID)
	if err != nil {
		return nil, err
	}

	interfaceType := model.Parameters.InterfaceType
	if interfaceType == "" {
		interfaceType = "openai"
	}

	return &proto.VLMConfig{
		ModelName:     model.Name,
		BaseUrl:       model.Parameters.BaseURL,
		ApiKey:        model.Parameters.APIKey,
		InterfaceType: interfaceType,
	}, nil
}

func IsImageType(fileType string) bool {
	switch fileType {
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "tiff":
		return true
	default:
		return false
	}
}

// ProcessDocument handles Asynq document processing tasks
func (s *knowledgeService) ProcessDocument(ctx context.Context, t *asynq.Task) error {
	var payload types.DocumentProcessPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "failed to unmarshal document process task payload: %v", err)
		return nil
	}

	ctx = logger.WithRequestID(ctx, payload.RequestId)
	ctx = logger.WithField(ctx, "document_process", payload.KnowledgeID)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)

	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	isLastRetry := retryCount >= maxRetry

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "failed to get tenant: %v", err)
		return nil
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	logger.Infof(ctx, "Processing document task: knowledge_id=%s, file_path=%s, retry=%d/%d",
		payload.KnowledgeID, payload.FilePath, retryCount, maxRetry)

	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "failed to get knowledge: %v", err)
		return nil
	}

	if knowledge == nil {
		return nil
	}

	if knowledge.ParseStatus == types.ParseStatusDeleting {
		logger.Infof(ctx, "Knowledge is being deleted, aborting processing: %s", payload.KnowledgeID)
		return nil
	}

	if knowledge.ParseStatus == types.ParseStatusCompleted {
		logger.Infof(ctx, "Document already completed, skipping: %s", payload.KnowledgeID)
		return nil
	}

	if knowledge.ParseStatus == types.ParseStatusFailed {
		logger.Warnf(
			ctx,
			"Document processing previously failed: %s, error: %s",
			payload.KnowledgeID,
			knowledge.ErrorMessage,
		)
	}

	if knowledge.ParseStatus != "completed" && knowledge.ParseStatus != "pending" &&
		knowledge.ParseStatus != "processing" {
		logger.Warnf(ctx, "Unexpected parse status: %s for knowledge: %s", knowledge.ParseStatus, payload.KnowledgeID)
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "failed to get knowledge base: %v", err)
		knowledge.ParseStatus = "failed"
		knowledge.ErrorMessage = fmt.Sprintf("failed to get knowledge base: %v", err)
		knowledge.UpdatedAt = time.Now()
		s.repo.UpdateKnowledge(ctx, knowledge)
		return nil
	}

	knowledge.ParseStatus = "processing"
	knowledge.UpdatedAt = time.Now()
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		logger.Errorf(ctx, "failed to update knowledge status to processing: %v", err)
		return nil
	}

	var vlmConfig *proto.VLMConfig
	if payload.EnableMultimodel {
		vlmConfig, err = s.getVLMProtoConfig(ctx, kb)
		if err != nil {
			logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
				WithField("error", err).Errorf("processDocument build VLM config failed")
		}
		if vlmConfig == nil {
			logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
				Warn("processDocument enable multimodal but VLM config missing")
		}
	}

	if payload.FilePath != "" && !payload.EnableMultimodel && IsImageType(payload.FileType) {
		logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
			WithField("error", ErrImageNotParse).Errorf("processDocument image without enable multimodel")
		knowledge.ParseStatus = "failed"
		knowledge.ErrorMessage = ErrImageNotParse.Error()
		knowledge.UpdatedAt = time.Now()
		s.repo.UpdateKnowledge(ctx, knowledge)
		return nil
	}

	var chunks []*proto.Chunk
	if payload.URL != "" {
		urlResp, err := s.docReaderClient.ReadFromURL(ctx, &proto.ReadFromURLRequest{
			Url:   payload.URL,
			Title: knowledge.Title,
			ReadConfig: &proto.ReadConfig{
				ChunkSize:        int32(kb.ChunkingConfig.ChunkSize),
				ChunkOverlap:     int32(kb.ChunkingConfig.ChunkOverlap),
				Separators:       kb.ChunkingConfig.Separators,
				EnableMultimodal: payload.EnableMultimodel,
				StorageConfig: &proto.StorageConfig{
					Provider: proto.StorageProvider(
						proto.StorageProvider_value[strings.ToUpper(kb.StorageConfig.Provider)],
					),
					Region:          kb.StorageConfig.Region,
					BucketName:      kb.StorageConfig.BucketName,
					AccessKeyId:     kb.StorageConfig.SecretID,
					SecretAccessKey: kb.StorageConfig.SecretKey,
					AppId:           kb.StorageConfig.AppID,
					PathPrefix:      kb.StorageConfig.PathPrefix,
				},
				VlmConfig: vlmConfig,
			},
			RequestId: payload.RequestId,
		})
		if err != nil {
			if isLastRetry {
				knowledge.ParseStatus = "failed"
				knowledge.ErrorMessage = err.Error()
				knowledge.UpdatedAt = time.Now()
				s.repo.UpdateKnowledge(ctx, knowledge)
			}
			return fmt.Errorf("failed to read from URL: %w", err)
		}
		chunks = urlResp.Chunks
	} else if len(payload.Passages) > 0 {
		chunks := make([]*proto.Chunk, 0, len(payload.Passages))
		start, end := 0, 0
		for i, p := range payload.Passages {
			if p == "" {
				continue
			}
			end += len([]rune(p))
			chunk := &proto.Chunk{
				Content: p,
				Seq:     int32(i),
				Start:   int32(start),
				End:     int32(end),
			}
			start = end
			chunks = append(chunks, chunk)
		}
		s.processChunks(ctx, kb, knowledge, chunks)
		return nil
	} else {
		fileReader, err := s.fileSvc.GetFile(ctx, payload.FilePath)
		if err != nil {
			logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
				WithField("error", err).Errorf("processDocument get file failed")
			if isLastRetry {
				knowledge.ParseStatus = "failed"
				knowledge.ErrorMessage = err.Error()
				knowledge.UpdatedAt = time.Now()
				s.repo.UpdateKnowledge(ctx, knowledge)
			}
			return fmt.Errorf("failed to get file: %w", err)
		}
		defer fileReader.Close()

		contentBytes, err := io.ReadAll(fileReader)
		if err != nil {
			if isLastRetry {
				knowledge.ParseStatus = "failed"
				knowledge.ErrorMessage = err.Error()
				knowledge.UpdatedAt = time.Now()
				s.repo.UpdateKnowledge(ctx, knowledge)
			}
			return fmt.Errorf("failed to read file: %w", err)
		}

		fileResp, err := s.docReaderClient.ReadFromFile(ctx, &proto.ReadFromFileRequest{
			FileContent: contentBytes,
			FileName:    payload.FileName,
			FileType:    payload.FileType,
			ReadConfig: &proto.ReadConfig{
				ChunkSize:        int32(kb.ChunkingConfig.ChunkSize),
				ChunkOverlap:     int32(kb.ChunkingConfig.ChunkOverlap),
				Separators:       kb.ChunkingConfig.Separators,
				EnableMultimodal: payload.EnableMultimodel,
				StorageConfig: &proto.StorageConfig{
					Provider:        proto.StorageProvider(proto.StorageProvider_value[strings.ToUpper(kb.StorageConfig.Provider)]),
					Region:          kb.StorageConfig.Region,
					BucketName:      kb.StorageConfig.BucketName,
					AccessKeyId:     kb.StorageConfig.SecretID,
					SecretAccessKey: kb.StorageConfig.SecretKey,
					AppId:           kb.StorageConfig.AppID,
					PathPrefix:      kb.StorageConfig.PathPrefix,
				},
				VlmConfig: vlmConfig,
			},
			RequestId: payload.RequestId,
		})
		if err != nil {
			logger.GetLogger(ctx).WithField("knowledge_id", knowledge.ID).
				WithField("error", err).Errorf("processDocument read file failed")
			if isLastRetry {
				knowledge.ParseStatus = "failed"
				knowledge.ErrorMessage = err.Error()
				knowledge.UpdatedAt = time.Now()
				s.repo.UpdateKnowledge(ctx, knowledge)
			}
			return fmt.Errorf("failed to read file from docreader: %w", err)
		}
		chunks = fileResp.Chunks
	}

	s.processChunks(ctx, kb, knowledge, chunks, ProcessChunksOptions{
		EnableQuestionGeneration: payload.EnableQuestionGeneration,
		QuestionCount:            payload.QuestionCount,
	})

	return nil
}

// ProcessFAQImport handles Asynq FAQ import tasks
func (s *knowledgeService) ProcessFAQImport(ctx context.Context, t *asynq.Task) error {
	var payload types.FAQImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "failed to unmarshal FAQ import task payload: %v", err)
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}

	ctx = logger.WithRequestID(ctx, uuid.New().String())
	ctx = logger.WithField(ctx, "faq_import", payload.TaskID)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)

	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	isLastRetry := retryCount >= maxRetry

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "failed to get tenant: %v", err)
		return nil
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	logger.Infof(ctx, "Processing FAQ import task: task_id=%s, kb_id=%s, total_entries=%d, retry=%d/%d",
		payload.TaskID, payload.KBID, len(payload.Entries), retryCount, maxRetry)

	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.TaskID)
	if err != nil {
		logger.Errorf(ctx, "failed to get FAQ knowledge: %v", err)
		return nil
	}

	if knowledge == nil {
		return nil
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KBID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base: %v", err)
		if isLastRetry {
			if updateErr := s.updateFAQImportStatus(ctx, payload.TaskID, types.FAQImportStatusFailed, 0, len(payload.Entries), 0, err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
		}
		return fmt.Errorf("failed to get knowledge base: %w", err)
	}

	if knowledge.ParseStatus == "completed" {
		logger.Infof(ctx, "FAQ import already completed, skipping: %s", payload.TaskID)
		return nil
	}

	importMeta, _ := types.ParseFAQImportMetadata(knowledge)
	var processedCount int
	if importMeta != nil {
		processedCount = importMeta.ImportProcessed
		logger.Infof(ctx, "Resuming FAQ import from progress: %d/%d", processedCount, len(payload.Entries))
	}

	originalTotalEntries := len(payload.Entries)

	if processedCount < originalTotalEntries {
		chunksDeleted, err := s.chunkRepo.DeleteUnindexedChunks(ctx, payload.TenantID, payload.KnowledgeID)
		if err != nil {
			logger.Errorf(ctx, "Failed to delete unindexed chunks: %v", err)
			if isLastRetry {
				if updateErr := s.updateFAQImportStatus(ctx, payload.TaskID, types.FAQImportStatusFailed, 0, originalTotalEntries, processedCount, err.Error()); updateErr != nil {
					logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
				}
			}
			return fmt.Errorf("failed to delete unindexed chunks: %w", err)
		}
		logger.Infof(ctx, "Deleted unindexed chunks: %d", len(chunksDeleted))

		embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
		if err == nil {
			retrieveEngine, err := retriever.NewCompositeRetrieveEngine(
				s.retrieveEngine,
				tenantInfo.GetEffectiveEngines(),
			)
			if err == nil {
				chunkIDs := make([]string, 0, len(chunksDeleted))
				for _, chunk := range chunksDeleted {
					chunkIDs = append(chunkIDs, chunk.ID)
				}
				if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
					logger.Warnf(ctx, "Failed to delete index data for chunks (may not exist): %v", err)
				} else {
					logger.Infof(ctx, "Successfully deleted index data for %d chunks", len(chunksDeleted))
				}
			}
		}

		if payload.Mode == types.FAQBatchModeAppend {
			payload.Entries = payload.Entries[processedCount:]
		}
		logger.Infof(
			ctx,
			"Continuing FAQ import from entry %d, remaining: %d entries",
			processedCount,
			len(payload.Entries),
		)
	}

	if err := s.updateFAQImportStatusWithRanges(ctx, payload.TaskID, types.FAQImportStatusProcessing, 0,
		originalTotalEntries, processedCount, ""); err != nil {
		logger.Errorf(ctx, "Failed to update task status to running: %v", err)
	}

	faqPayload := &types.FAQBatchUpsertPayload{
		Entries: payload.Entries,
		Mode:    payload.Mode,
	}

	if err := s.executeFAQImport(ctx, payload.TaskID, payload.KBID, faqPayload, payload.TenantID, originalTotalEntries-len(payload.Entries)); err != nil {
		logger.Errorf(ctx, "FAQ import task failed: %s, error: %v", payload.TaskID, err)
		if isLastRetry {
			currentMeta, _ := types.ParseFAQImportMetadata(knowledge)
			currentProcessed := 0
			if currentMeta != nil {
				currentProcessed = currentMeta.ImportProcessed
			}
			if updateErr := s.updateFAQImportStatus(ctx, payload.TaskID, types.FAQImportStatusFailed, 0, originalTotalEntries, currentProcessed, err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
		}
		return fmt.Errorf("FAQ import failed: %w", err)
	}

	logger.Infof(ctx, "FAQ import task completed: %s", payload.TaskID)
	if err := s.updateFAQImportStatus(ctx, payload.TaskID, types.FAQImportStatusCompleted, 100, originalTotalEntries, originalTotalEntries, ""); err != nil {
		logger.Errorf(ctx, "Failed to update task status to success: %v", err)
	}

	return nil
}

const (
	kbCloneProgressKeyPrefix = "kb_clone_progress:"
	kbCloneProgressTTL       = 24 * time.Hour
)

// getKBCloneProgressKey returns the Redis key for storing KB clone progress
func getKBCloneProgressKey(taskID string) string {
	return kbCloneProgressKeyPrefix + taskID
}

// ProcessKBClone handles Asynq knowledge base clone tasks
func (s *knowledgeService) ProcessKBClone(ctx context.Context, t *asynq.Task) error {
	var payload types.KBClonePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal KB clone payload: %w", err)
	}

	// Add tenant ID to context
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)

	// Get tenant info and add to context
	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get tenant info: %v", err)
		return fmt.Errorf("failed to get tenant info: %w", err)
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	// Check if this is the last retry
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	isLastRetry := retryCount >= maxRetry

	logger.Infof(ctx, "Processing KB clone task: %s, source: %s, target: %s, retry: %d/%d",
		payload.TaskID, payload.SourceID, payload.TargetID, retryCount, maxRetry)

	// Helper function to handle errors - only mark as failed on last retry
	handleError := func(progress *types.KBCloneProgress, err error, message string) {
		if isLastRetry {
			progress.Status = types.KBCloneStatusFailed
			progress.Error = err.Error()
			progress.Message = message
			progress.UpdatedAt = time.Now().Unix()
			_ = s.saveKBCloneProgress(ctx, progress)
		}
	}

	// Update progress to processing
	progress := &types.KBCloneProgress{
		TaskID:    payload.TaskID,
		SourceID:  payload.SourceID,
		TargetID:  payload.TargetID,
		Status:    types.KBCloneStatusProcessing,
		Progress:  0,
		Message:   "Starting knowledge base clone...",
		UpdatedAt: time.Now().Unix(),
	}
	if err := s.saveKBCloneProgress(ctx, progress); err != nil {
		logger.Errorf(ctx, "Failed to update KB clone progress: %v", err)
	}

	// Get source and target knowledge bases
	srcKB, dstKB, err := s.kbService.CopyKnowledgeBase(ctx, payload.SourceID, payload.TargetID)
	if err != nil {
		logger.Errorf(ctx, "Failed to copy knowledge base: %v", err)
		handleError(progress, err, "Failed to copy knowledge base configuration")
		return err
	}

	// Use different sync strategies based on knowledge base type
	if srcKB.Type == types.KnowledgeBaseTypeFAQ {
		return s.cloneFAQKnowledgeBase(ctx, srcKB, dstKB, progress, handleError)
	}

	// Document type: use Knowledge-level diff based on file_hash
	addKnowledge, err := s.repo.AminusB(ctx, srcKB.TenantID, srcKB.ID, dstKB.TenantID, dstKB.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge to add: %v", err)
		handleError(progress, err, "Failed to calculate knowledge difference")
		return err
	}

	delKnowledge, err := s.repo.AminusB(ctx, dstKB.TenantID, dstKB.ID, srcKB.TenantID, srcKB.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge to delete: %v", err)
		handleError(progress, err, "Failed to calculate knowledge difference")
		return err
	}

	totalOperations := len(addKnowledge) + len(delKnowledge)
	progress.Total = totalOperations
	progress.Message = fmt.Sprintf("Found %d knowledge to add, %d to delete", len(addKnowledge), len(delKnowledge))
	progress.UpdatedAt = time.Now().Unix()
	_ = s.saveKBCloneProgress(ctx, progress)

	logger.Infof(ctx, "Knowledge after update to add: %d, delete: %d", len(addKnowledge), len(delKnowledge))

	processedCount := 0
	batch := 10

	// Delete knowledge in target that doesn't exist in source
	g, gctx := errgroup.WithContext(ctx)
	for ids := range slices.Chunk(delKnowledge, batch) {
		g.Go(func() error {
			err := s.DeleteKnowledgeList(gctx, ids)
			if err != nil {
				logger.Errorf(gctx, "delete partial knowledge %v: %v", ids, err)
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		logger.Errorf(ctx, "delete total knowledge %d: %v", len(delKnowledge), err)
		handleError(progress, err, "Failed to delete knowledge")
		return err
	}

	processedCount += len(delKnowledge)
	if totalOperations > 0 {
		progress.Progress = processedCount * 100 / totalOperations
	}
	progress.Processed = processedCount
	progress.Message = fmt.Sprintf("Deleted %d knowledge, cloning %d...", len(delKnowledge), len(addKnowledge))
	progress.UpdatedAt = time.Now().Unix()
	_ = s.saveKBCloneProgress(ctx, progress)

	// Clone knowledge from source to target
	g, gctx = errgroup.WithContext(ctx)
	g.SetLimit(batch)
	for _, knowledge := range addKnowledge {
		g.Go(func() error {
			srcKn, err := s.repo.GetKnowledgeByID(gctx, srcKB.TenantID, knowledge)
			if err != nil {
				logger.Errorf(gctx, "get knowledge %s: %v", knowledge, err)
				return err
			}
			err = s.cloneKnowledge(gctx, srcKn, dstKB)
			if err != nil {
				logger.Errorf(gctx, "clone knowledge %s: %v", knowledge, err)
				return err
			}

			// Update progress
			processedCount++
			if totalOperations > 0 {
				progress.Progress = processedCount * 100 / totalOperations
			}
			progress.Processed = processedCount
			progress.Message = fmt.Sprintf("Cloned %d/%d knowledge", processedCount-len(delKnowledge), len(addKnowledge))
			progress.UpdatedAt = time.Now().Unix()
			_ = s.saveKBCloneProgress(ctx, progress)

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		logger.Errorf(ctx, "add total knowledge %d: %v", len(addKnowledge), err)
		handleError(progress, err, "Failed to clone knowledge")
		return err
	}

	// Mark as completed
	progress.Status = types.KBCloneStatusCompleted
	progress.Progress = 100
	progress.Processed = totalOperations
	progress.Message = "Knowledge base clone completed successfully"
	progress.UpdatedAt = time.Now().Unix()
	if err := s.saveKBCloneProgress(ctx, progress); err != nil {
		logger.Errorf(ctx, "Failed to update KB clone progress to completed: %v", err)
	}

	logger.Infof(ctx, "KB clone task completed: %s", payload.TaskID)
	return nil
}

// cloneFAQKnowledgeBase handles FAQ knowledge base cloning with chunk-level incremental sync
func (s *knowledgeService) cloneFAQKnowledgeBase(
	ctx context.Context,
	srcKB, dstKB *types.KnowledgeBase,
	progress *types.KBCloneProgress,
	handleError func(*types.KBCloneProgress, error, string),
) error {
	// Get source FAQ knowledge first (FAQ KB has exactly one Knowledge entry)
	srcKnowledgeList, err := s.repo.ListKnowledgeByKnowledgeBaseID(ctx, srcKB.TenantID, srcKB.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get source FAQ knowledge: %v", err)
		handleError(progress, err, "Failed to get source FAQ knowledge")
		return err
	}
	if len(srcKnowledgeList) == 0 {
		// Source has no FAQ knowledge, nothing to clone
		progress.Status = types.KBCloneStatusCompleted
		progress.Progress = 100
		progress.Message = "Source FAQ knowledge base is empty"
		progress.UpdatedAt = time.Now().Unix()
		_ = s.saveKBCloneProgress(ctx, progress)
		return nil
	}
	srcKnowledge := srcKnowledgeList[0]

	// Get chunk-level differences based on content_hash
	chunksToAdd, chunksToDelete, err := s.chunkRepo.FAQChunkDiff(ctx, srcKB.TenantID, srcKB.ID, dstKB.TenantID, dstKB.ID)
	if err != nil {
		logger.Errorf(ctx, "Failed to calculate FAQ chunk difference: %v", err)
		handleError(progress, err, "Failed to calculate FAQ chunk difference")
		return err
	}

	totalOperations := len(chunksToAdd) + len(chunksToDelete)
	progress.Total = totalOperations
	progress.Message = fmt.Sprintf("Found %d FAQ entries to add, %d to delete", len(chunksToAdd), len(chunksToDelete))
	progress.UpdatedAt = time.Now().Unix()
	_ = s.saveKBCloneProgress(ctx, progress)

	logger.Infof(ctx, "FAQ chunks to add: %d, delete: %d", len(chunksToAdd), len(chunksToDelete))

	// If nothing to do, mark as completed
	if totalOperations == 0 {
		progress.Status = types.KBCloneStatusCompleted
		progress.Progress = 100
		progress.Message = "FAQ knowledge base is already in sync"
		progress.UpdatedAt = time.Now().Unix()
		_ = s.saveKBCloneProgress(ctx, progress)
		return nil
	}

	// Get tenant info and initialize retrieve engine
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.NewCompositeRetrieveEngine(s.retrieveEngine, tenantInfo.GetEffectiveEngines())
	if err != nil {
		logger.Errorf(ctx, "Failed to init retrieve engine: %v", err)
		handleError(progress, err, "Failed to initialize retrieve engine")
		return err
	}

	// Get embedding model
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, dstKB.EmbeddingModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get embedding model: %v", err)
		handleError(progress, err, "Failed to get embedding model")
		return err
	}

	processedCount := 0

	// Delete FAQ chunks that don't exist in source
	if len(chunksToDelete) > 0 {
		// Delete from vector store
		if err := retrieveEngine.DeleteByChunkIDList(ctx, chunksToDelete, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
			logger.Errorf(ctx, "Failed to delete FAQ chunks from vector store: %v", err)
			handleError(progress, err, "Failed to delete FAQ entries from vector store")
			return err
		}
		// Delete from database
		if err := s.chunkRepo.DeleteChunks(ctx, dstKB.TenantID, chunksToDelete); err != nil {
			logger.Errorf(ctx, "Failed to delete FAQ chunks from database: %v", err)
			handleError(progress, err, "Failed to delete FAQ entries from database")
			return err
		}
		processedCount += len(chunksToDelete)
		if totalOperations > 0 {
			progress.Progress = processedCount * 100 / totalOperations
		}
		progress.Processed = processedCount
		progress.Message = fmt.Sprintf("Deleted %d FAQ entries, adding %d...", len(chunksToDelete), len(chunksToAdd))
		progress.UpdatedAt = time.Now().Unix()
		_ = s.saveKBCloneProgress(ctx, progress)
	}

	// Get or create the FAQ knowledge entry in destination
	dstKnowledge, err := s.getOrCreateFAQKnowledge(ctx, dstKB, srcKnowledge)
	if err != nil {
		logger.Errorf(ctx, "Failed to get or create FAQ knowledge: %v", err)
		handleError(progress, err, "Failed to prepare FAQ knowledge entry")
		return err
	}

	// Clone FAQ chunks from source to destination
	batch := 50
	tagIDMapping := map[string]string{} // srcTagID -> dstTagID
	for i := 0; i < len(chunksToAdd); i += batch {
		end := i + batch
		if end > len(chunksToAdd) {
			end = len(chunksToAdd)
		}
		batchIDs := chunksToAdd[i:end]

		// Get source chunks
		srcChunks, err := s.chunkRepo.ListChunksByID(ctx, srcKB.TenantID, batchIDs)
		if err != nil {
			logger.Errorf(ctx, "Failed to get source FAQ chunks: %v", err)
			handleError(progress, err, "Failed to get source FAQ entries")
			return err
		}

		// Create new chunks for destination
		newChunks := make([]*types.Chunk, 0, len(srcChunks))
		for _, srcChunk := range srcChunks {
			// Map TagID to target knowledge base
			targetTagID := ""
			if srcChunk.TagID != "" {
				if mappedTagID, ok := tagIDMapping[srcChunk.TagID]; ok {
					targetTagID = mappedTagID
				} else {
					// Try to find or create the tag in target knowledge base
					targetTagID = s.getOrCreateTagInTarget(ctx, srcKB.TenantID, dstKB.TenantID, dstKB.ID, srcChunk.TagID, tagIDMapping)
				}
			}

			newChunk := &types.Chunk{
				ID:              uuid.New().String(),
				TenantID:        dstKB.TenantID,
				KnowledgeID:     dstKnowledge.ID,
				KnowledgeBaseID: dstKB.ID,
				TagID:           targetTagID,
				Content:         srcChunk.Content,
				ChunkIndex:      srcChunk.ChunkIndex,
				IsEnabled:       srcChunk.IsEnabled,
				Flags:           srcChunk.Flags,
				ChunkType:       types.ChunkTypeFAQ,
				Metadata:        srcChunk.Metadata,
				ContentHash:     srcChunk.ContentHash,
				ImageInfo:       srcChunk.ImageInfo,
				Status:          int(types.ChunkStatusStored), // Initially stored, will be indexed
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			newChunks = append(newChunks, newChunk)
		}

		// Save to database
		if err := s.chunkRepo.CreateChunks(ctx, newChunks); err != nil {
			logger.Errorf(ctx, "Failed to create FAQ chunks: %v", err)
			handleError(progress, err, "Failed to create FAQ entries")
			return err
		}

		// Index in vector store using existing method
		// This will index standard question + similar questions based on FAQConfig
		if err := s.indexFAQChunks(ctx, dstKB, dstKnowledge, newChunks, embeddingModel, false, false); err != nil {
			logger.Errorf(ctx, "Failed to index FAQ chunks: %v", err)
			handleError(progress, err, "Failed to index FAQ entries")
			return err
		}

		// Update chunk status to indexed
		for _, chunk := range newChunks {
			chunk.Status = int(types.ChunkStatusIndexed)
		}
		if err := s.chunkService.UpdateChunks(ctx, newChunks); err != nil {
			logger.Warnf(ctx, "Failed to update FAQ chunks status: %v", err)
			// Don't fail the whole operation for status update failure
		}

		processedCount += len(batchIDs)
		if totalOperations > 0 {
			progress.Progress = processedCount * 100 / totalOperations
		}
		progress.Processed = processedCount
		progress.Message = fmt.Sprintf("Added %d/%d FAQ entries", processedCount-len(chunksToDelete), len(chunksToAdd))
		progress.UpdatedAt = time.Now().Unix()
		_ = s.saveKBCloneProgress(ctx, progress)
	}

	// Mark as completed
	progress.Status = types.KBCloneStatusCompleted
	progress.Progress = 100
	progress.Processed = totalOperations
	progress.Message = "FAQ knowledge base clone completed successfully"
	progress.UpdatedAt = time.Now().Unix()
	if err := s.saveKBCloneProgress(ctx, progress); err != nil {
		logger.Errorf(ctx, "Failed to update KB clone progress to completed: %v", err)
	}

	return nil
}

// getOrCreateFAQKnowledge gets or creates the FAQ knowledge entry for a knowledge base
// If srcKnowledge is provided, it will copy relevant fields from source when creating new knowledge
func (s *knowledgeService) getOrCreateFAQKnowledge(ctx context.Context, kb *types.KnowledgeBase, srcKnowledge *types.Knowledge) (*types.Knowledge, error) {
	// FAQ knowledge base should have exactly one Knowledge entry
	knowledgeList, err := s.repo.ListKnowledgeByKnowledgeBaseID(ctx, kb.TenantID, kb.ID)
	if err != nil {
		return nil, err
	}

	if len(knowledgeList) > 0 {
		return knowledgeList[0], nil
	}

	// Create a new FAQ knowledge entry, copying from source if available
	knowledge := &types.Knowledge{
		ID:               uuid.New().String(),
		TenantID:         kb.TenantID,
		KnowledgeBaseID:  kb.ID,
		Type:             types.KnowledgeTypeFAQ,
		Title:            "FAQ",
		ParseStatus:      "completed",
		EnableStatus:     "enabled",
		EmbeddingModelID: kb.EmbeddingModelID,
	}

	// Copy additional fields from source knowledge if available
	if srcKnowledge != nil {
		knowledge.Title = srcKnowledge.Title
		knowledge.Description = srcKnowledge.Description
		knowledge.Source = srcKnowledge.Source
		knowledge.Metadata = srcKnowledge.Metadata
	}

	if err := s.repo.CreateKnowledge(ctx, knowledge); err != nil {
		return nil, err
	}
	return knowledge, nil
}

// saveKBCloneProgress saves the KB clone progress to Redis
func (s *knowledgeService) saveKBCloneProgress(ctx context.Context, progress *types.KBCloneProgress) error {
	key := getKBCloneProgressKey(progress.TaskID)
	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, kbCloneProgressTTL).Err()
}

// SaveKBCloneProgress saves the KB clone progress to Redis (public method for handler use)
func (s *knowledgeService) SaveKBCloneProgress(ctx context.Context, progress *types.KBCloneProgress) error {
	return s.saveKBCloneProgress(ctx, progress)
}

// GetKBCloneProgress retrieves the progress of a knowledge base clone task
func (s *knowledgeService) GetKBCloneProgress(ctx context.Context, taskID string) (*types.KBCloneProgress, error) {
	key := getKBCloneProgressKey(taskID)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, werrors.NewNotFoundError("KB clone task not found")
		}
		return nil, fmt.Errorf("failed to get progress from Redis: %w", err)
	}

	var progress types.KBCloneProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal progress: %w", err)
	}
	return &progress, nil
}

// getOrCreateTagInTarget finds or creates a tag in the target knowledge base based on the source tag.
// It looks up the source tag by ID, then tries to find a tag with the same name in the target KB.
// If not found, it creates a new tag with the same properties.
// The mapping is cached in tagIDMapping for subsequent lookups.
func (s *knowledgeService) getOrCreateTagInTarget(
	ctx context.Context,
	srcTenantID, dstTenantID uint64,
	dstKnowledgeBaseID string,
	srcTagID string,
	tagIDMapping map[string]string,
) string {
	// Get source tag
	srcTag, err := s.tagRepo.GetByID(ctx, srcTenantID, srcTagID)
	if err != nil || srcTag == nil {
		logger.Warnf(ctx, "Failed to get source tag %s: %v", srcTagID, err)
		tagIDMapping[srcTagID] = "" // Cache empty result to avoid repeated lookups
		return ""
	}

	// Try to find existing tag with same name in target KB
	dstTag, err := s.tagRepo.GetByName(ctx, dstTenantID, dstKnowledgeBaseID, srcTag.Name)
	if err == nil && dstTag != nil {
		tagIDMapping[srcTagID] = dstTag.ID
		return dstTag.ID
	}

	// Create new tag in target KB
	newTag := &types.KnowledgeTag{
		ID:              uuid.New().String(),
		TenantID:        dstTenantID,
		KnowledgeBaseID: dstKnowledgeBaseID,
		Name:            srcTag.Name,
		Color:           srcTag.Color,
		SortOrder:       srcTag.SortOrder,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := s.tagRepo.Create(ctx, newTag); err != nil {
		logger.Warnf(ctx, "Failed to create tag %s in target KB: %v", srcTag.Name, err)
		tagIDMapping[srcTagID] = "" // Cache empty result
		return ""
	}

	tagIDMapping[srcTagID] = newTag.ID
	logger.Infof(ctx, "Created tag %s (ID: %s) in target KB %s", newTag.Name, newTag.ID, dstKnowledgeBaseID)
	return newTag.ID
}

package service

import (
	"context"
	"errors"
	"strings"
	"time"

	werrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// knowledgeTagService implements KnowledgeTagService.
type knowledgeTagService struct {
	kbService interfaces.KnowledgeBaseService
	repo      interfaces.KnowledgeTagRepository
	chunkRepo interfaces.ChunkRepository
}

// NewKnowledgeTagService creates a new tag service.
func NewKnowledgeTagService(
	kbService interfaces.KnowledgeBaseService,
	repo interfaces.KnowledgeTagRepository,
	chunkRepo interfaces.ChunkRepository,
) (interfaces.KnowledgeTagService, error) {
	return &knowledgeTagService{
		kbService: kbService,
		repo:      repo,
		chunkRepo: chunkRepo,
	}, nil
}

// ListTags lists all tags for a knowledge base with usage stats.
func (s *knowledgeTagService) ListTags(
	ctx context.Context,
	kbID string,
	page *types.Pagination,
	keyword string,
) (*types.PageResult, error) {
	if kbID == "" {
		return nil, werrors.NewBadRequestError("지식베이스 ID는 비워둘 수 없습니다")
	}
	if page == nil {
		page = &types.Pagination{}
	}
	keyword = strings.TrimSpace(keyword)
	// Ensure KB exists and belongs to current tenant
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	tenantID := kb.TenantID

	tags, total, err := s.repo.ListByKB(ctx, tenantID, kbID, page, keyword)
	if err != nil {
		return nil, err
	}

	results := make([]*types.KnowledgeTagWithStats, 0, len(tags))
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		kCount, cCount, err := s.repo.CountReferences(ctx, tenantID, kbID, tag.ID)
		if err != nil {
			logger.ErrorWithFields(ctx, err, map[string]interface{}{
				"kb_id":  kbID,
				"tag_id": tag.ID,
			})
			return nil, err
		}
		results = append(results, &types.KnowledgeTagWithStats{
			KnowledgeTag:   *tag,
			KnowledgeCount: kCount,
			ChunkCount:     cCount,
		})
	}
	return types.NewPageResult(total, page, results), nil
}

// CreateTag creates a new tag under a KB.
func (s *knowledgeTagService) CreateTag(
	ctx context.Context,
	kbID string,
	name string,
	color string,
	sortOrder int,
) (*types.KnowledgeTag, error) {
	name = strings.TrimSpace(name)
	if kbID == "" || name == "" {
		return nil, werrors.NewBadRequestError("지식베이스 ID와 태그 이름은 비워둘 수 없습니다")
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tag := &types.KnowledgeTag{
		ID:              uuid.New().String(),
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		Name:            name,
		Color:           strings.TrimSpace(color),
		SortOrder:       sortOrder,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

// UpdateTag updates tag basic information.
func (s *knowledgeTagService) UpdateTag(
	ctx context.Context,
	id string,
	name *string,
	color *string,
	sortOrder *int,
) (*types.KnowledgeTag, error) {
	if id == "" {
		return nil, werrors.NewBadRequestError("태그 ID는 비워둘 수 없습니다")
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	tag, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		newName := strings.TrimSpace(*name)
		if newName == "" {
			return nil, werrors.NewBadRequestError("태그 이름은 비워둘 수 없습니다")
		}
		tag.Name = newName
	}
	if color != nil {
		tag.Color = strings.TrimSpace(*color)
	}
	if sortOrder != nil {
		tag.SortOrder = *sortOrder
	}
	tag.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

// DeleteTag deletes a tag. When force=true, also deletes all chunks under this tag.
func (s *knowledgeTagService) DeleteTag(ctx context.Context, id string, force bool) error {
	if id == "" {
		return werrors.NewBadRequestError("태그 ID는 비워둘 수 없습니다")
	}
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	tag, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	kCount, cCount, err := s.repo.CountReferences(ctx, tenantID, tag.KnowledgeBaseID, tag.ID)
	if err != nil {
		return err
	}
	if !force && (kCount > 0 || cCount > 0) {
		return werrors.NewBadRequestError("태그를 참조하는 지식 또는 FAQ 항목이 있어 삭제할 수 없습니다")
	}
	// When force=true, delete all chunks under this tag first
	if force && cCount > 0 {
		if err := s.chunkRepo.DeleteChunksByTagID(ctx, tenantID, tag.KnowledgeBaseID, tag.ID); err != nil {
			logger.Errorf(ctx, "Failed to delete chunks by tag ID %s: %v", tag.ID, err)
			return werrors.NewInternalServerError("태그 하위 데이터 삭제에 실패했습니다")
		}
		logger.Infof(ctx, "Deleted %d chunks under tag %s", cCount, tag.ID)
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// FindOrCreateTagByName finds a tag by name or creates it if not exists.
func (s *knowledgeTagService) FindOrCreateTagByName(ctx context.Context, kbID string, name string) (*types.KnowledgeTag, error) {
	name = strings.TrimSpace(name)
	if kbID == "" || name == "" {
		return nil, werrors.NewBadRequestError("지식베이스 ID와 태그 이름은 비워둘 수 없습니다")
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}

	tenantID := kb.TenantID

	tag, err := s.repo.GetByName(ctx, tenantID, kbID, name)
	if err == nil {
		return tag, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return s.CreateTag(ctx, kbID, name, "", 0)
}

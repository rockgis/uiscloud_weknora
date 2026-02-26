// Package types defines data structures and types used throughout the system
// These types are shared across different service modules to ensure data consistency
package types

import (
	"time"

	"gorm.io/gorm"
)

type ChunkType string

const (
	ChunkTypeText ChunkType = "text"
	ChunkTypeImageOCR ChunkType = "image_ocr"
	ChunkTypeImageCaption ChunkType = "image_caption"
	ChunkTypeSummary = "summary"
	ChunkTypeEntity ChunkType = "entity"
	ChunkTypeRelationship ChunkType = "relationship"
	ChunkTypeFAQ ChunkType = "faq"
	ChunkTypeWebSearch ChunkType = "web_search"
)

type ChunkStatus int

const (
	ChunkStatusDefault ChunkStatus = 0
	ChunkStatusStored ChunkStatus = 1
	ChunkStatusIndexed ChunkStatus = 2
)

type ChunkFlags int

const (
	ChunkFlagRecommended ChunkFlags = 1 << 0
)

func (f ChunkFlags) HasFlag(flag ChunkFlags) bool {
	return f&flag != 0
}

func (f ChunkFlags) SetFlag(flag ChunkFlags) ChunkFlags {
	return f | flag
}

func (f ChunkFlags) ClearFlag(flag ChunkFlags) ChunkFlags {
	return f &^ flag
}

func (f ChunkFlags) ToggleFlag(flag ChunkFlags) ChunkFlags {
	return f ^ flag
}

type ImageInfo struct {
	URL string `json:"url"          gorm:"type:text"`
	OriginalURL string `json:"original_url" gorm:"type:text"`
	StartPos int `json:"start_pos"`
	EndPos int `json:"end_pos"`
	Caption string `json:"caption"`
	OCRText string `json:"ocr_text"`
}

// Chunk represents a document chunk
// Chunks are meaningful text segments extracted from original documents
// and are the basic units of knowledge base retrieval
// Each chunk contains a portion of the original content
// and maintains its positional relationship with the original text
// Chunks can be independently embedded as vectors and retrieved, supporting precise content localization
type Chunk struct {
	// Unique identifier of the chunk, using UUID format
	ID string `json:"id"                       gorm:"type:varchar(36);primaryKey"`
	// Tenant ID, used for multi-tenant isolation
	TenantID uint64 `json:"tenant_id"`
	// ID of the parent knowledge, associated with the Knowledge model
	KnowledgeID string `json:"knowledge_id"`
	// ID of the knowledge base, for quick location
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// Optional tag ID for categorization within a knowledge base (used for FAQ)
	TagID string `json:"tag_id"                   gorm:"type:varchar(36);index"`
	// Actual text content of the chunk
	Content string `json:"content"`
	// Index position of the chunk in the original document
	ChunkIndex int `json:"chunk_index"`
	// Whether the chunk is enabled, can be used to temporarily disable certain chunks
	IsEnabled bool `json:"is_enabled"               gorm:"default:true"`
	Flags ChunkFlags `json:"flags"                    gorm:"default:1"`
	// Status of the chunk
	Status int `json:"status"                   gorm:"default:0"`
	// Starting character position in the original text
	StartAt int `json:"start_at"`
	// Ending character position in the original text
	EndAt int `json:"end_at"`
	// Previous chunk ID
	PreChunkID string `json:"pre_chunk_id"`
	// Next chunk ID
	NextChunkID string `json:"next_chunk_id"`
	ChunkType ChunkType `json:"chunk_type"               gorm:"type:varchar(20);default:'text'"`
	ParentChunkID string `json:"parent_chunk_id"          gorm:"type:varchar(36);index"`
	RelationChunks JSON `json:"relation_chunks"          gorm:"type:json"`
	IndirectRelationChunks JSON `json:"indirect_relation_chunks" gorm:"type:json"`
	Metadata JSON `json:"metadata"                 gorm:"type:json"`
	ContentHash string `json:"content_hash"             gorm:"type:varchar(64);index"`
	ImageInfo string `json:"image_info"               gorm:"type:text"`
	// Chunk creation time
	CreatedAt time.Time `json:"created_at"`
	// Chunk last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Soft delete marker, supports data recovery
	DeletedAt gorm.DeletedAt `json:"deleted_at"               gorm:"index"`
}

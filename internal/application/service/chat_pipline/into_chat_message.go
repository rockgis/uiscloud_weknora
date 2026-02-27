package chatpipline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/uiscloud_weknora/internal/utils"
)

// PluginIntoChatMessage handles the transformation of search results into chat messages
type PluginIntoChatMessage struct{}

// NewPluginIntoChatMessage creates and registers a new PluginIntoChatMessage instance
func NewPluginIntoChatMessage(eventManager *EventManager) *PluginIntoChatMessage {
	res := &PluginIntoChatMessage{}
	eventManager.Register(res)
	return res
}

// ActivationEvents returns the event types this plugin handles
func (p *PluginIntoChatMessage) ActivationEvents() []types.EventType {
	return []types.EventType{types.INTO_CHAT_MESSAGE}
}

// OnEvent processes the INTO_CHAT_MESSAGE event to format chat message content
func (p *PluginIntoChatMessage) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	pipelineInfo(ctx, "IntoChatMessage", "input", map[string]interface{}{
		"session_id":       chatManage.SessionID,
		"merge_result_cnt": len(chatManage.MergeResult),
		"template_len":     len(chatManage.SummaryConfig.ContextTemplate),
	})

	// Extract content from merge results
	passages := make([]string, len(chatManage.MergeResult))
	for i, result := range chatManage.MergeResult {
		passages[i] = getEnrichedPassageForChat(ctx, result)
	}

	// Parse the context template
	tmpl, err := template.New("searchContent").Parse(chatManage.SummaryConfig.ContextTemplate)
	if err != nil {
		pipelineError(ctx, "IntoChatMessage", "parse_template", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"error":      err.Error(),
		})
		return ErrTemplateParse.WithError(err)
	}

	// Prepare weekday names for template
	weekdayName := []string{"일요일", "월요일", "화요일", "수요일", "목요일", "금요일", "토요일"}
	var userContent bytes.Buffer

	safeQuery, isValid := secutils.ValidateInput(chatManage.Query)
	if !isValid {
		pipelineWarn(ctx, "IntoChatMessage", "invalid_query", map[string]interface{}{
			"session_id": chatManage.SessionID,
		})
		return ErrTemplateExecute.WithError(fmt.Errorf("사용자 쿼리에 허용되지 않는 내용이 포함되어 있습니다"))
	}

	// Execute template with context data
	err = tmpl.Execute(&userContent, map[string]interface{}{
		"Query":       safeQuery,                                // User's original query
		"Contexts":    passages,                                 // Extracted passages from search results
		"CurrentTime": time.Now().Format("2006-01-02 15:04:05"), // Formatted current time
		"CurrentWeek": weekdayName[time.Now().Weekday()],        // Current weekday in Chinese
	})
	if err != nil {
		pipelineError(ctx, "IntoChatMessage", "render_template", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"error":      err.Error(),
		})
		return ErrTemplateExecute.WithError(err)
	}

	// Set formatted content back to chat management
	chatManage.UserContent = userContent.String()
	pipelineInfo(ctx, "IntoChatMessage", "output", map[string]interface{}{
		"session_id":       chatManage.SessionID,
		"user_content_len": len(chatManage.UserContent),
	})
	return next()
}

func getEnrichedPassageForChat(ctx context.Context, result *types.SearchResult) string {
	if result.Content == "" && result.ImageInfo == "" {
		return ""
	}

	if result.ImageInfo == "" {
		return result.Content
	}

	return enrichContentWithImageInfo(ctx, result.Content, result.ImageInfo)
}

var markdownImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

func enrichContentWithImageInfo(ctx context.Context, content string, imageInfoJSON string) string {
	var imageInfos []types.ImageInfo
	err := json.Unmarshal([]byte(imageInfoJSON), &imageInfos)
	if err != nil {
		pipelineWarn(ctx, "IntoChatMessage", "image_parse_error", map[string]interface{}{
			"error": err.Error(),
		})
		return content
	}

	if len(imageInfos) == 0 {
		return content
	}

	imageInfoMap := make(map[string]*types.ImageInfo)
	for i := range imageInfos {
		if imageInfos[i].URL != "" {
			imageInfoMap[imageInfos[i].URL] = &imageInfos[i]
		}
		if imageInfos[i].OriginalURL != "" {
			imageInfoMap[imageInfos[i].OriginalURL] = &imageInfos[i]
		}
	}

	matches := markdownImageRegex.FindAllStringSubmatch(content, -1)

	processedURLs := make(map[string]bool)

	pipelineInfo(ctx, "IntoChatMessage", "image_markdown_links", map[string]interface{}{
		"match_count": len(matches),
	})

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		imgURL := match[2]

		processedURLs[imgURL] = true

		imgInfo, found := imageInfoMap[imgURL]

		if found && imgInfo != nil {
			replacement := match[0] + "\n"
			if imgInfo.Caption != "" {
				replacement += fmt.Sprintf("이미지 설명: %s\n", imgInfo.Caption)
			}
			if imgInfo.OCRText != "" {
				replacement += fmt.Sprintf("이미지 텍스트: %s\n", imgInfo.OCRText)
			}
			content = strings.Replace(content, match[0], replacement, 1)
		}
	}

	var additionalImageTexts []string
	for _, imgInfo := range imageInfos {
		if processedURLs[imgInfo.URL] || processedURLs[imgInfo.OriginalURL] {
			continue
		}

		var imgTexts []string
		if imgInfo.Caption != "" {
			imgTexts = append(imgTexts, fmt.Sprintf("이미지 %s 설명: %s", imgInfo.URL, imgInfo.Caption))
		}
		if imgInfo.OCRText != "" {
			imgTexts = append(imgTexts, fmt.Sprintf("이미지 %s 텍스트: %s", imgInfo.URL, imgInfo.OCRText))
		}

		if len(imgTexts) > 0 {
			additionalImageTexts = append(additionalImageTexts, imgTexts...)
		}
	}

	if len(additionalImageTexts) > 0 {
		if content != "" {
			content += "\n\n"
		}
		content += "추가 이미지 정보:\n" + strings.Join(additionalImageTexts, "\n")
	}

	pipelineInfo(ctx, "IntoChatMessage", "image_enrich_summary", map[string]interface{}{
		"markdown_images": len(matches),
		"additional_imgs": len(additionalImageTexts),
	})

	return content
}

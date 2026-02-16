package openai

import (
	"agent_study/pkg/llm_core/model"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/sashabaranov/go-openai"
)

func buildOpenAIMessages(messages []model.Message) ([]openai.ChatCompletionMessage, []string, error) {
	msgs := make([]openai.ChatCompletionMessage, 0, len(messages))
	promptMessages := make([]string, 0, len(messages))

	for _, m := range messages {
		if len(m.Attachments) == 0 {
			msgs = append(msgs, openai.ChatCompletionMessage{
				Role:    m.Role,
				Content: m.Content,
			})
			promptMessages = append(promptMessages, m.Content)
			continue
		}

		parts := make([]openai.ChatMessagePart, 0, len(m.Attachments)+1)
		promptParts := make([]string, 0, len(m.Attachments)+1)
		if m.Content != "" {
			parts = append(parts, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: m.Content,
			})
			promptParts = append(promptParts, m.Content)
		}

		for _, attachment := range m.Attachments {
			part, promptPart, err := toChatMessagePart(attachment)
			if err != nil {
				return nil, nil, err
			}
			parts = append(parts, part)
			if promptPart != "" {
				promptParts = append(promptParts, promptPart)
			}
		}

		msg := openai.ChatCompletionMessage{Role: m.Role}
		if len(parts) > 0 {
			msg.MultiContent = parts
		} else {
			msg.Content = m.Content
		}
		msgs = append(msgs, msg)
		promptMessages = append(promptMessages, strings.Join(promptParts, "\n"))
	}

	return msgs, promptMessages, nil
}

func toChatMessagePart(attachment model.Attachment) (openai.ChatMessagePart, string, error) {
	mimeType := strings.TrimSpace(attachment.MimeType)
	if mimeType == "" {
		mimeType = http.DetectContentType(attachment.Data)
	}

	if strings.HasPrefix(mimeType, "image/") {
		if len(attachment.Data) == 0 {
			return openai.ChatMessagePart{}, "", fmt.Errorf("image attachment %q data is empty", attachment.FileName)
		}
		encoded := base64.StdEncoding.EncodeToString(attachment.Data)
		dataURL := "data:" + mimeType + ";base64," + encoded
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: dataURL,
			},
		}, "[image attachment]", nil
	}

	if isTextMimeType(mimeType) || utf8.Valid(attachment.Data) {
		fileName := attachment.FileName
		if fileName == "" {
			fileName = "attachment.txt"
		}
		content := string(attachment.Data)
		text := "[附件:" + fileName + "]\n" + content
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: text,
		}, text, nil
	}

	return openai.ChatMessagePart{}, "", fmt.Errorf("unsupported attachment type: %s", mimeType)
}

func isTextMimeType(mimeType string) bool {
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	if mimeType == "application/json" || strings.HasSuffix(mimeType, "+json") {
		return true
	}
	if mimeType == "application/xml" || strings.HasSuffix(mimeType, "+xml") {
		return true
	}
	return false
}

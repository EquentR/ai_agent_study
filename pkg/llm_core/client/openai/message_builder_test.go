package openai

import (
	"agent_study/pkg/llm_core/model"
	"strings"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
)

func TestBuildOpenAIMessages_TextOnly(t *testing.T) {
	msgs, promptMessages, err := buildOpenAIMessages([]model.Message{{
		Role:    model.MessageUser,
		Content: "hello",
	}})
	if err != nil {
		t.Fatalf("buildOpenAIMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("msgs[0].Content = %q, want %q", msgs[0].Content, "hello")
	}
	if msgs[0].MultiContent != nil {
		t.Fatalf("msgs[0].MultiContent should be nil")
	}
	if len(promptMessages) != 1 || promptMessages[0] != "hello" {
		t.Fatalf("promptMessages = %#v, want [hello]", promptMessages)
	}
}

func TestBuildOpenAIMessages_WithAttachments(t *testing.T) {
	msgs, promptMessages, err := buildOpenAIMessages([]model.Message{{
		Role:    model.MessageUser,
		Content: "请分析附件",
		Attachments: []model.Attachment{
			{
				FileName: "img.png",
				MimeType: "image/png",
				Data:     []byte{1, 2, 3},
			},
			{
				FileName: "note.txt",
				MimeType: "text/plain",
				Data:     []byte("line1"),
			},
		},
	}})
	if err != nil {
		t.Fatalf("buildOpenAIMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if len(msgs[0].MultiContent) != 3 {
		t.Fatalf("len(msgs[0].MultiContent) = %d, want 3", len(msgs[0].MultiContent))
	}
	if msgs[0].MultiContent[0].Type != goopenai.ChatMessagePartTypeText {
		t.Fatalf("first part type = %q, want %q", msgs[0].MultiContent[0].Type, goopenai.ChatMessagePartTypeText)
	}
	if msgs[0].MultiContent[1].Type != goopenai.ChatMessagePartTypeImageURL {
		t.Fatalf("second part type = %q, want %q", msgs[0].MultiContent[1].Type, goopenai.ChatMessagePartTypeImageURL)
	}
	if !strings.HasPrefix(msgs[0].MultiContent[1].ImageURL.URL, "data:image/png;base64,") {
		t.Fatalf("image url = %q, want data url", msgs[0].MultiContent[1].ImageURL.URL)
	}
	if msgs[0].MultiContent[2].Type != goopenai.ChatMessagePartTypeText {
		t.Fatalf("third part type = %q, want %q", msgs[0].MultiContent[2].Type, goopenai.ChatMessagePartTypeText)
	}
	if !strings.Contains(msgs[0].MultiContent[2].Text, "[附件:note.txt]") {
		t.Fatalf("third part text = %q, want file marker", msgs[0].MultiContent[2].Text)
	}
	if len(promptMessages) != 1 || !strings.Contains(promptMessages[0], "请分析附件") || !strings.Contains(promptMessages[0], "[附件:note.txt]") {
		t.Fatalf("promptMessages = %#v, want combined prompt text", promptMessages)
	}
}

func TestBuildOpenAIMessages_UnsupportedAttachment(t *testing.T) {
	_, _, err := buildOpenAIMessages([]model.Message{{
		Role: model.MessageUser,
		Attachments: []model.Attachment{{
			FileName: "doc.pdf",
			MimeType: "application/pdf",
			Data:     []byte{0xff, 0x00, 0x10},
		}},
	}})
	if err == nil {
		t.Fatal("expected error for unsupported attachment type")
	}
}

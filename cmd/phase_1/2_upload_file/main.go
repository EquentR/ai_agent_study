package main

import (
	"agent_study/pkg/llm_core/client/openai"
	"agent_study/pkg/llm_core/model"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	question := flag.String("question", "请阅读附件并给出总结", "用户问题")
	imagePath := flag.String("image", "", "图片文件路径")
	textPath := flag.String("text", "", "文本文件路径")
	modelName := flag.String("model", "kimi-k2.5", "模型名称")
	flag.Parse()

	if *imagePath == "" && *textPath == "" {
		panic("请至少提供一个附件：--image 或 --text")
	}

	attachments := make([]model.Attachment, 0, 2)
	if *imagePath != "" {
		att, err := loadAttachment(*imagePath)
		if err != nil {
			panic(err)
		}
		attachments = append(attachments, att)
	}
	if *textPath != "" {
		att, err := loadAttachment(*textPath)
		if err != nil {
			panic(err)
		}
		attachments = append(attachments, att)
	}

	llmClient := openai.NewOpenAiClient(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"))
	resp, err := llmClient.Chat(context.Background(), model.ChatRequest{
		Model: *modelName,
		Messages: []model.Message{
			{
				Role:        model.MessageUser,
				Content:     *question,
				Attachments: attachments,
			},
		},
		MaxTokens: 1024,
		TraceID:   "phase1-upload-file-example",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Response Content:", resp.Content)
}

func loadAttachment(path string) (model.Attachment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Attachment{}, err
	}
	mimeType := http.DetectContentType(data)
	ext := strings.ToLower(filepath.Ext(path))
	if strings.HasPrefix(ext, ".") {
		ext = ext[1:]
	}
	if strings.HasPrefix(mimeType, "application/octet-stream") {
		switch ext {
		case "txt", "md", "csv", "log", "json", "yaml", "yml", "xml":
			mimeType = "text/plain"
		case "png":
			mimeType = "image/png"
		case "jpg", "jpeg":
			mimeType = "image/jpeg"
		case "webp":
			mimeType = "image/webp"
		}
	}

	return model.Attachment{
		FileName: filepath.Base(path),
		MimeType: mimeType,
		Data:     data,
	}, nil
}

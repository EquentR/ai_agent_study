package model

import (
	"context"
	"time"
)

type Stream interface {
	// Recv 返回下一段内容
	// content == "" 且 err == nil 表示暂时无数据
	Recv() (content string, err error)

	// Close 主动中断（如用户关闭页面）
	Close() error

	// Context 关联生命周期
	Context() context.Context

	// Stats 统计数据
	Stats() *StreamStats
}

type StreamStats struct {
	Usage           TokenUsage
	TTFT            time.Duration
	TotalLatency    time.Duration
	LocalTokenCount int64 // 本地计数的completion tokens
}

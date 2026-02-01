package openai

import (
	"agent_study/pkg/llm_core/model"
	"agent_study/pkg/llm_core/tools"
	"context"
	"sync"
	"time"
)

type openAIStream struct {
	ctx               context.Context
	cancel            context.CancelFunc
	ch                <-chan string
	stats             *model.StreamStats
	startTime         time.Time
	firstTok          sync.Once
	asyncTokenCounter *tools.AsyncTokenCounter // 异步token计数器
}

func (s *openAIStream) Recv() (string, error) {
	select {
	case <-s.ctx.Done():
		return "", s.ctx.Err()
	case msg, ok := <-s.ch:
		if !ok {
			return "", nil
		}
		return msg, nil
	}
}

func (s *openAIStream) Close() error {
	s.cancel()
	if s.asyncTokenCounter != nil {
		s.asyncTokenCounter.Close()
	}
	return nil
}

func (s *openAIStream) Context() context.Context {
	return s.ctx
}

func (s *openAIStream) Stats() *model.StreamStats {
	return s.stats
}

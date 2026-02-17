package google

import (
	"agent_study/pkg/llm_core/model"
	"context"
	"errors"
	"testing"
)

func TestGenAIStreamRecv_ReturnsStreamErrorWhenChannelClosed(t *testing.T) {
	ch := make(chan string)
	close(ch)

	s := &genAIStream{
		ctx:   context.Background(),
		ch:    ch,
		stats: &model.StreamStats{},
	}
	streamErr := errors.New("stream failed")
	s.setStreamError(streamErr)

	_, err := s.Recv()
	if !errors.Is(err, streamErr) {
		t.Fatalf("Recv() error = %v, want %v", err, streamErr)
	}
}

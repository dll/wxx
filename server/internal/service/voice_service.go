package service

import (
	"context"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

const (
	voiceASRTimeoutService = 18 * time.Second
	voiceTTSTimeoutService = 15 * time.Second
)

type VoiceService struct {
	xfClient *llm.XfyunClient
}

func NewVoiceService(xfClient *llm.XfyunClient) *VoiceService {
	return &VoiceService{xfClient: xfClient}
}

func (s *VoiceService) ASR(ctx context.Context, audioBytes []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, voiceASRTimeoutService)
	defer cancel()
	return s.xfClient.ASR(ctx, audioBytes)
}

func (s *VoiceService) TTS(ctx context.Context, text string, voiceName string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, voiceTTSTimeoutService)
	defer cancel()
	return s.xfClient.TTS(ctx, text, voiceName)
}

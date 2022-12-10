package chatgpt

import (
	"context"
	"errors"
	"notionboy/internal/pkg/logger"
	"strings"
	"sync/atomic"

	gogpt "github.com/sashabaranov/go-gpt3"
)

type apiClient struct {
	*gogpt.Client
	isRateLimit atomic.Bool
}

func newApiClient(apiKey string) Chatter {
	client := &apiClient{
		Client: gogpt.NewClient(apiKey),
	}
	client.setIsRateLimit(false)
	return client
}

func (cli *apiClient) Chat(ctx context.Context, parentMessageId, prompt string) (string, string, error) {
	if cli.GetIsRateLimit() {
		return "", "", errors.New("hit rate limit, please increase your quote")
	}
	req := gogpt.CompletionRequest{
		Model:     gogpt.GPT3TextDavinci003,
		MaxTokens: 1024,
		Prompt:    prompt,
	}

	respChan := make(chan *gogpt.CompletionResponse)
	errChan := make(chan error)

	chat := func() {
		resp, err := cli.CreateCompletion(ctx, req)
		if err != nil {
			errChan <- err
		} else {
			respChan <- &resp
		}
	}
	var err error
	for i := 0; i < 3; i++ {
		go chat()
		select {
		case resp := <-respChan:
			msgId := resp.ID
			sb := strings.Builder{}
			for _, item := range resp.Choices {
				sb.WriteString(item.Text)
				sb.WriteString("\n")
			}
			logger.SugaredLogger.Debugw("Response", "conversation_id", msgId, "error", nil, "message", sb.String())
			return msgId, sb.String(), nil
		case err = <-errChan:
			logger.SugaredLogger.Warnw("Get response from chatGPT error", "retry_times", i+1, "err", err)
		}
	}

	return "", "", err
}

func (cli *apiClient) GetIsRateLimit() bool {
	return cli.isRateLimit.Load()
}

func (cli *apiClient) setIsRateLimit(flag bool) {
	cli.isRateLimit.Store(flag)
}
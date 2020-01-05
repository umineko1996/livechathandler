package livechathandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	youtube "google.golang.org/api/youtube/v3"

	"github.com/umineko1996/livechathandler/internal/oauth2"
)

type LiveChatHandler struct {
	interval   int
	videoID    string
	liveChatID string
	client     *youtube.Service
	MessageHandler
}

type MessageHandler interface {
	MessageHandle(message *youtube.LiveChatMessage) error
}

type MessageHandlerFunc func(message *youtube.LiveChatMessage) error

func (mh MessageHandlerFunc) MessageHandle(message *youtube.LiveChatMessage) error {
	return mh(message)
}

type Option interface {
	Apply(handler *LiveChatHandler)
}

type OptionFunc func(handler *LiveChatHandler)

func (of OptionFunc) Apply(handler *LiveChatHandler) {
	of(handler)
}

func WithInterval(interval int) Option {
	return OptionFunc(func(handler *LiveChatHandler) {
		handler.interval = interval
	})
}

func New(videoID string, options ...Option) (*LiveChatHandler, error) {
	newService := func() (*youtube.Service, error) {
		client, err := oauth2.NewClient()
		if err != nil {
			return nil, err
		}
		service, err := youtube.New(client)
		if err != nil {
			return nil, fmt.Errorf("new client: %w", err)
		}
		return service, nil
	}

	getLiveID := func(service *youtube.Service, videoID string) (liveChatID string, err error) {
		call := service.Videos.List("liveStreamingDetails").Id(videoID)
		resp, err := call.Do()
		if err != nil {
			return "", fmt.Errorf("get broadcast: %w", err)
		} else if len(resp.Items) == 0 {
			return "", errors.New("get broadcast: Not Found")
		}
		return resp.Items[0].LiveStreamingDetails.ActiveLiveChatId, nil
	}

	service, err := newService()
	if err != nil {
		return nil, err
	}
	liveChatID, err := getLiveID(service, videoID)
	if err != nil {
		return nil, err
	}

	handler := &LiveChatHandler{
		interval:   5,
		videoID:    videoID,
		liveChatID: liveChatID,
		client:     service,
	}

	for _, opt := range options {
		opt.Apply(handler)
	}

	return handler, nil
}

func (lh *LiveChatHandler) Polling(ctx context.Context, handler MessageHandler) error {
	lh.MessageHandler = handler

	// 初回読み込み
	resp, err := lh.client.LiveChatMessages.List(lh.liveChatID, "id").Do()
	if err != nil {
		return fmt.Errorf("get livechat: %w", err)
	}

	call := lh.client.LiveChatMessages.List(lh.liveChatID, "snippet, AuthorDetails")
	next := resp.NextPageToken
	defaultInterval := int64(lh.interval * 1000)
	timer := time.NewTimer(0)
	defer timer.Stop()

	// コンテキストにより中断するまでポーリング
	for ctx.Err() == nil {
		// ポーリング間隔調整
		intervalMs := defaultInterval
		if resp.PollingIntervalMillis > intervalMs {
			intervalMs = resp.PollingIntervalMillis
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(time.Duration(intervalMs) * time.Millisecond)

		select {
		case <-timer.C:
		}

		// コメント取得
		resp, err := call.PageToken(next).MaxResults(2000).Do()
		if err != nil {
			return fmt.Errorf("get livechat: %w", err)
		}

		for _, item := range resp.Items {
			handler.MessageHandle(item)
		}

		next = resp.NextPageToken
	}

	return nil
}

package livechathandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	youtube "google.golang.org/api/youtube/v3"

	"github.com/umineko1996/livechathandler/internal/oauth2"
)

type (
	MessageHandler interface {
		MessageHandle(message *youtube.LiveChatMessage)
	}
	MessageHandlerFunc func(message *youtube.LiveChatMessage)

	SimpleMessageHandler interface {
		SimpleMessageHandl(messageText string)
	}

	SimpleMessageHandlerFunc func(messageText string)
	SuperChatHandler         interface {
		SuperChatHandle(tier int64, userName, messageText string)
	}

	SuperChatHandlerFunc func(tier int64, userName, messageText string)

	IntervalHandler interface {
		IntervalHandle(pollingIntervalMillis int64)
	}
	IntervalHandlerFunc func(pollingIntervalMillis int64)

	Option interface {
		Apply(handler *LiveChatHandler)
	}
	OptionFunc func(handler *LiveChatHandler)
)

func (mh MessageHandlerFunc) MessageHandle(message *youtube.LiveChatMessage) {
	mh(message)
}

func (sh SimpleMessageHandlerFunc) SimpleMessageHandl(messageText string) {
	sh(messageText)
}

func (ch SuperChatHandlerFunc) SuperChatHandle(tier int64, userName, messageText string) {
	ch(tier, userName, messageText)
}

func (ih IntervalHandlerFunc) IntervalHandle(pollingIntervalMillis int64) {
	ih(pollingIntervalMillis)
}

func (of OptionFunc) Apply(handler *LiveChatHandler) {
	of(handler)
}

func WithInterval(interval int) Option {
	return OptionFunc(func(handler *LiveChatHandler) {
		handler.interval = interval
	})
}

func WithIntervalHandler(intervalHandler IntervalHandler) Option {
	return OptionFunc(func(handler *LiveChatHandler) {
		handler.IntervalHandler = intervalHandler
	})
}

func NewMessageHandler(simpleHnalder SimpleMessageHandler, superChatHandler SuperChatHandler) MessageHandler {
	if simpleHnalder == nil {
		simpleHnalder = SimpleMessageHandlerFunc(func(messageText string) { return })
	}
	if superChatHandler == nil {
		superChatHandler = SuperChatHandlerFunc(func(tier int64, userName, messageText string) {
			simpleHnalder.SimpleMessageHandl(messageText)
		})
	}
	return MessageHandlerFunc(func(message *youtube.LiveChatMessage) {
		if superChat := message.Snippet.SuperChatDetails; superChat != nil {
			superChatHandler.SuperChatHandle(superChat.Tier, message.AuthorDetails.DisplayName, superChat.UserComment)
			return
		}
		simpleHnalder.SimpleMessageHandl(message.Snippet.DisplayMessage)
	})
}

type LiveChatHandler struct {
	interval   int
	videoID    string
	liveChatID string
	client     *youtube.Service
	MessageHandler
	IntervalHandler
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
		interval:        5,
		videoID:         videoID,
		liveChatID:      liveChatID,
		client:          service,
		IntervalHandler: IntervalHandlerFunc(func(interval int64) { return }),
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

	defaultIntervalMillis := int64(lh.interval * 1000)
	defaultIntervalTimer := time.NewTicker(time.Duration(defaultIntervalMillis) * time.Millisecond)
	defer defaultIntervalTimer.Stop()
	respIntervalTimer := time.NewTimer(0)
	defer respIntervalTimer.Stop()

	waitPollingInterval := func(ctx context.Context, pollingIntervalMillis int64) {
		// ポーリング間隔調整
		if !respIntervalTimer.Stop() {
			select {
			case <-respIntervalTimer.C:
			default:
			}
		}
		respIntervalTimer.Reset(time.Duration(pollingIntervalMillis) * time.Millisecond)

		// default do notihng
		// MEMO( ): defaultIntervalTimerはTickerなので厳密にはこの段階から指定された秒数分待つわけではない
		lh.IntervalHandle(max(pollingIntervalMillis, defaultIntervalMillis))

		// 指定インターバルかレスポンスで指定されたインターバルの長い方でブロックする
		// コンテキストによる中断の場合、指定インターバルの待機は中断するが、レスポンスインターバルの待機は中断されない
		// これは、レスポンスのインターバルを守らないと最後のリクエストがエラーになってしまうためである
		select {
		case <-ctx.Done():
		case <-defaultIntervalTimer.C:
		}

		select {
		case <-respIntervalTimer.C:
		}
	}

	// コンテキストにより中断するまでポーリング
	for ctx.Err() == nil {
		waitPollingInterval(ctx, resp.PollingIntervalMillis)

		// コメント取得
		resp, err := call.PageToken(next).MaxResults(2000).Do()
		if err != nil {
			return fmt.Errorf("get livechat: %w", err)
		}

		for _, item := range resp.Items {
			lh.MessageHandle(item)
		}

		next = resp.NextPageToken
	}

	return nil
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

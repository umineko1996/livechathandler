package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/umineko1996/livechathandler"
)

func main() {
	// os.Setenv("GOOGLE_API_CLIENTID", "XXXXXXXXX.apps.googleusercontent.com")
	// os.Setenv("GOOGLE_API_CLIENTSERCRET", "XXXXXXXXX")

	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return errors.New("usage: example.exe videoid")
	}
	videoid := os.Args[1]

	intervalHandler := livechathandler.IntervalHandlerFunc(func(pollingIntervalMillis int64) {
		fmt.Println("interval: ", pollingIntervalMillis, "ms")
	})
	options := []livechathandler.Option{livechathandler.WithInterval(8), livechathandler.WithIntervalHandler(intervalHandler)}

	handler, err := livechathandler.New(videoid, options...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// polling end  with press enter
	go func() {
		b := make([]byte, 1)
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			time.Sleep(100 * time.Millisecond)
			os.Stdin.Read(b)
			if len(b) != 0 {
				cancel()
			}
		}
	}()

	handler.Polling(ctx, livechathandler.NewMessageHandler(livechathandler.SimpleMessageHandlerFunc(
		func(messageText string) {
			fmt.Println(messageText)
		}), livechathandler.MemberMessageHandlerFunc(
		func(userName, messageText string) {
			fmt.Printf("member: %s, message: %s\n", userName, messageText)
		}), livechathandler.SuperChatHandlerFunc(
		func(tier livechathandler.SuperChatTier, userName, messageText string) {
			fmt.Println("superchat-----------------------------------------------------------------------")
			fmt.Printf("tier: %d, color: %s, user: %s, message: %s\n", tier, tier.Color(), userName, messageText)
			fmt.Println("--------------------------------------------------------------------------------")
		})))

	return nil
}

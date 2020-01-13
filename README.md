# livechathandler

livechathandler makes using youtubeLiveChat easy.

# example

```golang
	handler, err := livechathandler.New("video-id")
	if err != nil {
		return err
	}

	handler.Polling(context.Background(), livechathandler.MessageHandlerFunc(
		func(message *youtube.LiveChatMessage) {
			// do something
			fmt.Println(message.Snippet.DisplayMessage)
		}))
```

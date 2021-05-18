package bot

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/reaitten/flowerss-bot/bot/fsm"
	"github.com/reaitten/flowerss-bot/config"
	"github.com/reaitten/flowerss-bot/model"

	tb "gopkg.in/tucnak/telebot.v2"
)

var (
	feedSettingTmpl = `
Subscription<b>Setup</b>
[id] {{ .sub.ID }}
[title] {{ .source.Title }}
[Link] {{.source.Link }}
[Fetch updates] {{if ge .source.ErrorCount .Count }}time out{{else if lt .source.ErrorCount .Count }}Crawling{{end}}
[Crawl frequency] {{ .sub.Interval }}minute
[Notification] {{if eq .sub.EnableNotification 0}}关闭{{else if eq .sub.EnableNotification 1}}Turn on{{end}}
[Telegraph] {{if eq .sub.EnableTelegraph 0}}shut down{{else if eq .sub.EnableTelegraph 1}}Turn on{{end}}
[Tag] {{if .sub.Tag}}{{ .sub.Tag }}{{else}}no{{end}}
`
)

func toggleCtrlButtons(c *tb.Callback, action string) {

	if (c.Message.Chat.Type == tb.ChatGroup || c.Message.Chat.Type == tb.ChatSuperGroup) &&
		!userIsAdminOfGroup(c.Sender.ID, c.Message.Chat) {
		// check admin
		return
	}

	data := strings.Split(c.Data, ":")
	subscriberID, _ := strconv.Atoi(data[0])
	// If the subscriber id is different from that of the button clicker, you need to verify the administrator authority
	if subscriberID != c.Sender.ID {
		channelChat, err := B.ChatByID(fmt.Sprintf("%d", subscriberID))

		if err != nil {
			return
		}

		if !UserIsAdminChannel(c.Sender.ID, channelChat) {
			return
		}
	}

	msg := strings.Split(c.Message.Text, "\n")
	subID, err := strconv.Atoi(strings.Split(msg[1], " ")[1])
	if err != nil {
		_ = B.Respond(c, &tb.CallbackResponse{
			Text: "error",
		})
		return
	}
	sub, err := model.GetSubscribeByID(subID)
	if sub == nil || err != nil {
		_ = B.Respond(c, &tb.CallbackResponse{
			Text: "error",
		})
		return
	}

	source, _ := model.GetSourceById(sub.SourceID)
	t := template.New("setting template")
	_, _ = t.Parse(feedSettingTmpl)

	switch action {
	case "toggleNotice":
		err = sub.ToggleNotification()
	case "toggleTelegraph":
		err = sub.ToggleTelegraph()
	case "toggleUpdate":
		err = source.ToggleEnabled()
	}

	if err != nil {
		_ = B.Respond(c, &tb.CallbackResponse{
			Text: "error",
		})
		return
	}

	sub.Save()

	text := new(bytes.Buffer)

	_ = t.Execute(text, map[string]interface{}{"source": source, "sub": sub, "Count": config.ErrorThreshold})
	_ = B.Respond(c, &tb.CallbackResponse{
		Text: "Successfully modified",
	})
	_, _ = B.Edit(c.Message, text.String(), &tb.SendOptions{
		ParseMode: tb.ModeHTML,
	}, &tb.ReplyMarkup{
		InlineKeyboard: genFeedSetBtn(c, sub, source),
	})
}

func startCmdCtr(m *tb.Message) {
	user, _ := model.FindOrCreateUserByTelegramID(m.Chat.ID)
	zap.S().Infof("/start user_id: %d telegram_id: %d", user.ID, user.TelegramID)
	_, _ = B.Send(m.Chat, fmt.Sprintf("Hello, welcome to flowerss."))
}

func subCmdCtr(m *tb.Message) {

	url, mention := GetURLAndMentionFromMessage(m)

	if mention == "" {
		if url != "" {
			registFeed(m.Chat, url)
		} else {
			_, err := B.Send(m.Chat, "Please reply to RSS URL", &tb.ReplyMarkup{ForceReply: true})
			if err == nil {
				UserState[m.Chat.ID] = fsm.Sub
			}
		}
	} else {
		if url != "" {
			FeedForChannelRegister(m, url, mention)
		} else {
			_, _ = B.Send(m.Chat, "Please use channel subscription' /sub @ChannelID URL ' command")
		}
	}

}

func exportCmdCtr(m *tb.Message) {

	mention := GetMentionFromMessage(m)
	var sourceList []model.Source
	var err error
	if mention == "" {

		sourceList, err = model.GetSourcesByUserID(m.Chat.ID)
		if err != nil {
			zap.S().Warnf(err.Error())
			_, _ = B.Send(m.Chat, fmt.Sprintf("Export failed"))
			return
		}
	} else {
		channelChat, err := B.ChatByID(mention)

		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}

		adminList, err := B.AdminsOf(channelChat)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}

		senderIsAdmin := false
		for _, admin := range adminList {
			if m.Sender.ID == admin.User.ID {
				senderIsAdmin = true
			}
		}

		if !senderIsAdmin {
			_, _ = B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
			return
		}

		sourceList, err = model.GetSourcesByUserID(channelChat.ID)
		if err != nil {
			zap.S().Errorf(err.Error())
			_, _ = B.Send(m.Chat, fmt.Sprintf("Export failed"))
			return
		}
	}

	if len(sourceList) == 0 {
		_, _ = B.Send(m.Chat, fmt.Sprintf("The subscription list is empty"))
		return
	}

	opmlStr, err := ToOPML(sourceList)

	if err != nil {
		_, _ = B.Send(m.Chat, fmt.Sprintf("Export failed"))
		return
	}
	opmlFile := &tb.Document{File: tb.FromReader(strings.NewReader(opmlStr))}
	opmlFile.FileName = fmt.Sprintf("subscriptions_%d.opml", time.Now().Unix())
	_, err = B.Send(m.Chat, opmlFile)

	if err != nil {
		_, _ = B.Send(m.Chat, fmt.Sprintf("Export failed"))
		zap.S().Errorf("send opml file failed, err:%+v", err)
	}

}

func listCmdCtr(m *tb.Message) {
	mention := GetMentionFromMessage(m)

	var rspMessage string
	if mention != "" {
		// channel feed list
		channelChat, err := B.ChatByID(mention)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}

		if !checkPermitOfChat(int64(m.Sender.ID), channelChat) {
			B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
			return
		}

		user, err := model.FindOrCreateUserByTelegramID(channelChat.ID)
		if err != nil {
			B.Send(m.Chat, fmt.Sprintf("Internal error list@1"))
			return
		}

		subSourceMap, err := user.GetSubSourceMap()
		if err != nil {
			B.Send(m.Chat, fmt.Sprintf("Internal error list@2"))
			return
		}

		sources, _ := model.GetSourcesByUserID(channelChat.ID)
		rspMessage = fmt.Sprintf("Channel [%s](https://t.me/%s) Subscription list：\n", channelChat.Title, channelChat.Username)
		if len(sources) == 0 {
			rspMessage = fmt.Sprintf("Channel [%s](https://t.me/%s) subscription list is empty", channelChat.Title, channelChat.Username)
		} else {
			for sub, source := range subSourceMap {
				rspMessage = rspMessage + fmt.Sprintf("[[%d]] [%s](%s)\n", sub.ID, source.Title, source.Link)
			}
		}
	} else {
		// private chat or group
		if m.Chat.Type != tb.ChatPrivate && !checkPermitOfChat(int64(m.Sender.ID), m.Chat) {
			// Channel
			return
		}

		user, err := model.FindOrCreateUserByTelegramID(m.Chat.ID)
		if err != nil {
			B.Send(m.Chat, fmt.Sprintf("Internal error list@1"))
			return
		}

		subSourceMap, err := user.GetSubSourceMap()
		if err != nil {
			B.Send(m.Chat, fmt.Sprintf("Internal error list@2"))
			return
		}

		rspMessage = "Current subscription list：\n"
		if len(subSourceMap) == 0 {
			rspMessage = "The subscription list is empty"
		} else {
			for sub, source := range subSourceMap {
				rspMessage = rspMessage + fmt.Sprintf("[[%d]] [%s](%s)\n", sub.ID, source.Title, source.Link)
			}
		}
	}
	_, _ = B.Send(m.Chat, rspMessage, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeMarkdown,
	})
}

func checkCmdCtr(m *tb.Message) {
	mention := GetMentionFromMessage(m)
	if mention != "" {
		channelChat, err := B.ChatByID(mention)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}
		adminList, err := B.AdminsOf(channelChat)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}

		senderIsAdmin := false
		for _, admin := range adminList {
			if m.Sender.ID == admin.User.ID {
				senderIsAdmin = true
			}
		}

		if !senderIsAdmin {
			_, _ = B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
			return
		}

		sources, _ := model.GetErrorSourcesByUserID(channelChat.ID)
		message := fmt.Sprintf("Channel [%s](https://t.me/%s) List of expired subscriptions：\n", channelChat.Title, channelChat.Username)
		if len(sources) == 0 {
			message = fmt.Sprintf("Channel [%s](https://t.me/%s) All subscriptions are normal", channelChat.Title, channelChat.Username)
		} else {
			for _, source := range sources {
				message = message + fmt.Sprintf("[[%d]] [%s](%s)\n", source.ID, source.Title, source.Link)
			}
		}

		_, _ = B.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
		})

	} else {
		sources, _ := model.GetErrorSourcesByUserID(m.Chat.ID)
		message := "List of expired subscriptions：\n"
		if len(sources) == 0 {
			message = "All subscriptions are normal"
		} else {
			for _, source := range sources {
				message = message + fmt.Sprintf("[[%d]] [%s](%s)\n", source.ID, source.Title, source.Link)
			}
		}
		_, _ = B.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
		})
	}

}

func setCmdCtr(m *tb.Message) {

	mention := GetMentionFromMessage(m)
	var sources []model.Source
	var ownerID int64
	// 获取订阅列表
	if mention == "" {
		sources, _ = model.GetSourcesByUserID(m.Chat.ID)
		ownerID = int64(m.Chat.ID)
		if len(sources) <= 0 {
			_, _ = B.Send(m.Chat, "There are currently no feeds")
			return
		}

	} else {

		channelChat, err := B.ChatByID(mention)

		if err != nil {
			_, _ = B.Send(m.Chat, "Error obtaining Channel information.")
			return
		}

		if UserIsAdminChannel(m.Sender.ID, channelChat) {
			sources, _ = model.GetSourcesByUserID(channelChat.ID)

			if len(sources) <= 0 {
				_, _ = B.Send(m.Chat, "Channel has no feed.")
				return
			}
			ownerID = channelChat.ID

		} else {
			_, _ = B.Send(m.Chat, "Non-Channel administrators cannot perform this operation.")
			return
		}

	}

	var replyButton []tb.ReplyButton
	replyKeys := [][]tb.ReplyButton{}
	setFeedItemBtns := [][]tb.InlineButton{}

	// Configure button
	for _, source := range sources {
		// Add button
		text := fmt.Sprintf("%s %s", source.Title, source.Link)
		replyButton = []tb.ReplyButton{
			tb.ReplyButton{Text: text},
		}
		replyKeys = append(replyKeys, replyButton)

		setFeedItemBtns = append(setFeedItemBtns, []tb.InlineButton{
			tb.InlineButton{
				Unique: "set_feed_item_btn",
				Text:   fmt.Sprintf("[%d] %s", source.ID, source.Title),
				Data:   fmt.Sprintf("%d:%d", ownerID, source.ID),
			},
		})
	}

	_, _ = B.Send(m.Chat, "Please select the source you want to set", &tb.ReplyMarkup{
		InlineKeyboard: setFeedItemBtns,
	})
}

func setFeedItemBtnCtr(c *tb.Callback) {

	if (c.Message.Chat.Type == tb.ChatGroup || c.Message.Chat.Type == tb.ChatSuperGroup) &&
		!userIsAdminOfGroup(c.Sender.ID, c.Message.Chat) {
		return
	}

	data := strings.Split(c.Data, ":")
	subscriberID, _ := strconv.Atoi(data[0])

	// If the subscriber id is different from the button clicker id, you need to verify the administrator authority

	if subscriberID != c.Sender.ID {
		channelChat, err := B.ChatByID(fmt.Sprintf("%d", subscriberID))

		if err != nil {
			return
		}

		if !UserIsAdminChannel(c.Sender.ID, channelChat) {
			return
		}
	}

	sourceID, _ := strconv.Atoi(data[1])

	source, err := model.GetSourceById(uint(sourceID))

	if err != nil {
		_, _ = B.Edit(c.Message, "The feed could not be found, error code 01.")
		return
	}

	sub, err := model.GetSubscribeByUserIDAndSourceID(int64(subscriberID), source.ID)
	if err != nil {
		_, _ = B.Edit(c.Message, "The user has not subscribed to the rss, the error code is 02.")
		return
	}

	t := template.New("setting template")
	_, _ = t.Parse(feedSettingTmpl)
	text := new(bytes.Buffer)
	_ = t.Execute(text, map[string]interface{}{"source": source, "sub": sub, "Count": config.ErrorThreshold})

	_, _ = B.Edit(
		c.Message,
		text.String(),
		&tb.SendOptions{
			ParseMode: tb.ModeHTML,
		}, &tb.ReplyMarkup{
			InlineKeyboard: genFeedSetBtn(c, sub, source),
		},
	)
}

func setSubTagBtnCtr(c *tb.Callback) {

	// Permission Validation
	if !feedSetAuth(c) {
		return
	}
	data := strings.Split(c.Data, ":")
	ownID, _ := strconv.Atoi(data[0])
	sourceID, _ := strconv.Atoi(data[1])

	sub, err := model.GetSubscribeByUserIDAndSourceID(int64(ownID), uint(sourceID))
	if err != nil {
		_, _ = B.Send(
			c.Message.Chat,
			"System error, code 04",
		)
		return
	}
	msg := fmt.Sprintf(
		"Please use the `/setfeedtag %d tags` command to set tags for this subscription. Tags are the tags to be set, separated by spaces. (Up to three labels can be set) \n"+
			"For example: `/setfeedtag %d technology apple`",
		sub.ID, sub.ID)

	_ = B.Delete(c.Message)

	_, _ = B.Send(
		c.Message.Chat,
		msg,
		&tb.SendOptions{ParseMode: tb.ModeMarkdown},
	)
}

func genFeedSetBtn(c *tb.Callback, sub *model.Subscribe, source *model.Source) [][]tb.InlineButton {
	setSubTagKey := tb.InlineButton{
		Unique: "set_set_sub_tag_btn",
		Text:   "Label settings",
		Data:   c.Data,
	}

	toggleNoticeKey := tb.InlineButton{
		Unique: "set_toggle_notice_btn",
		Text:   "Turn on notifications",
		Data:   c.Data,
	}
	if sub.EnableNotification == 1 {
		toggleNoticeKey.Text = "Close notification"
	}

	toggleTelegraphKey := tb.InlineButton{
		Unique: "set_toggle_telegraph_btn",
		Text:   "Turn on Telegraph transcoding",
		Data:   c.Data,
	}
	if sub.EnableTelegraph == 1 {
		toggleTelegraphKey.Text = "Turn off Telegraph transcoding"
	}

	toggleEnabledKey := tb.InlineButton{
		Unique: "set_toggle_update_btn",
		Text:   "Pause update",
		Data:   c.Data,
	}

	if source.ErrorCount >= config.ErrorThreshold {
		toggleEnabledKey.Text = "Restart update"
	}

	feedSettingKeys := [][]tb.InlineButton{
		[]tb.InlineButton{
			toggleEnabledKey,
			toggleNoticeKey,
		},
		[]tb.InlineButton{
			toggleTelegraphKey,
			setSubTagKey,
		},
	}
	return feedSettingKeys
}

func setToggleNoticeBtnCtr(c *tb.Callback) {
	toggleCtrlButtons(c, "toggleNotice")
}

func setToggleTelegraphBtnCtr(c *tb.Callback) {
	toggleCtrlButtons(c, "toggleTelegraph")
}

func setToggleUpdateBtnCtr(c *tb.Callback) {
	toggleCtrlButtons(c, "toggleUpdate")
}

func unsubCmdCtr(m *tb.Message) {

	url, mention := GetURLAndMentionFromMessage(m)

	if mention == "" {
		if url != "" {
			//Unsub by url
			source, _ := model.GetSourceByUrl(url)
			if source == nil {
				_, _ = B.Send(m.Chat, "Not subscribed to this RSS feed")
			} else {
				err := model.UnsubByUserIDAndSource(m.Chat.ID, source)
				if err == nil {
					_, _ = B.Send(
						m.Chat,
						fmt.Sprintf("[%s](%s) 退订成功！", source.Title, source.Link),
						&tb.SendOptions{
							DisableWebPagePreview: true,
							ParseMode:             tb.ModeMarkdown,
						},
					)
					zap.S().Infof("%d unsubscribe [%d]%s %s", m.Chat.ID, source.ID, source.Title, source.Link)
				} else {
					_, err = B.Send(m.Chat, err.Error())
				}
			}
		} else {
			//Unsub by button

			subs, err := model.GetSubsByUserID(m.Chat.ID)

			if err != nil {
				errorCtr(m, "Bot error, please contact the administrator! Error code 01")
				return
			}

			if len(subs) > 0 {
				unsubFeedItemBtns := [][]tb.InlineButton{}

				for _, sub := range subs {

					source, err := model.GetSourceById(sub.SourceID)
					if err != nil {
						errorCtr(m, "Bot error, please contact the administrator! Error code 02")
						return
					}

					unsubFeedItemBtns = append(unsubFeedItemBtns, []tb.InlineButton{
						tb.InlineButton{
							Unique: "unsub_feed_item_btn",
							Text:   fmt.Sprintf("[%d] %s", sub.SourceID, source.Title),
							Data:   fmt.Sprintf("%d:%d:%d", sub.UserID, sub.ID, source.ID),
						},
					})
				}

				_, _ = B.Send(m.Chat, "Please select the source you want to unsubscribe", &tb.ReplyMarkup{
					InlineKeyboard: unsubFeedItemBtns,
				})
			} else {
				_, _ = B.Send(m.Chat, "There are currently no feeds")
			}
		}
	} else {
		if url != "" {
			channelChat, err := B.ChatByID(mention)
			if err != nil {
				_, _ = B.Send(m.Chat, "error")
				return
			}
			adminList, err := B.AdminsOf(channelChat)
			if err != nil {
				_, _ = B.Send(m.Chat, "error")
				return
			}

			senderIsAdmin := false
			for _, admin := range adminList {
				if m.Sender.ID == admin.User.ID {
					senderIsAdmin = true
				}
			}

			if !senderIsAdmin {
				_, _ = B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
				return
			}

			source, _ := model.GetSourceByUrl(url)
			sub, err := model.GetSubByUserIDAndURL(channelChat.ID, url)

			if err != nil {
				if err.Error() == "record not found" {
					_, _ = B.Send(
						m.Chat,
						fmt.Sprintf("Channel [%s](https://t.me/%s) not subscribed to this RSS feed", channelChat.Title, channelChat.Username),
						&tb.SendOptions{
							DisableWebPagePreview: true,
							ParseMode:             tb.ModeMarkdown,
						},
					)

				} else {
					_, _ = B.Send(m.Chat, "Failed to unsubscribe")
				}
				return

			}

			err = sub.Unsub()
			if err == nil {
				_, _ = B.Send(
					m.Chat,
					fmt.Sprintf("Channel [%s](https://t.me/%s) sucessfully [%s](%s) unsubscribed", channelChat.Title, channelChat.Username, source.Title, source.Link),
					&tb.SendOptions{
						DisableWebPagePreview: true,
						ParseMode:             tb.ModeMarkdown,
					},
				)
				zap.S().Infof("%d for [%d]%s unsubscribe %s", m.Chat.ID, source.ID, source.Title, source.Link)
			} else {
				_, err = B.Send(m.Chat, err.Error())
			}
			return

		}
		_, _ = B.Send(m.Chat, "Please use the '/unsub @ChannelID URL' command to unsubscribe the channel")
	}

}

func unsubFeedItemBtnCtr(c *tb.Callback) {

	if (c.Message.Chat.Type == tb.ChatGroup || c.Message.Chat.Type == tb.ChatSuperGroup) &&
		!userIsAdminOfGroup(c.Sender.ID, c.Message.Chat) {
		// check admin
		return
	}

	data := strings.Split(c.Data, ":")
	if len(data) == 3 {
		userID, _ := strconv.Atoi(data[0])
		subID, _ := strconv.Atoi(data[1])
		sourceID, _ := strconv.Atoi(data[2])
		source, _ := model.GetSourceById(uint(sourceID))

		rtnMsg := fmt.Sprintf("[%d] <a href=\"%s\">%s</a> Successfully unsubscribed", sourceID, source.Link, source.Title)

		err := model.UnsubByUserIDAndSubID(int64(userID), uint(subID))

		if err == nil {
			_, _ = B.Edit(
				c.Message,
				rtnMsg,
				&tb.SendOptions{
					ParseMode: tb.ModeHTML,
				},
			)
			return
		}
	}
	_, _ = B.Edit(c.Message, "Unsubscribe error！")
}

func unsubAllCmdCtr(m *tb.Message) {
	mention := GetMentionFromMessage(m)
	confirmKeys := [][]tb.InlineButton{}
	confirmKeys = append(confirmKeys, []tb.InlineButton{
		tb.InlineButton{
			Unique: "unsub_all_confirm_btn",
			Text:   "Confirm",
		},
		tb.InlineButton{
			Unique: "unsub_all_cancel_btn",
			Text:   "Cancel",
		},
	})

	var msg string

	if mention == "" {
		msg = "Do you want to unsubscribe from all subscriptions of the current user?"
	} else {
		msg = fmt.Sprintf("%s Do you want to unsubscribe from all subscriptions of this Channel?", mention)
	}

	_, _ = B.Send(
		m.Chat,
		msg,
		&tb.SendOptions{
			ParseMode: tb.ModeHTML,
		}, &tb.ReplyMarkup{
			InlineKeyboard: confirmKeys,
		},
	)
}

func unsubAllCancelBtnCtr(c *tb.Callback) {
	_, _ = B.Edit(c.Message, "Operation canceled")
}

func unsubAllConfirmBtnCtr(c *tb.Callback) {
	mention := GetMentionFromMessage(c.Message)
	var msg string
	if mention == "" {
		success, fail, err := model.UnsubAllByUserID(int64(c.Sender.ID))
		if err != nil {
			msg = "Failed to unsubscribe"
		} else {
			msg = fmt.Sprintf("Sucessfully Unsubscribed：%d\nFailed to unsubscribe：%d", success, fail)
		}

	} else {
		channelChat, err := B.ChatByID(mention)

		if err != nil {
			_, _ = B.Edit(c.Message, "error")
			return
		}

		if UserIsAdminChannel(c.Sender.ID, channelChat) {
			success, fail, err := model.UnsubAllByUserID(channelChat.ID)
			if err != nil {
				msg = "Failed to unsubscribe"

			} else {
				msg = fmt.Sprintf("Sucessfully Unsubscribed：%d\nFailed to unsubscribe：%d", success, fail)
			}

		} else {
			msg = "Non-channel managers cannot perform this operation"
		}
	}

	_, _ = B.Edit(c.Message, msg)
}

func pingCmdCtr(m *tb.Message) {
	_, _ = B.Send(m.Chat, "pong")
	zap.S().Debugw(
		"pong",
		"telegram msg", m,
	)
}

func helpCmdCtr(m *tb.Message) {
	message := `
Commands：
/sub [url] Subscribe (url is optional)
/unsub [url] Unsubscribe (url is optional)
/list View current subscriptions
/set Set subscription
/check check current subscription
/setfeedtag [sub id] [tag1] [tag2] Set the subscription tag (set up to three tags, separated by spaces)
/setinterval [interval] [sub id] Set subscription refresh frequency (multiple sub ids can be set, separated by spaces)
/activeall open all subscriptions
/pauseall Pause all subscriptions
/import Import OPML files
/export Export OPML file
/unsuball cancel all subscriptions
/help Get this message

For detailed usage, please see: https://github.com/reaitten/flowerss-bot 
`

	_, _ = B.Send(m.Chat, message)
}

func versionCmdCtr(m *tb.Message) {
	_, _ = B.Send(m.Chat, config.AppVersionInfo())
}

func importCmdCtr(m *tb.Message) {
	message := `Please send the OPML file directly,
If you need to import OPML for the channel, please include the channel ID when sending the file, Example: @telegram
`
	_, _ = B.Send(m.Chat, message)
}

func setFeedTagCmdCtr(m *tb.Message) {
	mention := GetMentionFromMessage(m)
	args := strings.Split(m.Payload, " ")

	if len(args) < 1 {
		B.Send(m.Chat, "/setfeedtag [sub id] [tag1] [tag2] Set subscription tags (up to three tags, separated by spaces)")
		return
	}

	var subID int
	var err error
	if mention == "" {
		// Truncation parameter
		if len(args) > 4 {
			args = args[:4]
		}
		subID, err = strconv.Atoi(args[0])
		if err != nil {
			B.Send(m.Chat, "Please enter the correct subscription ID!")
			return
		}
	} else {
		if len(args) > 5 {
			args = args[:5]
		}
		subID, err = strconv.Atoi(args[1])
		if err != nil {
			B.Send(m.Chat, "Please enter the correct subscription ID!")
			return
		}
	}

	sub, err := model.GetSubscribeByID(subID)
	if err != nil || sub == nil {
		B.Send(m.Chat, "Please enter the correct subscription ID!")
		return
	}

	if !checkPermit(int64(m.Sender.ID), sub.UserID) {
		B.Send(m.Chat, "Permission denied!")
		return
	}

	if mention == "" {
		err = sub.SetTag(args[1:])
	} else {
		err = sub.SetTag(args[2:])
	}

	if err != nil {
		B.Send(m.Chat, "Failed to set subscription label!")
		return
	}
	B.Send(m.Chat, "The subscription label is set successfully!")
}

func setIntervalCmdCtr(m *tb.Message) {

	args := strings.Split(m.Payload, " ")

	if len(args) < 1 {
		_, _ = B.Send(m.Chat, "/setinterval [interval] [sub id] Set subscription refresh frequency (multiple sub id can be set, separated by spaces)")
		return
	}

	interval, err := strconv.Atoi(args[0])
	if interval <= 0 || err != nil {
		_, _ = B.Send(m.Chat, "Please enter the correct crawl frequency")
		return
	}

	for _, id := range args[1:] {

		subID, err := strconv.Atoi(id)
		if err != nil {
			_, _ = B.Send(m.Chat, "Please enter the correct subscription id!")
			return
		}

		sub, err := model.GetSubscribeByID(subID)

		if err != nil || sub == nil {
			_, _ = B.Send(m.Chat, "Please enter the correct subscription id!")
			return
		}

		if !checkPermit(int64(m.Sender.ID), sub.UserID) {
			_, _ = B.Send(m.Chat, "Permission denied!")
			return
		}

		_ = sub.SetInterval(interval)

	}
	_, _ = B.Send(m.Chat, "Crawl frequency is set successfully!")

	return
}

func activeAllCmdCtr(m *tb.Message) {
	mention := GetMentionFromMessage(m)
	if mention != "" {
		channelChat, err := B.ChatByID(mention)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}
		adminList, err := B.AdminsOf(channelChat)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}

		senderIsAdmin := false
		for _, admin := range adminList {
			if m.Sender.ID == admin.User.ID {
				senderIsAdmin = true
			}
		}

		if !senderIsAdmin {
			_, _ = B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
			return
		}

		_ = model.ActiveSourcesByUserID(channelChat.ID)
		message := fmt.Sprintf("Channel [%s](https://t.me/%s) subscriptions are all turned on", channelChat.Title, channelChat.Username)

		_, _ = B.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
		})

	} else {
		_ = model.ActiveSourcesByUserID(m.Chat.ID)
		message := "Subscriptions are all turned on"

		_, _ = B.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
		})
	}

}

func pauseAllCmdCtr(m *tb.Message) {
	mention := GetMentionFromMessage(m)
	if mention != "" {
		channelChat, err := B.ChatByID(mention)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}
		adminList, err := B.AdminsOf(channelChat)
		if err != nil {
			_, _ = B.Send(m.Chat, "error")
			return
		}

		senderIsAdmin := false
		for _, admin := range adminList {
			if m.Sender.ID == admin.User.ID {
				senderIsAdmin = true
			}
		}

		if !senderIsAdmin {
			_, _ = B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
			return
		}

		_ = model.PauseSourcesByUserID(channelChat.ID)
		message := fmt.Sprintf("Channel [%s](https://t.me/%s) all subscriptions are suspended", channelChat.Title, channelChat.Username)

		_, _ = B.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
		})

	} else {
		_ = model.PauseSourcesByUserID(m.Chat.ID)
		message := "All subscriptions are suspended"

		_, _ = B.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
		})
	}

}

func textCtr(m *tb.Message) {
	switch UserState[m.Chat.ID] {
	case fsm.UnSub:
		{
			str := strings.Split(m.Text, " ")

			if len(str) < 2 && (strings.HasPrefix(str[0], "[") && strings.HasSuffix(str[0], "]")) {
				_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
			} else {

				var sourceID uint
				if _, err := fmt.Sscanf(str[0], "[%d]", &sourceID); err != nil {
					_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
					return
				}

				source, err := model.GetSourceById(sourceID)

				if err != nil {
					_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
					return
				}

				err = model.UnsubByUserIDAndSource(m.Chat.ID, source)

				if err != nil {
					_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
					return
				}

				_, _ = B.Send(
					m.Chat,
					fmt.Sprintf("[%s](%s) successfully unsubscribed", source.Title, source.Link),
					&tb.SendOptions{
						ParseMode: tb.ModeMarkdown,
					}, &tb.ReplyMarkup{
						ReplyKeyboardRemove: true,
					},
				)
				UserState[m.Chat.ID] = fsm.None
				return
			}
		}

	case fsm.Sub:
		{
			url := strings.Split(m.Text, " ")
			if !CheckURL(url[0]) {
				_, _ = B.Send(m.Chat, "Please reply to the correct URL", &tb.ReplyMarkup{ForceReply: true})
				return
			}

			registFeed(m.Chat, url[0])
			UserState[m.Chat.ID] = fsm.None
		}
	case fsm.SetSubTag:
		{
			return
		}
	case fsm.Set:
		{
			str := strings.Split(m.Text, " ")
			url := str[len(str)-1]
			if len(str) != 2 && !CheckURL(url) {
				_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
			} else {
				source, err := model.GetSourceByUrl(url)

				if err != nil {
					_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
					return
				}
				sub, err := model.GetSubscribeByUserIDAndSourceID(m.Chat.ID, source.ID)
				if err != nil {
					_, _ = B.Send(m.Chat, "Please choose the correct instruction!")
					return
				}
				t := template.New("setting template")
				_, _ = t.Parse(feedSettingTmpl)

				toggleNoticeKey := tb.InlineButton{
					Unique: "set_toggle_notice_btn",
					Text:   "Turn on notifications",
				}
				if sub.EnableNotification == 1 {
					toggleNoticeKey.Text = "Close notification"
				}

				toggleTelegraphKey := tb.InlineButton{
					Unique: "set_toggle_telegraph_btn",
					Text:   "Turn on Telegraph transcoding",
				}
				if sub.EnableTelegraph == 1 {
					toggleTelegraphKey.Text = "Turn off Telegraph transcoding"
				}

				toggleEnabledKey := tb.InlineButton{
					Unique: "set_toggle_update_btn",
					Text:   "Pause update",
				}

				if source.ErrorCount >= config.ErrorThreshold {
					toggleEnabledKey.Text = "Restart update"
				}

				feedSettingKeys := [][]tb.InlineButton{
					[]tb.InlineButton{
						toggleEnabledKey,
						toggleNoticeKey,
						toggleTelegraphKey,
					},
				}

				text := new(bytes.Buffer)

				_ = t.Execute(text, map[string]interface{}{"source": source, "sub": sub, "Count": config.ErrorThreshold})

				// send null message to remove old keyboard
				delKeyMessage, err := B.Send(m.Chat, "processing", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				err = B.Delete(delKeyMessage)

				_, _ = B.Send(
					m.Chat,
					text.String(),
					&tb.SendOptions{
						ParseMode: tb.ModeHTML,
					}, &tb.ReplyMarkup{
						InlineKeyboard: feedSettingKeys,
					},
				)
				UserState[m.Chat.ID] = fsm.None
			}
		}
	}
}

// docCtr Document handler
func docCtr(m *tb.Message) {
	if m.FromGroup() {
		if !userIsAdminOfGroup(m.Sender.ID, m.Chat) {
			return
		}
	}

	if m.FromChannel() {
		if !UserIsAdminChannel(m.ID, m.Chat) {
			return
		}
	}

	url, _ := B.FileURLByID(m.Document.FileID)
	if !strings.HasSuffix(url, ".opml") {
		B.Send(m.Chat, "If you need to import subscriptions, please send the correct OPML file.")
		return
	}

	opml, err := GetOPMLByURL(url)
	if err != nil {
		if err.Error() == "fetch opml file error" {
			_, _ = B.Send(m.Chat,
				"Failed to download the OPML file. Please check whether the bot server can connect to the Telegram server normally or try to import it later. Error code 02")

		} else {
			_, _ = B.Send(
				m.Chat,
				fmt.Sprintf(
					"If you need to import subscriptions, please send the correct OPML file. Error code 01, doc mimetype: %s",
					m.Document.MIME),
			)
		}
		return
	}

	userID := m.Chat.ID
	mention := GetMentionFromMessage(m)
	if mention != "" {
		// import for channel
		channelChat, err := B.ChatByID(mention)
		if err != nil {
			_, _ = B.Send(m.Chat, "Get channel information error, please check if the channel id is correct")
			return
		}

		if !checkPermitOfChat(int64(m.Sender.ID), channelChat) {
			_, _ = B.Send(m.Chat, fmt.Sprintf("Non-channel managers cannot perform this operation"))
			return
		}

		userID = channelChat.ID
	}

	message, _ := B.Send(m.Chat, "Processing, please wait...")
	outlines, _ := opml.GetFlattenOutlines()
	var failImportList []Outline
	var successImportList []Outline

	for _, outline := range outlines {
		source, err := model.FindOrNewSourceByUrl(outline.XMLURL)
		if err != nil {
			failImportList = append(failImportList, outline)
			continue
		}
		err = model.RegistFeed(userID, source.ID)
		if err != nil {
			failImportList = append(failImportList, outline)
			continue
		}
		zap.S().Infof("%d subscribe [%d]%s %s", m.Chat.ID, source.ID, source.Title, source.Link)
		successImportList = append(successImportList, outline)
	}

	importReport := fmt.Sprintf("<b>Imported successfully：%d，Import failed：%d</b>", len(successImportList), len(failImportList))
	if len(successImportList) != 0 {
		successReport := "\n\n<b>The following feeds were imported successfully:</b>"
		for i, line := range successImportList {
			if line.Text != "" {
				successReport += fmt.Sprintf("\n[%d] <a href=\"%s\">%s</a>", i+1, line.XMLURL, line.Text)
			} else {
				successReport += fmt.Sprintf("\n[%d] %s", i+1, line.XMLURL)
			}
		}
		importReport += successReport
	}

	if len(failImportList) != 0 {
		failReport := "\n\n<b>The following feed failed to import:</b>"
		for i, line := range failImportList {
			if line.Text != "" {
				failReport += fmt.Sprintf("\n[%d] <a href=\"%s\">%s</a>", i+1, line.XMLURL, line.Text)
			} else {
				failReport += fmt.Sprintf("\n[%d] %s", i+1, line.XMLURL)
			}
		}
		importReport += failReport
	}

	_, _ = B.Edit(message, importReport, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeHTML,
	})
}

func errorCtr(m *tb.Message, errMsg string) {
	_, _ = B.Send(m.Chat, errMsg)
}

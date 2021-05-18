package bot

import (
	"go.uber.org/zap"
	"time"

	"github.com/reaitten/flowerss-bot/bot/fsm"
	"github.com/reaitten/flowerss-bot/config"
	"github.com/reaitten/flowerss-bot/util"
	tb "gopkg.in/tucnak/telebot.v2"
)

var (
	// User state，Used to indicate the status of the current user operation
	UserState map[int64]fsm.UserStatus = make(map[int64]fsm.UserStatus)

	// B telebot
	B *tb.Bot
)

func init() {
	if config.RunMode == config.TestMode {
		return
	}
	poller := &tb.LongPoller{Timeout: 10 * time.Second}
	spamProtected := tb.NewMiddlewarePoller(poller, func(upd *tb.Update) bool {
		if !isUserAllowed(upd) {
			// Check if the user can use the bot
			return false
		}

		if !CheckAdmin(upd) {
			return false
		}
		return true
	})
	zap.S().Infow("init telegram bot",
		"token", config.BotToken,
		"endpoint", config.TelegramEndpoint,
	)

	// create bot
	var err error

	B, err = tb.NewBot(tb.Settings{
		URL:    config.TelegramEndpoint,
		Token:  config.BotToken,
		Poller: spamProtected,
		Client: util.HttpClient,
	})

	if err != nil {
		zap.S().Fatal(err)
		return
	}
}

//Start bot
func Start() {
	if config.RunMode != config.TestMode {
		zap.S().Infof("bot start %s", config.AppVersionInfo())
		setCommands()
		setHandle()
		B.Start()
	}
}

func setCommands() {
	// Set bot command prompt information
	commands := []tb.Command{
		{"start", "Get started"},
		{"sub", "Subscribe to RSS feed"},
		{"list", "RSS feeds currently subscribed"},
		{"unsub", "Unsubscribe RSS feed"},
		{"unsuball", "Unsubscribe from all RSS sources"},

		{"set", "Set up RSS subscription"},
		{"setfeedtag", "Set RSS subscription label"},
		{"setinterval", "Set RSS subscription crawl interval"},

		{"export", "Export subscription as OPML file"},
			{"import", "Import subscription from OPML file"},

		{"check", "Check the status of my RSS subscription"},
		{"pauseall", "Pause all crawling subscription updates"},
		{"activeall", "Turn on fetching subscription updates"},

		{"help", "Get a list of available commands"},
		{"version", "Bot Version"},
	}

	zap.S().Debugf("set bot command %+v", commands)

	if err := B.SetCommands(commands); err != nil {
		zap.S().Errorw("set bot commands failed", "error", err.Error())
	}
}

func setHandle() {
	B.Handle(&tb.InlineButton{Unique: "set_feed_item_btn"}, setFeedItemBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "set_toggle_notice_btn"}, setToggleNoticeBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "set_toggle_telegraph_btn"}, setToggleTelegraphBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "set_toggle_update_btn"}, setToggleUpdateBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "set_set_sub_tag_btn"}, setSubTagBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "unsub_all_confirm_btn"}, unsubAllConfirmBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "unsub_all_cancel_btn"}, unsubAllCancelBtnCtr)

	B.Handle(&tb.InlineButton{Unique: "unsub_feed_item_btn"}, unsubFeedItemBtnCtr)

	B.Handle("/start", startCmdCtr)

	B.Handle("/export", exportCmdCtr)

	B.Handle("/sub", subCmdCtr)

	B.Handle("/list", listCmdCtr)

	B.Handle("/set", setCmdCtr)

	B.Handle("/unsub", unsubCmdCtr)

	B.Handle("/unsuball", unsubAllCmdCtr)

	B.Handle("/ping", pingCmdCtr)

	B.Handle("/help", helpCmdCtr)

	B.Handle("/import", importCmdCtr)

	B.Handle("/setfeedtag", setFeedTagCmdCtr)

	B.Handle("/setinterval", setIntervalCmdCtr)

	B.Handle("/check", checkCmdCtr)

	B.Handle("/activeall", activeAllCmdCtr)

	B.Handle("/pauseall", pauseAllCmdCtr)

	B.Handle("/version", versionCmdCtr)

	B.Handle(tb.OnText, textCtr)

	B.Handle(tb.OnDocument, docCtr)
}

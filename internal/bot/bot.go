package bot

import (
	"time"

	"github.com/reaitten/flowerss-bot/internal/bot/fsm"
	"github.com/reaitten/flowerss-bot/internal/config"
	"github.com/reaitten/flowerss-bot/internal/util"

	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

var (
	// User state, Used to indicate the status of the current user operation
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
		{Text: "start", Description: "Get started"},
		{Text: "sub", Description: "Subscribe to RSS feed"},
		{Text: "list", Description: "RSS feeds currently subscribed"},
		{Text: "unsub", Description: "Unsubscribe RSS feed"},
		{Text: "unsuball", Description: "Unsubscribe from all RSS sources"},

		{Text: "set", Description: "Set up RSS subscription"},
		{Text: "setfeedtag", Description: "Set RSS subscription label"},
		{Text: "setinterval", Description: "Set RSS subscription crawl interval"},

		{Text: "export", Description: "Export subscription as OPML file"},
		{Text: "import", Description: "Import subscription from OPML file"},

		{Text: "check", Description: "Check the status of my RSS subscription"},
		{Text: "pauseall", Description: "Pause all crawling subscription updates"},
		{Text: "activeall", Description: "Turn on fetching subscription updates"},

		{Text: "help", Description: "Get a list of available commands"},
		{Text: "version", Description: "Bot Version"},
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

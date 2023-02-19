package main

import (
	gstrings "strings"
	"time"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/core-utils/strings"
	"github.com/go-zoox/debug"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/retry"

	"github.com/eatmoreapple/openwechat"

	chatgpt "github.com/go-zoox/chatgpt-client"
)

type FeishuBotConfig struct {
	ChatGPTAPIKey string
	AdminNickname string
}

func ServeWechatBot(cfg *FeishuBotConfig) (err error) {
	isInSerice := true
	var admin *openwechat.Friend

	client, err := chatgpt.New(&chatgpt.Config{
		APIKey: cfg.ChatGPTAPIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create chatgpt client: %v", err)
	}

	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式，上面登录不上的可以尝试切换这种模式

	var self *openwechat.Self

	bot.MessageHandler = func(msg *openwechat.Message) {
		// exit if not a text message
		if !msg.IsText() {
			return
		}

		isAdmin := func() bool {
			return !msg.IsSendByGroup() && admin != nil && msg.FromUserName == admin.UserName
		}

		// ADMIN
		if isAdmin() {
			command := msg.Content
			// command
			switch command {
			case "service::stop", "stop", "休息", "下线", "去睡觉吧":
				isInSerice = false
				msg.ReplyText("服务已下线")
				return
			case "service::start", "start", "启动", "上线", "快醒来":
				isInSerice = true
				msg.ReplyText("服务上线")
				return
			case "半小时后下线":
				msg.ReplyText("好的，服务将在半小时后下线")
				go func(msg *openwechat.Message) {
					time.Sleep(30 * time.Minute)
					isInSerice = false
					msg.ReplyText("服务已下线")
				}(msg)
				return
			}
		}

		if debug.IsDebugMode() {
			fmt.PrintJSON(msg)
		}

		if !msg.IsAt() {
			return
		}

		if debug.IsDebugMode() {
			fmt.Printf("StartsWith: #%s#%s#%v", msg.Content, fmt.Sprintf("@%s", self.NickName), strings.StartsWith(msg.Content, fmt.Sprintf("@%s\u2005", self.NickName)))
		}

		// 注意，这里有不可见字符出现
		if !strings.StartsWith(msg.Content, fmt.Sprintf("@%s", self.NickName)) {
			return
		}

		if !isInSerice {
			msg.ReplyText("休息中 ...")
			return
		}

		chatID := msg.FromUserName
		user := msg.FromUserName
		question := msg.Content[len(fmt.Sprintf("@%s", self.NickName))+1:]
		question = strings.TrimSpace(question)
		question = gstrings.TrimLeft(question, "\ufffd")

		// 群聊触发
		// msg.IsComeFromGroup()
		// 	 @: msg.IsAt()
		//   /chatgpt:

		// 私聊触发 => /chatgpt
		if debug.IsDebugMode() {
			fmt.Printf("[%s] %s => %s\n", msg.Owner().NickName, question, self.NickName)
		}

		logger.Infof("问：%s", question)
		var err error

		var answer []byte
		err = retry.Retry(func() error {
			conversation, err := client.GetOrCreateConversation(chatID, &chatgpt.ConversationConfig{
				MaxMessages: 50,
			})
			if err != nil {
				return fmt.Errorf("failed to get or create conversation by ChatID %s", chatID)
			}

			answer, err = conversation.Ask([]byte(question), &chatgpt.ConversationAskConfig{
				User: user,
			})
			if err != nil {
				logger.Errorf("failed to request answer: %v", err)
				return fmt.Errorf("failed to request answer: %v", err)
			}

			return nil
		}, 5, 3*time.Second)
		if err != nil {
			logger.Errorf("failed to get answer: %v", err)

			if _, err := msg.ReplyText("ChatGPT 繁忙，请稍后重试"); err != nil {
				return
			}
			return
		}

		logger.Infof("答：%s", answer)

		answerX := string(answer)
		if msg.IsSendByGroup() {
			answerX = fmt.Sprintf("%s\n-------------\n%s", question, answer)
		}

		msg.ReplyText(answerX)
	}

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err := bot.Login(); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}

	// 获取登陆的用户
	self, err = bot.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	if cfg.AdminNickname != "" {
		friends, err := self.Friends()
		if err != nil {
			return fmt.Errorf("failed to list friends: %v", err)
		}
		admin = friends.GetByNickName(cfg.AdminNickname)
	}

	fmt.PrintJSON(map[string]any{
		"cfg":   cfg,
		"bot":   self,
		"admin": admin,
	})

	return bot.Block()
}

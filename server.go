package main

import (
	gstrings "strings"
	"time"

	"github.com/go-zoox/chatbot-wechat"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/core-utils/strings"
	"github.com/go-zoox/debug"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/retry"

	chatgpt "github.com/go-zoox/chatgpt-client"
)

type FeishuBotConfig struct {
	ChatGPTAPIKey string
	AdminNickname string
	ReportURL     string
}

func ServeWechatBot(cfg *FeishuBotConfig) (err error) {
	bot, err := chatbot.New(&chatbot.Config{
		AdminNickname: cfg.AdminNickname,
		ReprtURL:      cfg.ReportURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create wechat chatbot: %v", err)
	}

	client, err := chatgpt.New(&chatgpt.Config{
		APIKey: cfg.ChatGPTAPIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create chatgpt client: %v", err)
	}

	bot.OnOffline(func(request *chatbot.EventRequest, reply func(content string, msgType ...string) error) error {
		return reply("休息中 ...")
	})

	bot.OnCommand("醒来", &chatbot.Command{
		Handler: func(args []string, request *chatbot.EventRequest, reply chatbot.MessageReply) error {
			if err := bot.SetOnline(); err != nil {
				return err
			}

			return reply("醒啦")
		},
	})

	bot.OnCommand("去睡觉", &chatbot.Command{
		Handler: func(args []string, request *chatbot.EventRequest, reply chatbot.MessageReply) error {
			if err := bot.SetOffline(); err != nil {
				return err
			}

			return reply("睡啦")
		},
	})

	bot.OnCommand("半小时后下线", &chatbot.Command{
		Handler: func(args []string, request *chatbot.EventRequest, reply chatbot.MessageReply) error {
			go func() {
				time.Sleep(30 * time.Minute)
				if err := bot.SetOffline(); err != nil {
					logger.Errorf("failed to set offline: %v", err)
				}

				reply("睡啦")
			}()

			return reply("好的，服务将在半小时后下线")
		},
	})

	bot.OnCommand("重置", &chatbot.Command{
		Handler: func(args []string, request *chatbot.EventRequest, reply chatbot.MessageReply) error {
			if err := client.ResetConversations(); err != nil {
				reply("重置失败")
				return err
			}

			return reply("重置完成")
		},
	})

	bot.OnMessage(func(content string, msg *chatbot.EventRequest, reply chatbot.MessageReply) (err error) {
		if !msg.IsAt() {
			return
		}

		self, err := bot.Info()
		if err != nil {
			return
		}

		if debug.IsDebugMode() {
			fmt.Printf("StartsWith: #%s#%s#%v", msg.Content, fmt.Sprintf("@%s", self.NickName), strings.StartsWith(msg.Content, fmt.Sprintf("@%s\u2005", self.NickName)))
		}

		// 注意，这里有不可见字符出现
		if !strings.StartsWith(msg.Content, fmt.Sprintf("@%s", self.NickName)) {
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

			if err := reply("ChatGPT 繁忙，请稍后重试"); err != nil {
				return err
			}

			return
		}

		logger.Infof("答：%s", answer)

		answerX := string(answer)
		if msg.IsSendByGroup() {
			answerX = fmt.Sprintf("%s\n-------------\n%s", question, answer)
		}

		return reply(answerX)
	})

	return bot.Run()
}

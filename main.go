package main

import (
	"github.com/go-zoox/cli"
)

func main() {
	app := cli.NewSingleProgram(&cli.SingleProgramConfig{
		Name:    "chatgpt-for-chatbot-wechat",
		Usage:   "chatgpt-for-chatbot-wechat is a portable chatgpt server",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "chatgpt-api-key",
				Usage:    "ChatGPT API Key",
				EnvVars:  []string{"CHATGPT_API_KEY"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "admin-nickname",
				Usage:   "The admin nickname for advanced commands",
				EnvVars: []string{"ADMIN_NICKNAME"},
			},
			&cli.StringFlag{
				Name:    "report-url",
				Usage:   "Report URL for send qrcode url",
				EnvVars: []string{"REPORT_URL"},
			},
		},
	})

	app.Command(func(ctx *cli.Context) (err error) {
		return ServeWechatBot(&FeishuBotConfig{
			ChatGPTAPIKey: ctx.String("chatgpt-api-key"),
			AdminNickname: ctx.String("admin-nickname"),
			ReportURL:     ctx.String("report-url"),
		})
	})

	app.Run()
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gempir/go-twitch-irc/v4"
)

// Config is 設定ファイルの構造体
type Config struct {
	BotName      string `json:"botName"`
	ChannelName  string `json:"channelName"`
	OauthToken   string `json:"oauthToken"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"ClientSecret"`
	RedirectUri  string `json:"redirectUri"`
}

func main() {

	configPath := "config.json"

	// 設定ファイルを読み込む
	configJSON, loaderr := loadConfig(configPath)
	if loaderr != nil {
		log.Fatalf("Failed to load config file: %s", loaderr.Error())
	}

	// BotのTwitchアカウント名とOAuthトークンを設定
	botUsername := configJSON.BotName
	oauthToken := configJSON.OauthToken
	joinChannelName := configJSON.ChannelName

	// クライアントを作成
	client := twitch.NewClient(botUsername, oauthToken)

	// メッセージを受信したときの処理
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// ここでメッセージを表示する
		// 送信はどうするんだ？
		if len(message.Emotes) > 0 {
			fmt.Printf("> [%s/%v]%s %v  \n", message.User.Name, message.User.DisplayName, message.Message, message.Emotes)
		} else {
			fmt.Printf("<%s/%v]> %s  \n", message.User.Name, message.User.DisplayName, message.Message)
		}

		// 初回メッセージの場合は挨拶を返す
		if message.FirstMessage {
			first := fmt.Sprintf("<%s/%v]>さん,はじめまして!", message.User.Name, message.User.DisplayName)
			client.Say(joinChannelName, first)
		}
		// ビッツ付きの場合は挨拶を返す
		if message.Bits > 0 {
			client.Say(joinChannelName, fmt.Sprintf("%v ビッツありがとうございます!!", message.Bits))
		}

		client.Say(joinChannelName, "スタンプ対応もしたいなあァ")
	})

	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		fmt.Printf("%v joined the channel \n", message.User)
	})
	client.OnConnect(func() {
		fmt.Printf("Welcome to %v@Twitch ! \n", joinChannelName)
	})

	// チャンネルに参加
	client.Join(joinChannelName)

	// 接続開始
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
}

// 設定ファイルを読み込む関数
func loadConfig(path string) (*Config, error) {
	// 設定ファイルを読み込む
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// JSONデコード
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/go-resty/resty/v2"
)

// Config is 設定ファイルの構造体
type Config struct {
	BotName      string `json:"botName"`
	ChannelName  string `json:"channelName"`
	OauthToken   string `json:"oauthToken"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"ClientSecret"`
	RedirectUri  string `json:"redirectUri"`
	DeepLAPIKey  string `json:"deepLAPIKey"`
}

// Botメッセージ構造体
type BotMessages struct {
	OnConnect         string `json:"OnConnect"`
	OnUserJoinMessage string `json:"OnUserJoinMessage"`
}

var UsedMsg = ""

// JSONファイルを読み込む関数
func loadBotMessages(filePath string) (*BotMessages, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var messages BotMessages
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, err
	}

	return &messages, nil
}

func main() {

	// botMessage.json を読み込む
	botMessages, errJson := loadBotMessages("botMessage.json")
	if errJson != nil {
		log.Fatalf("Failed to load bot messages: %v", errJson)
	}

	configPath := "authconfig.json"

	// 発言数を記録するマップ
	var charUsrs = map[string]int{}

	// 設定ファイルを読み込む
	configJSON, loaderr := loadConfig(configPath)
	if loaderr != nil {
		log.Fatalf("Failed to load config file: %s", loaderr.Error())
	}

	// Twitchの配信状況を取得する
	PopStreaming(configJSON)

	// BotのTwitchアカウント名とOAuthトークンを設定
	botUsername := configJSON.BotName

	oauthToken := configJSON.OauthToken
	joinChannelName := configJSON.ChannelName

	// OAuthトークンが空の場合のチェック
	if oauthToken == "" {
		log.Fatalf("OAuth token is empty. Please check the 'oauthToken' field in %s", configPath)
	}

	// Twitchクライアントを作成
	client := twitch.NewClient(botUsername, oauthToken)

	// DeepL APIキーを設定
	deepLApiKey := configJSON.DeepLAPIKey

	// メッセージを受信したときの処理
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// 発言数を記録
		charUsrs[message.User.Name]++
		log.Printf("%s has sent %d messages.", message.User.Name, charUsrs[message.User.Name])

		// ここでメッセージを表示する
		// 送信はどうするんだ？
		if len(message.Emotes) > 0 {
			fmt.Printf("> [%s/%v]%s %v  \n", message.User.Name, message.User.DisplayName, message.Message, message.Emotes)
		} else {
			fmt.Printf("<%s/%v]> %s  \n", message.User.Name, message.User.DisplayName, message.Message)
		}

		joinName := fmt.Sprintf("<%s/%v]>", message.User.Name, message.User.DisplayName)

		// 初回メッセージの場合は挨拶を返す
		if message.FirstMessage {
			first := fmt.Sprintf("%vさん,はじめまして!", joinName)
			client.Say(joinChannelName, first)
		} else {

			if charUsrs[message.User.Name] == 1 {
				welcomeMsg := fmt.Sprintf("%v、、お会いできて何よりです。", joinName)
				client.Say(joinChannelName, welcomeMsg)
			}
		}

		// ビッツ付きの場合は挨拶を返す
		if message.Bits > 0 {
			client.Say(joinChannelName, fmt.Sprintf("%v ビッツありがとうございます!!", message.Bits))
		}

		// チャット欄にメッセージを送信する
		// 翻訳と組み合わせる？
		// それか固定文言を繰り返し送信するのも手？

		// とりあえずメッセージを送信する（いまは絶対送信しないように）
		sendFlg := false
		if sendFlg {
			botMsg := fmt.Sprintf("%v by %v.", message.Message, joinName)
			client.Say(joinChannelName, botMsg)
			log.Println(botMsg)

		}

		// チャット欄にメッセージを送信する
		translatedMsg, err := translateText(deepLApiKey, message.Message, "JA") // 日本語から英語に翻訳
		if err != nil {
			log.Printf("Failed to translate message: %v", err)

		} else if len(translatedMsg) > 0 {
			// 翻訳結果を表示
			log.Printf("<翻訳結果>%v \n", translatedMsg)
			botMsg := fmt.Sprintf("%v%v", translatedMsg, joinName)
			client.Say(joinChannelName, botMsg)
		}

	})

	//Twitchのチャットに新しいユーザがきたときらしい
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		joinMsg := fmt.Sprintf("<%v> hey!\n", message.User)
		log.Println(joinMsg)
		client.Say(joinChannelName, joinMsg)

	})

	// 接続時
	client.OnConnect(func() {

		connectionMsg := fmt.Sprintf("< Welcome to %v@Twitch >  ", joinChannelName)
		log.Println(connectionMsg)
		// チャット欄に送信するメッセージ
		if len(botMessages.OnConnect) > 0 {
			connectionMsg = botMessages.OnConnect
		}
		client.Say(joinChannelName, "Login... Success!")
		client.Say(joinChannelName, connectionMsg)
		client.Say(joinChannelName, UsedMsg)
	})

	// Botの起動時にDeepLの使用状況を確認
	characterCount, characterLimit, errAPI := getUsage(deepLApiKey)
	if errAPI != nil {
		log.Printf("Failed to get DeepL usage: %v", errAPI)
	} else {
		UsedMsg = fmt.Sprintf(">> DeepL Usage: %d/%d characters used. (DeepL 使用状況: %d/%d 文字を使用しました。)", characterCount, characterLimit, characterCount, characterLimit)
		log.Println(UsedMsg)
	}

	// チャンネルに参加
	client.Join(joinChannelName)
	log.Printf("%vにインしたお,", joinChannelName)

	// 接続開始
	err := client.Connect()
	if err != nil {
		if err.Error() == "login authentication failed" {
			log.Fatalf("Authentication failed: Please check your OAuth token in %s. Ensure it is valid and has the required permissions.\n (認証に失敗しました: %s 内の OAuth トークンを確認してください。有効で必要な権限があることを確認してください。)", configPath, configPath)
		} else {
			log.Fatalf("Failed to connect: %s", err.Error())
		}
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

// DeepL APIを使用してテキストを翻訳する関数
func translateText(apiKey, text, targetLang string) (string, error) {
	log.Println(text)

	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).
		SetQueryParams(map[string]string{
			"text":        text,
			"target_lang": targetLang,
		}).
		Get("https://api-free.deepl.com/v2/translate")

	if err != nil {
		return "", err
	}

	// DeepL APIのレスポンスを解析
	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return "", err
	}

	// 翻訳結果を取得
	translations := result["translations"].([]interface{})
	translatedText := translations[0].(map[string]interface{})["text"].(string)
	srcLang := translations[0].(map[string]interface{})["detected_source_language"].(string)
	// 翻訳結果の整形
	retText := fmt.Sprintf("%v (%v > %v) by DeepLAPI", translatedText, srcLang, targetLang)
	// エラーチェック
	if errorMsg, exists := result["message"]; exists {
		return "", fmt.Errorf("DeepL API error: %v", errorMsg)
	}
	// 翻訳元と翻訳先の言語が同じ場合は空文字を返す
	if srcLang == targetLang {
		retText = ""
	}

	return retText, nil
}

// DeepL APIを使用して現在の使用状況を取得する関数
func getUsage(apiKey string) (int, int, error) {
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).
		Get("https://api-free.deepl.com/v2/usage")

	if err != nil {
		return 0, 0, err
	}

	// DeepL APIのレスポンスを解析
	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return 0, 0, err
	}

	// 使用状況を取得
	characterCount, ok1 := result["character_count"].(float64)
	characterLimit, ok2 := result["character_limit"].(float64)
	if !ok1 || !ok2 {
		return 0, 0, fmt.Errorf("unexpected response format: %v", result)
	}

	return int(characterCount), int(characterLimit), nil
}

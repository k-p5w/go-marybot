package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gempir/go-twitch-irc"
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

func main() {

	fmt.Println("start!")

	chatBot()
}

func chatBot() {

	configPath := "config.json"

	// 設定ファイルを読み込む
	configJSON, loaderr := loadConfig(configPath)
	if loaderr != nil {
		log.Fatalf("Failed to load config file: %s", loaderr.Error())
	}

	// Twitchのチャットボットの設定
	botUsername := configJSON.BotName
	channel := configJSON.ChannelName

	// Twitch開発者ダッシュボードから取得したクライアントIDとクライアントの秘密を設定
	clientID := configJSON.ClientID
	clientSecret := configJSON.ClientSecret
	redirectURI := "https://localhost:3000"

	// アクセストークンを取得するためのURL
	authURL := "https://id.twitch.tv/oauth2/authorize"
	tokenURL := "https://id.twitch.tv/oauth2/token"

	// TwitchのOAuth 2.0認証フローを開始
	// ユーザーを認証ページにリダイレクト
	authorizationURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=chat:read+chat:edit", authURL, clientID, redirectURI)
	fmt.Println("Visit this URL to authorize the application:", authorizationURL)
	fmt.Printf("認証に必要な情報:%v/%v/%v/%v \n", clientID, clientSecret, CodeItem, redirectURI)
	// アクセストークンを取得
	data := strings.NewReader(fmt.Sprintf("client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=%s", clientID, clientSecret, CodeItem, redirectURI))
	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Access Token Response:", string(body))
	accessToken := string(body)

	// oauthToken := config.OauthToken
	fmt.Printf("start-NewClient.:%v \n ", botUsername)
	// accessToken
	// oauth2Config = &clientcredentials.Config{
	// 	ClientID:     config.ClientID,
	// 	ClientSecret: config.ClientSecret,
	// 	TokenURL:     twitch.Endpoint.TokenURL,
	// }

	// token, err := oauth2Config.Token(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	fmt.Printf("Access token: %s\n", accessToken)

	// 認証する
	// ここがうまく行かなくなってるなんでや
	client := twitch.NewClient(botUsername, accessToken)

	// ここで翻訳するかなあ
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// おーここでメッセージがとれる
		fmt.Printf("bot:%v:%v\n", message.Emotes, message.Message)
		TextLine = message.Message
		var recItem ChatMsgInfo
		recItem.IsTranslateText = false

		// メッセージを翻訳する
		if !recItem.IsTranslateText {
			// inputを英語にする
			got, err := translateTextEN("en", TextLine)
			if err != nil {
				fmt.Printf("ERR(translateText): %v", err)
				// 翻訳エラーのときも二度と翻訳しないようにする
				recItem.IsTranslateText = true
			}
			recItem.TranslateMessageEn = got
			// 日本語→英語が翻訳できなかった場合
			fmt.Printf("BASE[%v]\nTRAN[%v]\n ", TextLine, got)

			// 韓国語とか多言語も英語にできちゃうから最初に日本語にするか？？
			// 翻訳ちゃんはわかる言語にするっていうオプションだった

			// 他の言語のメッセージなので、日本語にする
			got, err = translateTextJA("ja", TextLine)
			if err != nil {
				fmt.Printf("ERR(translateText): %v", err)
				// 翻訳エラーのときも二度と翻訳しないようにする
				recItem.IsTranslateText = true
			}
			// 日本以外→日本語の場合
			recItem.TranslateMessageJa = got
			recItem.TranslateMessage = message.Message

		}

		recItem.MsgOrg = message

		RecMsg[message.Time] = recItem
		// fmt.Println(message)
		// timeLine:= message.Time
		// chatLog := message
		// recMsg = append(recMsg, chatLog)
		// つまり、これをHTML表示できれば解決だな？
		// 表示はできたので、これを配列に詰めて翻訳も含めたいところ。
	})

	// チャンネルにjoinする
	client.Join(channel)

	errCon := client.Connect()
	// 認証に失敗したとき
	if errCon != nil {
		panic(errCon)
	}
}

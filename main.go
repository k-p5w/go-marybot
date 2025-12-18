package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

var UsedMsg = ""

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

// 日本語判定
func containsJapanese(text string) bool {
	var re = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}]`)
	return re.MatchString(text)
}

func main() {
	_ = godotenv.Load()
	// main関数内
	myURL := os.Getenv("MY_URL")
	// 1. Webサーバー設定（Render寝落ち防止 & ローカルおまじない）
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Bot is running! DeepL Usage: %s", UsedMsg)
		})
		addr := ":" + port
		if os.Getenv("PORT") == "" {
			addr = "localhost:" + port
			log.Printf("Local debug mode: http://%s", addr)
		}
		_ = http.ListenAndServe(addr, nil)
	}()

	botUsername := os.Getenv("BOT_NAME")
	oauthToken := os.Getenv("OAUTH_TOKEN")
	joinChannelName := os.Getenv("CHANNEL_NAME")
	deepLApiKey := os.Getenv("DEEPL_API_KEY")

	if botUsername == "" || oauthToken == "" || joinChannelName == "" || deepLApiKey == "" {
		log.Fatal("Error: 環境変数が足りません")
	}

	client := twitch.NewClient(botUsername, oauthToken)
	var charUsrs = map[string]int{}

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.User.Name == botUsername {
			return
		}

		// Renderの「30分スリープタイマー」をリセットし続ける
		go http.Get(myURL)

		first := ""
		charUsrs[message.User.Name]++
		if charUsrs[message.User.Name] == 1 {
			first = "[新]"
		}

		// ✅ 日本語なら英語へ、それ以外（英語等）なら日本語へ
		targetLang := "JA"
		if containsJapanese(message.Message) {
			targetLang = "EN"

		}

		// 翻訳実行
		translatedMsg, err := translateText(deepLApiKey, message.Message, targetLang)
		if err != nil {
			log.Printf("Translation error: %v", err)
			return
		}

		if translatedMsg != "" {

			postUser := ""
			if len(message.User.DisplayName) > 0 {

				postUser = message.User.DisplayName
			} else {
				postUser = message.User.Name
			}

			// [新] 翻訳文 [by ユーザー名] (翻訳元 > 翻訳先)
			finalMsg := fmt.Sprintf("%s%s 【by %s】", first, translatedMsg, postUser)
			client.Say(joinChannelName, finalMsg)
		}
	})

	client.OnConnect(func() {
		log.Printf("Connected to %s", joinChannelName)
		count, limit, err := getUsage(deepLApiKey)
		if err == nil {
			UsedMsg = fmt.Sprintf("%d/%d", count, limit)
			startupMsg := fmt.Sprintf("⚙ システム起動… DeepL残量：%s ！双方向翻訳モード。今日も翻訳頑張るぞい！", UsedMsg)
			client.Say(joinChannelName, startupMsg)
		}
	})

	client.Join(joinChannelName)
	err := client.Connect()
	if err != nil {
		log.Fatal("Twitch connection error:", err)
	}
}

func translateText(apiKey, text, targetLang string) (string, error) {
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).
		SetQueryParams(map[string]string{
			"text":        text,
			"target_lang": targetLang,
		}).
		Post("https://api-free.deepl.com/v2/translate")

	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body(), &result)

	if translations, ok := result["translations"].([]interface{}); ok && len(translations) > 0 {
		t := translations[0].(map[string]interface{})
		resText := t["text"].(string)
		srcLang := t["detected_source_language"].(string)

		// 翻訳前後が同じ言語なら表示しない
		if srcLang == targetLang {
			return "", nil
		}

		return fmt.Sprintf("%s (%s > %s)", resText, srcLang, targetLang), nil
	}
	return "", nil
}

func getUsage(apiKey string) (int, int, error) {
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).
		Get("https://api-free.deepl.com/v2/usage")
	if err != nil {
		return 0, 0, err
	}
	var result map[string]interface{}
	json.Unmarshal(resp.Body(), &result)
	count := int(result["character_count"].(float64))
	limit := int(result["character_limit"].(float64))
	return count, limit, nil
}

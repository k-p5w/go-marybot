package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http" // Render無料枠での起動に必須
	"os"
	"regexp"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/go-resty/resty/v2"
)

// --- グローバル変数 ---
var UsedMsg = ""

// --- 日本語判定ロジック（DeepL節約用） ---
func containsJapanese(text string) bool {
	// ひらがな・カタカナが含まれているか判定
	var re = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}]`)
	return re.MatchString(text)
}

func main() {
	// 1. 【重要】RenderのFreeプランで動かすためのWebサーバー機能
	// これがないとRender側で「エラー」と判定されて止まってしまいます
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカルテスト用
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Bot is running! DeepL Usage: %s", UsedMsg)
		})
		log.Printf("Web server starting on port %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("Web server error:", err)
		}
	}()

	// 2. 環境変数から設定を読み込む
	botUsername := os.Getenv("BOT_NAME")
	oauthToken := os.Getenv("OAUTH_TOKEN")
	joinChannelName := os.Getenv("CHANNEL_NAME")
	deepLApiKey := os.Getenv("DEEPL_API_KEY")

	if botUsername == "" || oauthToken == "" || joinChannelName == "" || deepLApiKey == "" {
		log.Fatal("Error: 必須な環境変数が設定されていません (BOT_NAME, OAUTH_TOKEN, CHANNEL_NAME, DEEPL_API_KEY)")
	}

	// 3. Twitchクライアントの作成
	client := twitch.NewClient(botUsername, oauthToken)
	var charUsrs = map[string]int{}

	// 4. メッセージ受信時の処理
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// Bot自身の発言はスルー（無限ループ防止）
		if message.User.Name == botUsername {
			return
		}

		charUsrs[message.User.Name]++
		joinName := fmt.Sprintf("(%s)", message.User.DisplayName)

		// はじめての挨拶
		if charUsrs[message.User.Name] == 1 {
			client.Say(joinChannelName, fmt.Sprintf("%vさん、はじめまして！", message.User.DisplayName))
		}

		// --- 翻訳の最適化 ---
		// A. まず日本語が含まれているか自前でチェック（DeepL APIを叩かない）
		if containsJapanese(message.Message) {
			return 
		}

		// B. 日本語が含まれていない場合のみ DeepL API を呼ぶ
		translatedMsg, err := translateText(deepLApiKey, message.Message, "JA")
		if err != nil {
			log.Printf("Translation error: %v", err)
		} else if translatedMsg != "" {
			// 翻訳結果をチャットに投稿
			client.Say(joinChannelName, fmt.Sprintf("%v %v", translatedMsg, joinName))
		}
	})

	// 接続完了時のログ
	client.OnConnect(func() {
		log.Printf("Connected to %s", joinChannelName)
		client.Say(joinChannelName, "Translation Bot Online! (Free Tier Mode)")
		
		// 起動時にDeepL使用状況を更新
		count, limit, _ := getUsage(deepLApiKey)
		UsedMsg = fmt.Sprintf("%d/%d", count, limit)
		client.Say(joinChannelName, fmt.Sprintf("DeepL Usage: %s", UsedMsg))
	})

	// 5. 接続開始
	client.Join(joinChannelName)
	err := client.Connect()
	if err != nil {
		log.Fatal("Twitch connection error:", err)
	}
}

// --- DeepL API 翻訳関数 ---
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
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	if translations, ok := result["translations"].([]interface{}); ok && len(translations) > 0 {
		t := translations[0].(map[string]interface{})
		resText := t["text"].(string)
		srcLang := t["detected_source_language"].(string)

		// 判定漏れでDeepL側が日本語と判断した場合も無視
		if srcLang == targetLang {
			return "", nil
		}
		return fmt.Sprintf("[DeepL] %s", resText), nil
	}
	return "", nil
}

// --- DeepL API 使用量取得関数 ---
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
	return int(result["character_count"].(float64)), int(result["character_limit"].(float64)), nil
}
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

var UsedMsg = "unknown"
var reJapanese = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}]`)

// --- ここを追記：他のファイルが参照しているConfig構造体を定義 ---
type Config struct {
	BotName      string `json:"botName"`
	ChannelName  string `json:"channelName"`
	OauthToken   string `json:"oauthToken"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"ClientSecret"`
	RedirectUri  string `json:"redirectUri"`
	DeepLAPIKey  string `json:"deepLAPIKey"`
}

type TwitchStreamInfo struct {
	Data []struct {
		Title    string `json:"title"`
		GameName string `json:"game_name"`
	} `json:"data"`
}

func main() {
	_ = godotenv.Load()
	myURL := os.Getenv("MY_URL")

	// --- 1. 必須設定のチェック (足りないとここで終了) ---
	botUsername := os.Getenv("BOT_NAME")
	oauthToken := os.Getenv("OAUTH_TOKEN")
	joinChannelName := os.Getenv("CHANNEL_NAME")
	deepLApiKey := os.Getenv("DEEPL_API_KEY")

	if botUsername == "" || oauthToken == "" || joinChannelName == "" || deepLApiKey == "" {
		log.Fatal("❌ 必須設定(BOT_NAME, OAUTH_TOKEN, CHANNEL_NAME, DEEPL_API_KEY)が足りません。")
	}

	// オプション設定
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	// --- 2. Webサーバー設定 ---
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

	client := twitch.NewClient(botUsername, oauthToken)
	charUsrs := map[string]int{}

	// --- 3. メッセージ翻訳処理 ---
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.User.Name == botUsername {
			return
		}

		cleanMsg := message.Message
		for _, emote := range message.Emotes {
			cleanMsg = strings.ReplaceAll(cleanMsg, emote.Name, "")
		}
		if strings.TrimSpace(cleanMsg) == "" {
			return
		}

		go http.Get(myURL)

		targetLang := "JA"
		if reJapanese.MatchString(cleanMsg) {
			targetLang = "EN"
		}

		translatedMsg, err := translateText(deepLApiKey, cleanMsg, targetLang)
		if err != nil || translatedMsg == "" {
			return
		}

		postUser := message.User.DisplayName
		if postUser == "" {
			postUser = message.User.Name
		}

		first := ""
		charUsrs[message.User.Name]++
		if charUsrs[message.User.Name] == 1 {
			first = "[新]"
		}

		client.Say(joinChannelName, fmt.Sprintf("%s%s 【by %s】", first, translatedMsg, postUser))
	})

	// --- 4. 接続時：配信チェック ＆ Bluesky告知 ---
	client.OnConnect(func() {
		log.Printf("Connected to %s", joinChannelName)

		// DeepL残量取得
		count, limit, err := getUsage(deepLApiKey)
		if err == nil {
			UsedMsg = fmt.Sprintf("%d/%d", count, limit)
		}

		// Twitch配信情報取得
		info, err := getStreamInfo(joinChannelName, clientID, clientSecret)

		if err == nil && len(info.Data) > 0 {
			// 配信中の場合：Blueskyにリッチ告知
			stream := info.Data[0]
			streamURL := "https://twitch.tv/" + joinChannelName
			bskyMsg := fmt.Sprintf("🔴 配信開始！\n【%s】\nカテゴリ: %s\n\n%s",
				stream.Title, stream.GameName, streamURL)

			if bskyErr := postToBluesky(bskyMsg); bskyErr != nil {
				log.Printf("Bluesky post skipped/failed: %v", bskyErr)
			}
		} else if err == nil && len(info.Data) == 0 {
			// ID設定はあるが、配信してない場合は終了
			log.Println("配信中ではないため、Botを終了します。")
			os.Exit(0)
			return
		} else {
			// ID設定自体がない場合は、配信チェックを無視して翻訳Botとして継続
			log.Printf("Twitch API設定がないため、配信チェックをスキップして継続します。")
		}

		// Twitchチャットへの起動メッセージ
		client.Say(joinChannelName, fmt.Sprintf("⚙ システム起動… DeepL残量：%s ｜今年残り：%d週 ｜Status：ALL GREEN 🟢", UsedMsg, calculateRemainingWeeks()))
	})

	client.Join(joinChannelName)
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}
}

// --- 以下、各API用関数 (Facet/リンク対応済み) ---

func getStreamInfo(channelName, clientID, clientSecret string) (*TwitchStreamInfo, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("credentials not set")
	}
	r := resty.New()
	tResp, err := r.R().SetQueryParams(map[string]string{
		"client_id": clientID, "client_secret": clientSecret, "grant_type": "client_credentials",
	}).Post("https://id.twitch.tv/oauth2/token")
	if err != nil {
		return nil, err
	}
	var tData map[string]interface{}
	json.Unmarshal(tResp.Body(), &tData)
	token, ok := tData["access_token"].(string)
	if !ok {
		return nil, fmt.Errorf("token error")
	}
	resp, err := r.R().SetHeader("Client-ID", clientID).SetHeader("Authorization", "Bearer "+token).
		SetQueryParam("user_login", channelName).Get("https://api.twitch.tv/helix/streams")
	if err != nil {
		return nil, err
	}
	var info TwitchStreamInfo
	json.Unmarshal(resp.Body(), &info)
	return &info, nil
}

func postToBluesky(text string) error {
	handle := os.Getenv("BLUESKY_HANDLE")
	appPw := os.Getenv("BLUESKY_APP_PASSWORD")
	if handle == "" || appPw == "" {
		return fmt.Errorf("Bluesky settings missing")
	}
	loginJson, _ := json.Marshal(map[string]string{"identifier": handle, "password": appPw})
	resp, err := http.Post("https://bsky.social/xrpc/com.atproto.server.createSession", "application/json", bytes.NewBuffer(loginJson))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var session map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&session)
	token, ok := session["accessJwt"].(string)
	if !ok {
		return fmt.Errorf("auth failed")
	}

	// Facet処理（リンクを青くする）
	var facets []map[string]interface{}
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	matches := urlRegex.FindAllStringIndex(text, -1)
	for _, m := range matches {
		facets = append(facets, map[string]interface{}{
			"index":    map[string]interface{}{"byteStart": m[0], "byteEnd": m[1]},
			"features": []map[string]interface{}{{"$type": "app.bsky.richtext.facet#link", "uri": text[m[0]:m[1]]}},
		})
	}

	postData := map[string]interface{}{
		"repo": session["did"], "collection": "app.bsky.feed.post",
		"record": map[string]interface{}{"text": text, "facets": facets, "createdAt": time.Now().Format(time.RFC3339), "$type": "app.bsky.feed.post"},
	}
	postJson, _ := json.Marshal(postData)
	req, _ := http.NewRequest("POST", "https://bsky.social/xrpc/com.atproto.repo.createRecord", bytes.NewBuffer(postJson))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	_, err = (&http.Client{}).Do(req)
	return err
}

func translateText(apiKey, text, targetLang string) (string, error) {
	resp, err := resty.New().R().SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).
		SetQueryParams(map[string]string{"text": text, "target_lang": targetLang}).
		Post("https://api-free.deepl.com/v2/translate")
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	json.Unmarshal(resp.Body(), &result)
	if trans, ok := result["translations"].([]interface{}); ok && len(trans) > 0 {
		t := trans[0].(map[string]interface{})
		if t["detected_source_language"].(string) == targetLang {
			return "", nil
		}
		return fmt.Sprintf("%s (%s > %s)", t["text"].(string), t["detected_source_language"].(string), targetLang), nil
	}
	return "", nil
}

func getUsage(apiKey string) (int, int, error) {
	resp, err := resty.New().R().SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).Get("https://api-free.deepl.com/v2/usage")
	if err != nil {
		return 0, 0, err
	}
	var r map[string]interface{}
	json.Unmarshal(resp.Body(), &r)
	return int(r["character_count"].(float64)), int(r["character_limit"].(float64)), nil
}

func calculateRemainingWeeks() int {
	now := time.Now()
	endOfYear := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
	return (int(endOfYear.Sub(now).Hours()/24) + 6) / 7
}

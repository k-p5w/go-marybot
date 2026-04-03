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
	"github.com/k-p5w/go-marybot/internal/bandainamco"
)

var UsedMsg = "unknown"
var reJapanese = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}]`)

// バージョン情報の定義
const BotVersion = "!コマンド追加 e.g.!help" // アメイジア東対応 & HELP追加版

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
	// このハンドラはユーザーがチャットに送信したメッセージを受け取ります。
	// 日本語と英語を自動判定して相互翻訳し、翻訳済みメッセージをチャットに投稿します。
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {

		// 1. コマンドかどうか判定
		if strings.HasPrefix(message.Message, "!") {
			fullCmd := strings.TrimPrefix(message.Message, "!")
			args := strings.Split(fullCmd, " ")
			command := strings.ToLower(args[0])

			// --- 全てのゲームで共通して使えるコマンド ---
			switch command {
			case "!", "help":
				// 利用可能なコマンドを一覧表示
				helpMsg := "📖 利用可能コマンド: !status (botの状態), !syn (シンデュアのMAP状況)"
				client.Say(joinChannelName, helpMsg)
				return
			case "status":
				statusMsg := "⚙ bot-status | " + formatStatus(BotVersion, UsedMsg, calculateRemainingWeeks()) + " for " + joinChannelName
				client.Say(joinChannelName, statusMsg)
				return
			case "syn":
				// どの配信中であっても、!syn と打たれれば即座に回答
				msg := bandainamco.GetSynStatus(true) // 「今の状況」を返す関数
				client.Say(joinChannelName, msg)
				return
			case "ff14":
				// 同様に、FF14の情報をいつでも呼び出せる
				// msg := squareenix.GetFF14Status()
				// client.Say(joinChannelName, msg)
				return
			}
		}

		// ボット自身のメッセージは処理しない（無限ループ防止）
		if message.User.Name == botUsername {
			return
		}

		// ステップ1: エモートを除去してテキストを整形
		cleanMsg := message.Message
		for _, emote := range message.Emotes {
			cleanMsg = strings.ReplaceAll(cleanMsg, emote.Name, "")
		}
		// 空白のメッセージはスキップ
		if strings.TrimSpace(cleanMsg) == "" {
			return
		}

		// ステップ2: MY_URLをバックグラウンドで呼び出し（外部トリガー用）
		go http.Get(myURL)

		// ステップ3: 言語判定と翻訳言語の決定
		// 日本語を含む場合は英語に、それ以外は日本語に翻訳
		targetLang := "JA"
		if reJapanese.MatchString(cleanMsg) {
			targetLang = "EN"
		}

		// ステップ4: DeepL APIで翻訳実行
		translatedMsg, err := translateText(deepLApiKey, cleanMsg, targetLang)
		if err != nil || translatedMsg == "" {
			return
		}

		// ステップ5: ユーザー表示名を取得（ない場合はユーザーIDを使用）
		postUser := message.User.DisplayName
		if postUser == "" {
			postUser = message.User.Name
		}

		// ステップ6: 初投稿ユーザーに [新] タグを付与
		first := ""
		charUsrs[message.User.Name]++
		if charUsrs[message.User.Name] == 1 {
			first = "[新]"
		}

		// ステップ7: 翻訳済みメッセージをチャットに投稿
		client.Say(joinChannelName, fmt.Sprintf("%s%s 【by %s】", first, translatedMsg, postUser))
	})

	// --- 4. 接続時：配信チェック ＆ Bluesky告知 ---
	// Twitch チャットへの接続が確立されたときに実行されます。
	// 配信情報を取得し、配信中であれば Bluesky に告知投稿します。
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
			// os.Exit(0)
			// return
		} else {
			// ID設定自体がない場合は、配信チェックを無視して翻訳Botとして継続
			log.Printf("Twitch API設定がないため、配信チェックをスキップして継続します。")
		}

		// Twitchチャットへの起動メッセージ
		// バージョン情報を組み込んだ起動メッセージ
		startMsg := "⚙ bot起動 | " + formatStatus(BotVersion, UsedMsg, calculateRemainingWeeks())

		client.Say(joinChannelName, startMsg)
	})

	// --- 5. SYNDUALITY 15分前通知タイマー ---
	// 1分ごとに「15分後」の予定をチェックする
	go func() {
		// 次の「00秒」まで待機して同期（リテラシーへのこだわり）
		time.Sleep(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute)))

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if msg := bandainamco.GetSynSchedule(); msg != "" {
				client.Say(joinChannelName, msg)
			}
		}
	}()

	client.Join(joinChannelName)
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}
}

// --- 以下、各API用関数 (Facet/リンク対応済み) ---

// getStreamInfo は Twitch API を使用して指定チャンネルの配信情報を取得します。
// 入力：
//   - channelName: Twitchチャンネル名
//   - clientID: Twitch OAuth Client ID
//   - clientSecret: Twitch OAuth Client Secret
//
// 出力：
//   - TwitchStreamInfo 構造体へのポインタ（タイトルとゲーム名を含む）
//   - エラー（認証情報がない場合など）
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

// postToBluesky は指定されたテキストを Bluesky ATProtocol API 経由で投稿します。
// URL を含む場合は自動的に Facet（リンク機能）を適用して、
// チャット内のリンクを青色化します。
// 入力：
//   - text: 投稿する本文テキスト
//
// 出力：
//   - エラー（認証失敗やAPI呼び出し失敗など）
//
// 環境変数：
//   - BLUESKY_HANDLE: Bluesky アカウントハンドル（例: user.bsky.social）
//   - BLUESKY_APP_PASSWORD: Bluesky App Password
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

	// ステップ2: URL を検出して Facet 処理（リンクを青色化）
	// ATProtocol では URL は Facet という特別な構造で マークアップされます。
	var facets []map[string]interface{}
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	matches := urlRegex.FindAllStringIndex(text, -1)
	for _, m := range matches {
		facets = append(facets, map[string]interface{}{
			"index":    map[string]interface{}{"byteStart": m[0], "byteEnd": m[1]},
			"features": []map[string]interface{}{{"$type": "app.bsky.richtext.facet#link", "uri": text[m[0]:m[1]]}},
		})
	}

	// ステップ3: ATProtocol 経由で投稿リクエストを送信
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

// translateText は DeepL API を使用してテキストを翻訳します。
// デフォルトで "api-free.deepl.com" のフリープランエンドポイントを使用します。
// 入力：
//   - apiKey: DeepL API キー
//   - text: 翻訳対象テキスト
//   - targetLang: 翻訳言語コード（"JA" または "EN"）
//
// 出力：
//   - 翻訳済みテキスト（言語コード付き形式: "翻訳文 (元言語 > 目標言語)）
//   - エラー（API呼び出し失敗など）
//
// 注意：翻訳元言語が既に目標言語と同じ場合は空文字列を返します。
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
		// 言語が既に一致している場合は翻訳不要（空文字列を返す）
		if t["detected_source_language"].(string) == targetLang {
			return "", nil
		}
		// 翻訳済みテキストを「翻訳文 (元言語 > 目標言語)」形式で返す
		return fmt.Sprintf("%s (%s > %s)", t["text"].(string), t["detected_source_language"].(string), targetLang), nil
	}
	return "", nil
}

// getUsage は DeepL API の現在の使用状況を取得します。
// 入力：
//   - apiKey: DeepL API キー
//
// 出力：
//   - character_count: 今月のキャラクター使用数
//   - character_limit: 月当たりの使用可能なキャラクター数（フリープラン限定）
//   - エラー（API呼び出し失敗など）
func getUsage(apiKey string) (int, int, error) {
	resp, err := resty.New().R().SetHeader("Authorization", "DeepL-Auth-Key "+apiKey).Get("https://api-free.deepl.com/v2/usage")
	if err != nil {
		return 0, 0, err
	}
	var r map[string]interface{}
	json.Unmarshal(resp.Body(), &r)
	return int(r["character_count"].(float64)), int(r["character_limit"].(float64)), nil
}

// calculateRemainingWeeks は 今日から年末（12月31日）までの残り週数を計算します。
// 出力：
//   - 年末までの残り週数（小数点第一位を四捨五入）
func calculateRemainingWeeks() int {
	now := time.Now()
	endOfYear := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
	return (int(endOfYear.Sub(now).Hours()/24) + 6) / 7
}

func formatStatus(version, deepl string, weeks int) string {
	return fmt.Sprintf(
		"OK 🟢 | ver:%s | DeepL:%s | %d週",
		version,
		deepl,
		weeks,
	)
}

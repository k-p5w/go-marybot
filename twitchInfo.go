package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math" // 追加
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	authURL    = "https://id.twitch.tv/oauth2/token"
	baseURL    = "https://api.twitch.tv/helix/"
	gameURL    = "https://api.twitch.tv/helix/search/categories"
	topGameURL = "https://api.twitch.tv/helix/games/top"
)

// ゲーム名マップを読み込む関数
func loadGameNameMap(filePath string) (map[string]map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var gameNameMap map[string]map[string]string
	if err := json.NewDecoder(file).Decode(&gameNameMap); err != nil {
		return nil, err
	}

	return gameNameMap, nil
}

// アクセストークン取得
func getAccessToken(clientID, clientSecret string) (string, error) {
	data := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials", clientID, clientSecret)
	resp, err := http.Post(authURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if token, ok := result["access_token"].(string); ok {
		return token, nil
	}
	return "", fmt.Errorf("アクセストークンの取得に失敗しました: %v", result)
}

// ストリーマー情報を整形して表示する関数
func formatAndDisplayStreamers(data map[string]interface{}) {
	cnt := 0
	if streams, ok := data["data"].([]interface{}); ok {
		for _, stream := range streams {
			cnt += 1
			if streamInfo, ok := stream.(map[string]interface{}); ok {
				user_login := streamInfo["user_login"].(string)
				userName := streamInfo["user_name"].(string)
				viewerCount := streamInfo["viewer_count"].(float64)
				title := streamInfo["title"].(string)
				gameId := streamInfo["game_id"].(string)
				gameTitle := streamInfo["game_name"].(string)
				language := streamInfo["language"].(string)
				fmt.Printf("No%v.%v[%v:%v]a.配信者: %v/%s\nb.視聴者: %.0f\nc.タイトル: %s\n---\n", cnt, language, gameId, gameTitle, user_login, userName, viewerCount, title)
			}
		}
	} else {
		fmt.Println("No stream data available or unexpected format.")
	}
}

// カテゴリごとの配信者数を集計して表示する関数
func countStreamersByCategory(data map[string]interface{}, targetCategory string) {
	categoryCounts := make(map[string]bool) // ユニークな配信者を記録

	if streams, ok := data["data"].([]interface{}); ok {
		for _, stream := range streams {
			if streamInfo, ok := stream.(map[string]interface{}); ok {
				category, categoryOk := streamInfo["game_name"].(string)
				userName, userNameOk := streamInfo["user_name"].(string)

				if categoryOk && userNameOk && category == targetCategory {
					categoryCounts[userName] = true
				}
			}
		}
	}

	// 結果を表示
	fmt.Printf("%s: %d ユニーク配信者数\n", targetCategory, len(categoryCounts))
}

// ストリーマー情報取得
func getStreamers(clientID, category string, token string) error {

	url := fmt.Sprintf("%sstreams?first=100&game_id=%s", baseURL, category)
	fmt.Println("Request URL:", url) // デバッグ用にURLを出力

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 整形して表示
	formatAndDisplayStreamers(result)

	// カテゴリごとの配信者数を集計して表示
	countStreamersByCategory(result, category)

	return nil
}

// CSV出力用の関数
func writeToCSV(filePath string, records [][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

// --- 雑談・非ゲームカテゴリIDの除外リスト (IDで確実にする) ---
var excludedCategoryIDs = map[string]bool{
	"509672":     true, // Just Chatting (雑談)
	"26936":      true, // Pools, Hot Tubs, and Beaches
	"32053":      true, // ASMR
	"509658":     true, // 雑談
	"509660":     true, // 雑談
	"518203":     true, // 雑談
	"498592":     true, // 雑談
	"1669431183": true, // 雑談
	"509663":     true, // 雑談
	"417752":     true, // 雑談
	"509659":     true, // 雑談

	// 必要に応じて他の非ゲームカテゴリIDをここに追加してください
}

// --------------------------------------------------------

func getTotalViewersForTopGames(clientID, token string, gameNameFile string) error {
	// 出力用ディレクトリを作成（なければ作成）
	outputDir := "output"
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return fmt.Errorf("出力フォルダの作成に失敗: %v", err)
		}
	}

	// ゲーム名マップを読み込む
	gameNameMap, err := loadGameNameMap(gameNameFile)
	if err != nil {
		return fmt.Errorf("failed to load game name map: %v", err)
	}

	topGamesURL := "https://api.twitch.tv/helix/games/top?first=100"
	req, _ := http.NewRequest("GET", topGamesURL, nil)
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// ヘッダー定義 (配慮された用語を使用)
	header := []string{
		"ゲームID", "ゲーム名", "配信者数", "視聴者総数",
		"視聴者数（牽引層TOP3）",
		"視聴者数（主要層TOP10）",
		"視聴者数（裾野層）",
		"記録日時",
		"視聴者分布（全体/TOP3/TOP10/裾野層）",
		"視聴者割合（主要層 vs 裾野層）",
		"牽引層 集中度",
		"TOP3シェア率", // 変更
		"裾野層比率",    // 変更
	}

	// 1. アーカイブ用 (全100件)
	rawCsvRecords := [][]string{header}
	// 2. クリーンなランキング用 (雑談除外)
	gameRankingCsvRecords := [][]string{header}

	txtOutputCnt := 0
	totalViewersAll := 0
	totalViewersTop10 := 0

	// 各ゲームごとに集計
	for i, game := range result.Data {
		// 各ゲームの配信情報を取得
		streamsURL := fmt.Sprintf("https://api.twitch.tv/helix/streams?first=100&game_id=%s", game.ID)
		req, _ := http.NewRequest("GET", streamsURL, nil)
		req.Header.Set("Client-ID", clientID)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var streamsResult struct {
			Data []struct {
				ViewerCount int `json:"viewer_count"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&streamsResult)

		streamerCnt := 0
		gameViewersTop3 := 0
		gameViewersTop10 := 0
		gameViewersOther := 0
		gameViewersALL := 0

		// 分散率計算用の合計2乗（平方和）
		var sumSquares float64 = 0.0

		// 配信者ごとに視聴者数を集計
		for j, stream := range streamsResult.Data {
			streamerCnt++
			gameViewersALL += stream.ViewerCount

			// 平方和に追加
			v := float64(stream.ViewerCount)
			sumSquares += v * v

			switch {
			case j < 3:
				gameViewersTop3 += stream.ViewerCount
				gameViewersTop10 += stream.ViewerCount
			case j < 10:
				gameViewersTop10 += stream.ViewerCount
			default:
				gameViewersOther += stream.ViewerCount
			}
		}

		// ゲーム名（日本語があれば優先して使う）
		gameName := game.Name
		if gameNameMap != nil {
			if names, ok := gameNameMap[game.ID]; ok {
				if jaName, exists := names["ja"]; exists && jaName != "" {
					gameName = jaName
				}
			}
		}

		// 日時
		fileTime := time.Now().Format("20060102_1504")

		// 割合計算（既存）
		top3Ratio := 0.0
		top10Ratio := 0.0
		otherRatio := 0.0
		if gameViewersALL > 0 {
			top3Ratio = float64(gameViewersTop3) / float64(gameViewersALL) * 100
			top10Ratio = float64(gameViewersTop10) / float64(gameViewersALL) * 100
			otherRatio = float64(gameViewersOther) / float64(gameViewersALL) * 100
		}

		// --- 分散率（CV）計算 ---
		cvPercent := 0.0
		if streamerCnt > 0 {
			mean := float64(gameViewersALL) / float64(streamerCnt)
			variance := sumSquares/float64(streamerCnt) - mean*mean
			if variance < 0 {
				variance = 0 // 浮動小数誤差対策
			}
			stddev := math.Sqrt(variance)
			if mean > 0 {
				cvPercent = stddev / mean * 100
			}
		}

		// --- CSV用レコードを作成 ---
		record := []string{
			game.ID,
			gameName,
			fmt.Sprintf("%d", streamerCnt),
			fmt.Sprintf("%d", gameViewersALL),
			fmt.Sprintf("%d", gameViewersTop3),
			fmt.Sprintf("%d", gameViewersTop10),
			fmt.Sprintf("%d", gameViewersOther),
			fileTime,
			fmt.Sprintf("%d/%d/%d/%d", gameViewersALL, gameViewersTop3, gameViewersTop10, gameViewersOther),
			fmt.Sprintf("%.1f%% vs %.1f%%", top10Ratio, otherRatio), // 主要層 vs 裾野層
			fmt.Sprintf("%.1f%%", top3Ratio),                        // 牽引層 集中度
			fmt.Sprintf("%.1f%%", top3Ratio),                        // 集中率（TOP3%） — 既存 top3Ratio を再利用
			fmt.Sprintf("%.1f%%", cvPercent),                        // 分散率（CV%）
			fmt.Sprintf("%.1f%%", top3Ratio),                        // TOP3シェア率
			fmt.Sprintf("%.1f%%", otherRatio),                       // 裾野層比率
		}

		// 【①アーカイブ用】全レコードを記録
		rawCsvRecords = append(rawCsvRecords, record)

		// 【②ランキング用】雑談でなければ記録し、TXT出力の対象とする
		if !excludedCategoryIDs[game.ID] { // IDで判定する
			gameRankingCsvRecords = append(gameRankingCsvRecords, record)

			// --- テキストファイル出力のロジック ---
			if txtOutputCnt < 10 {
				txtOutputCnt++

				rankStr := fmt.Sprintf("%03d", txtOutputCnt)

				txtFileName := fmt.Sprintf("%s/%s_%s_%s.txt", outputDir, game.ID, fileTime, rankStr)

				streamerCntStr := fmt.Sprintf("%d名", streamerCnt)
				if streamerCnt >= 100 {
					streamerCntStr = "100名+"
				}
				txt := fmt.Sprintf(
					"%s\n=-=総配信者数=-=\n%s\n\n=-=総視聴者数=-=\n%s人\n\n==TOP3の視聴者合計==\n%.1f%%（%s人）\n",
					gameName,
					streamerCntStr,
					formatWithSpace(gameViewersALL),
					top3Ratio,
					formatWithSpace(gameViewersTop3),
				)
				if err := os.WriteFile(txtFileName, []byte(txt), 0644); err != nil {
					fmt.Printf("テキストファイルの書き込みに失敗: %v\n", err)
				}
			}
		}

		// 総視聴者数を加算
		totalViewersAll += gameViewersALL
		if i < 10 {
			totalViewersTop10 += gameViewersALL
		}
	}

	// --- CSVファイル出力（2種類に分ける） ---
	rawCsvFilePath := fmt.Sprintf("%s/archive_raw_%s.csv", outputDir, time.Now().Format("20060102_1504"))
	if err := writeToCSV(rawCsvFilePath, rawCsvRecords); err != nil {
		fmt.Printf("アーカイブCSVの書き込みに失敗しました: %v\n", err)
	} else {
		fmt.Printf("アーカイブCSVファイルにデータを書き込みました: %s\n", rawCsvFilePath)
	}

	gameRankingCsvFilePath := fmt.Sprintf("%s/game_ranking_%s.csv", outputDir, time.Now().Format("20060102_1504"))
	if err := writeToCSV(gameRankingCsvFilePath, gameRankingCsvRecords); err != nil {
		fmt.Printf("ランキング用CSVの書き込みに失敗しました: %v\n", err)
	} else {
		fmt.Printf("ランキング用CSVファイルにデータを書き込みました: %s\n", gameRankingCsvFilePath)
	}

	// --- サマリーファイル出力 ---
	summaryFile := fmt.Sprintf("%s/summary_%s.txt", outputDir, time.Now().Format("20060102_1504"))
	summary := fmt.Sprintf(
		"【Twitch人気100カテゴリ視聴者集計】\n"+
			"人気上位100カテゴリの総視聴者数:\n %d人\n"+
			"TOP10カテゴリの総視聴者数:\n %d人\n"+
			"TOP10カテゴリの割合:\n %.1f%%\n",
		totalViewersAll,
		totalViewersTop10,
		float64(totalViewersTop10)/float64(totalViewersAll)*100,
	)
	if err := os.WriteFile(summaryFile, []byte(summary), 0644); err != nil {
		fmt.Printf("サマリーファイルの書き込みに失敗: %v\n", err)
	} else {
		fmt.Printf("サマリーファイルを出力しました: %s\n", summaryFile)
	}

	return nil
}

// 人気カテゴリを取得する関数
func getTopGamesOrg(clientID, token string) error {
	url := fmt.Sprintf("%s?first=%d", topGameURL, 100)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 人気カテゴリを表示
	if data, ok := result["data"].([]interface{}); ok {
		fmt.Println("人気カテゴリ:")
		for _, item := range data {
			if game, ok := item.(map[string]interface{}); ok {
				id := game["id"].(string)
				name := game["name"].(string)
				fmt.Printf("Game ID: %s, Name: %s\n", id, name)
			}
		}
	} else {
		fmt.Println("No data found or unexpected format.")
	}

	return nil
}

// PopStreaming は、Twitchのストリーミング情報を取得して表示する関数です。
func PopStreaming(item *Config) {

	// アクセストークンを取得
	token, err := getAccessToken(item.ClientID, item.ClientSecret)
	if err != nil {
		fmt.Println("エラー:", err)
		return
	}

	// 人気カテゴリを取得
	fmt.Println("人気カテゴリを取得中...")
	err = getTotalViewersForTopGames(item.ClientID, token, "twitchGames.json")
	if err != nil {
		fmt.Println("人気カテゴリの取得に失敗しました:", err)
	}

	// 取得したいカテゴリを設定
	find := "final-fantasy-xi-online"
	getCategory(find, item.ClientID, token)

	category := "509658"
	category = "10229"
	err = getStreamers(item.ClientID, category, token)
	if err != nil {
		fmt.Println("エラー:", err)
	}
}

// GET https://api.twitch.tv/helix/search/categories?query=Minecraft
func getCategory(find, clientID, token string) error {

	url := fmt.Sprintf("%s?query=%s", gameURL, find)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// IDを抽出
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if category, ok := item.(map[string]interface{}); ok {
				if id, ok := category["id"].(string); ok {
					name := category["name"].(string)
					fmt.Printf("Category ID:%v[%v] \n", id, name)
				}
			}
		}
	} else {
		fmt.Println("No data found or unexpected format.")
	}

	return nil
}

// 4桁ごとにスペース区切りする関数
func formatWithSpace(n int) string {
	s := fmt.Sprintf("%d", n)
	var out []rune
	cnt := 0
	for i := len(s) - 1; i >= 0; i-- {
		out = append([]rune{rune(s[i])}, out...)
		cnt++
		if cnt%4 == 0 && i != 0 {
			out = append([]rune{' '}, out...)
		}
	}
	return string(out)
}

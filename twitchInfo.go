package main

import (
	"encoding/csv" // 追加
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time" // 追加
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
	return result["access_token"].(string), nil
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

// https://api.twitch.tv/helix/streams?game_name=Minecraft
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

func getTotalViewersForTopGames(clientID, token string, gameNameFile string) error {
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

	totalViewers := 0
	csvRecords := [][]string{} // CSV用のデータを格納するスライス

	// ヘッダー行を追加
	csvRecords = append(csvRecords, []string{
		"ゲームID",
		"ゲーム名",
		"配信者数",
		"視聴者総数",
		"視聴者数(TOP10)",
		"視聴者数(圏外)",
		"記録日時",
		"視聴者分布（全体/トップ/圏外）",
		"視聴者割合（TOP10 vs 圏外）",
	})

	for _, game := range result.Data {
		// 各ゲームの視聴者数を取得
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

		noTopStreamer := true
		streamerCnt := 0
		gameViewers := 0
		gameViewersALL := 0
		for _, stream := range streamsResult.Data {
			streamerCnt += 1
			if noTopStreamer && streamerCnt > 10 {
				gameViewers += stream.ViewerCount
			}
			gameViewersALL += stream.ViewerCount
		}
		totalViewers += gameViewers

		gameName := game.Name
		gameID := game.ID
		if names, ok := gameNameMap[game.ID]; ok {
			if jaName, exists := names["ja"]; exists {
				gameName = jaName
			}
		}

		currentTime := time.Now().Format("2006/01/02 15:04")
		gameViewersTOP := gameViewersALL - gameViewers
		viewerRatio := float64(gameViewers) / float64(gameViewersALL) * 100
		viewerTopRatio := (1 - float64(gameViewers)/float64(gameViewersALL)) * 100

		record := []string{
			gameID,
			gameName,
			fmt.Sprintf("%d", streamerCnt),
			fmt.Sprintf("%d", gameViewersALL),
			fmt.Sprintf("%d", gameViewersTOP),
			fmt.Sprintf("%d", gameViewers),
			currentTime,
			fmt.Sprintf("%d/%d/%d", gameViewersALL, gameViewersTOP, gameViewers),
			fmt.Sprintf("%.1f%% vs %.1f%%", viewerTopRatio, viewerRatio),
		}
		csvRecords = append(csvRecords, record)
	}

	// 日時付きファイル名で保存
	csvFilePath := fmt.Sprintf("output_%s.csv", time.Now().Format("20060102_1504"))
	if err := writeToCSV(csvFilePath, csvRecords); err != nil {
		fmt.Printf("CSVファイルの書き込みに失敗しました: %v\n", err)
	} else {
		fmt.Printf("CSVファイルにデータを書き込みました: %s\n", csvFilePath)
	}

	// 標準出力は不要なので削除
	// fmt.Printf("合計視聴者数: %d\n", totalViewers)
	return nil
}

// 人気カテゴリを取得する関数
func getTopGamesOrg(clientID, token string) error {
	url := fmt.Sprintf("%s?first=%d", topGameURL, 100) // firstパラメータを指定
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
// 引数には、設定情報を含むConfig構造体を渡します。
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

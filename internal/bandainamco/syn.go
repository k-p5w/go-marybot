package bandainamco

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// スケジュールデータ
var (
	sandyDunes      = map[time.Weekday][]string{time.Monday: {"02:00", "05:00", "07:30", "10:00", "12:30", "15:30", "18:00", "20:30", "23:00"}, time.Tuesday: {"02:00", "04:30", "07:00", "09:30", "12:30", "15:00", "17:30", "20:00", "23:00"}, time.Wednesday: {"01:30", "04:00", "06:30", "09:30", "12:00", "14:30", "17:00", "20:00", "22:30"}, time.Thursday: {"01:00", "03:30", "06:30", "09:00", "11:30", "14:00", "17:00", "19:30", "22:00"}, time.Friday: {"00:30", "03:30", "06:00", "08:30", "11:00", "14:00", "16:30", "19:00", "21:30"}, time.Saturday: {"00:30", "03:00", "05:30", "08:00", "11:00", "13:30", "16:00", "18:30", "21:30"}, time.Sunday: {"00:00", "02:30", "05:00", "08:00", "10:30", "13:00", "15:30", "18:30", "21:00", "23:30"}}
	empress         = map[time.Weekday][]string{time.Monday: {"04:30", "09:30", "15:00", "20:00"}, time.Tuesday: {"01:30", "06:30", "12:00", "17:00", "22:30"}, time.Wednesday: {"03:30", "09:00", "14:00", "19:30"}, time.Thursday: {"00:30", "06:00", "11:00", "16:30", "21:30"}, time.Friday: {"03:00", "08:00", "13:30", "18:30"}, time.Saturday: {"00:00", "05:00", "10:30", "15:30", "21:00"}, time.Sunday: {"02:00", "07:30", "12:30", "18:00", "23:00"}}
	forestNormal    = map[time.Weekday][]string{time.Monday: {"00:30", "03:00", "06:00", "08:30", "11:00", "13:30", "16:30", "19:00", "21:30"}, time.Tuesday: {"00:00", "03:00", "05:30", "08:00", "10:30", "13:30", "16:00", "18:30", "21:00"}, time.Wednesday: {"00:00", "02:30", "05:00", "07:30", "10:30", "13:00", "15:30", "18:00", "21:00", "23:30"}, time.Thursday: {"02:00", "04:30", "07:30", "10:00", "12:30", "15:00", "18:00", "20:30", "23:00"}, time.Friday: {"01:30", "04:30", "07:00", "09:30", "12:00", "15:00", "17:30", "20:00", "22:30"}, time.Saturday: {"01:30", "04:00", "06:30", "09:00", "12:00", "14:30", "17:00", "19:30", "22:30"}, time.Sunday: {"01:00", "03:30", "06:00", "09:00", "11:30", "14:00", "16:30", "19:30", "22:00"}}
	forestSunny     = map[time.Weekday][]string{time.Monday: {"01:30", "04:30", "07:00", "09:30", "12:00", "15:00", "17:30", "20:00", "22:30"}, time.Tuesday: {"01:30", "04:00", "06:30", "09:00", "12:00", "14:30", "17:00", "19:30", "22:30"}, time.Wednesday: {"01:00", "03:30", "06:00", "09:00", "11:30", "14:00", "16:30", "19:30", "22:00"}, time.Thursday: {"00:30", "03:00", "06:00", "08:30", "11:00", "13:30", "16:30", "19:00", "21:30"}, time.Friday: {"00:00", "03:00", "05:30", "08:00", "10:30", "13:30", "16:00", "18:30", "21:00"}, time.Saturday: {"00:00", "02:30", "05:00", "07:30", "10:30", "13:00", "15:30", "18:00", "21:00", "23:30"}, time.Sunday: {"02:00", "04:30", "07:30", "10:00", "12:30", "15:00", "18:00", "20:30", "23:00"}}
	predatorNormal  = map[time.Weekday][]string{time.Monday: {"00:00", "05:30", "10:30"}, time.Wednesday: {"15:00", "20:30"}, time.Thursday: {"01:30", "07:00"}, time.Saturday: {"11:30", "16:30", "22:00"}, time.Sunday: {"03:00", "08:30", "13:30", "19:00"}}
	predatorExtreme = map[time.Weekday][]string{time.Monday: {"02:30", "08:00"}, time.Wednesday: {"12:30", "17:30", "23:00"}, time.Thursday: {"04:00", "09:30"}, time.Saturday: {"14:00", "19:00"}, time.Sunday: {"00:30", "05:30", "11:00", "16:00", "21:30"}}

	amaziaDry = map[time.Weekday][]string{
		time.Monday:    {"03:00", "06:30", "10:00", "13:30", "17:00", "20:30"},
		time.Tuesday:   {"00:00", "03:30", "07:00", "10:30", "14:00", "17:30", "21:00"},
		time.Wednesday: {"00:30", "04:00", "07:30", "11:00", "14:30", "18:00", "21:30"},
		time.Thursday:  {"01:00", "04:30", "08:00", "11:30", "15:00", "18:30", "22:00"},
		time.Friday:    {"01:30", "05:00", "08:30", "12:00", "15:30", "19:00", "22:30"},
		time.Saturday:  {"02:00", "05:30", "09:00", "12:30", "16:00", "19:30", "23:00"},
		time.Sunday:    {"02:30", "06:00", "09:30", "13:00", "16:30", "20:00", "23:30"},
	}
	amaziaRain = map[time.Weekday][]string{
		time.Monday:    {"00:30", "04:00", "07:30", "11:00", "14:30", "18:00", "21:30"},
		time.Tuesday:   {"01:00", "04:30", "08:00", "11:30", "15:00", "18:30", "22:00"},
		time.Wednesday: {"01:30", "05:00", "08:30", "12:00", "15:30", "19:00", "22:30"},
		time.Thursday:  {"02:00", "05:30", "09:00", "12:30", "16:00", "19:30", "23:00"},
		time.Friday:    {"02:30", "06:00", "09:30", "13:00", "16:30", "20:00", "23:30"},
		time.Saturday:  {"03:00", "06:30", "10:00", "13:30", "17:00", "20:30"},
		time.Sunday:    {"00:00", "03:30", "07:00", "10:30", "14:00", "17:30", "21:00"},
	}
	amaziaNight = map[time.Weekday][]string{
		time.Monday:    {"01:30", "05:00", "08:30", "12:00", "15:30", "19:00", "22:30"},
		time.Tuesday:   {"02:00", "05:30", "09:00", "12:30", "16:00", "19:30", "23:00"},
		time.Wednesday: {"02:30", "06:00", "09:30", "13:00", "16:30", "20:00", "23:30"},
		time.Thursday:  {"03:00", "06:30", "10:00", "13:30", "17:00", "20:30"},
		time.Friday:    {"00:00", "03:30", "07:00", "10:30", "14:00", "17:30", "21:00"},
		time.Saturday:  {"00:30", "04:00", "07:30", "11:00", "14:30", "18:00", "21:30"},
		time.Sunday:    {"01:00", "04:30", "08:00", "11:30", "15:00", "18:30", "22:00"},
	}
)

// ゲーム名の定義（配信スタイルに合わせて切り替え可能）
const (
	GameNameFull  = "SYNDUALITY Echo of Ada"
	GameNameShort = "SYNDUALITY"
)

// エリア定義構造体
type areaDef struct {
	name     string
	schedule map[time.Weekday][]string
	duration int // 開放時間（分）
	areatype string
}

// 全スケジュールデータ（一箇所に集約）
func getAreas() []areaDef {
	return []areaDef{
		{"炎熱砂丘", sandyDunes, 60, "PvE専用エリア出現‼"},
		{"エンプレス/炎熱砂丘", empress, 30, "レイドボス出現!"},
		{"汚染森林", forestNormal, 60, "エンダーバスター!"},
		{"汚染森林・晴", forestSunny, 60, "メイガス拡張メモリを入手できるチャンス‼"},
		{"プレデター/汚染森林（深部）", predatorNormal, 30, "レイドボス出現!!"},
		{"プレデター(EX)", predatorExtreme, 30, "レイドボス出現!!!"},
		{"アメイジア東(乾期)", amaziaDry, 60, "アメイジア東・乾期開放!"},
		{"アメイジア東(雨期)", amaziaRain, 60, "アメイジア東・雨期開放!!"},
		{"アメイジア東(夜間)", amaziaNight, 90, "アメイジア東・夜間開放!!!"},
	}
}

// GetSynStatus is「現在開放中（残り時間）」と「1時間以内の予定」を返します
func GetSynStatus(isFullMode bool) string {
	loc, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(loc)
	areas := getAreas()

	var active, upcoming []string

	for _, a := range areas {
		w := now.Weekday()
		for _, startStr := range a.schedule[w] {
			t, _ := time.ParseInLocation("15:04", startStr, loc)
			start := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc)
			end := start.Add(time.Duration(a.duration) * time.Minute)

			// 1. 【開放中判定】 残り時間を計算
			if (now.Equal(start) || now.After(start)) && now.Before(end) {
				remaining := end.Sub(now).Minutes()
				active = append(active, fmt.Sprintf("%s@残り%d分", a.name, int(math.Ceil(remaining))))
			}

			// 2. 【1時間以内の予定判定】
			diff := start.Sub(now)
			if diff > 0 && diff <= 60*time.Minute {
				upcoming = append(upcoming, fmt.Sprintf("%s [%s]", a.name, startStr))
			}
		}
	}

	// ゲーム名の選択
	displayGameName := GameNameShort
	if isFullMode {
		displayGameName = GameNameFull
	}

	var res []string
	if len(active) > 0 {
		res = append(res, "🔓 OPEN >> "+strings.Join(active, ", "))
	} else {
		res = append(res, "🔒 開放エリアなし")
	}
	if len(upcoming) > 0 {
		res = append(res, "📅 NEXT >> "+strings.Join(upcoming, " / "))
	}

	//【SYNDUALITY Echo of Ada】 🔓 開放中: 炎熱砂丘@残り42分, 汚染森林(晴)@残り12分 | 📅 NEXT >> 汚染森林 [09:30]
	return fmt.Sprintf("【%s】 %s | %s", displayGameName, res[0], res[1])
}

// GetSynSchedule は15分後の自動通知用（既存の挙動を維持）
func GetSynSchedule() string {
	loc, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(loc)
	target := now.Add(15 * time.Minute).Format("15:04")
	w := now.Add(15 * time.Minute).Weekday()

	for _, a := range getAreas() {
		for _, s := range a.schedule[w] {
			if s == target {
				return fmt.Sprintf("⚠️ 【15分前】 %s 【%s】", a.name, a.areatype)
			}
		}
	}
	return ""
}

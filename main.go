package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// プレイヤー（マスター）の国家データ
type PlayerEmpire struct {
	Gold int `json:"gold"`
}

// 領地のデータ構造
type Territory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner"` // "player" or "enemy"
}

// 戦闘開始時にフロントへ送る部隊配置データ
type UnitData struct {
	ID   int     `json:"id"`
	Team string  `json:"team"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Type string  `json:"type"`
}

type BattleStartResponse struct {
	BattleID    string     `json:"battle_id"`
	TerritoryID string     `json:"territory_id"`
	Units       []UnitData `json:"units"`
}

// 雇用リクエスト用
type HireRequest struct {
	UnitType string `json:"unit_type"`
}

// 戦闘結果レシーブ用
type BattleResultRequest struct {
	BattleID    string `json:"battle_id"`
	TerritoryID string `json:"territory_id"`
	Result      string `json:"result"`
}

// PHPから届くロードデータの同期用
type SyncData struct {
	Gold           int    `json:"gold"`
	TerritoryOwner string `json:"territory_owner"`
}

// --- 簡易データベース（メモリ保持） ---
var CurrentPlayer = PlayerEmpire{Gold: 600}
var WorldMap = map[string]*Territory{
	"fort_front": {ID: "fort_front", Name: "最前線の砦", Owner: "enemy"},
}

func main() {
	// ① 静的ファイル（index.html）の配信
	http.Handle("/", http.FileServer(http.Dir(".")))

	// 実際のAPI処理を行うための独自のマルチプレクサー（ルーター）を作成
	apiMux := http.NewServeMux()

	// ② 現在のプレイヤーデータを返すAPI
	apiMux.HandleFunc("/api/player", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CurrentPlayer)
	})

	// ③ 領地データを返すAPI
	apiMux.HandleFunc("/api/map", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(WorldMap["fort_front"])
	})

	// ④ 雇用制限チェックAPI (POST)
	apiMux.HandleFunc("/api/hire", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req HireRequest
		json.NewDecoder(r.Body).Decode(&req)

		cost := 0
		if req.UnitType == "knight" {
			cost = 200
		} else if req.UnitType == "mage" {
			cost = 300
		}

		w.Header().Set("Content-Type", "application/json")
		if CurrentPlayer.Gold < cost {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("ゴールドが足りません！ (必要: %d / 所持: %d)", cost, CurrentPlayer.Gold),
			})
			return
		}

		CurrentPlayer.Gold -= cost
		fmt.Printf("【Goログ】雇用成功: %s (残り資金: %dG)\n", req.UnitType, CurrentPlayer.Gold)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"message":      fmt.Sprintf("%s を1体雇用しました！", req.UnitType),
			"current_gold": CurrentPlayer.Gold,
		})
	})

	// ⑤ 戦闘開始API (GET) - 敵軍マイルド（弱め）調整版
	apiMux.HandleFunc("/api/battle/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := BattleStartResponse{
			BattleID:    "B_MATCH_001",
			TerritoryID: "fort_front",
			Units: []UnitData{
				// 自軍（プレイヤー）：従来通り3体
				{ID: 1, Team: "player", X: 150, Y: 120, Type: "knight"},
				{ID: 2, Team: "player", X: 150, Y: 225, Type: "knight"},
				{ID: 3, Team: "player", X: 150, Y: 330, Type: "knight"},
				
				// 敵軍：数を5体から3体に減らし、布陣を緩くしました
				{ID: 6, Team: "enemy", X: 650, Y: 160, Type: "knight"}, // 敵前衛1
				{ID: 7, Team: "enemy", X: 650, Y: 290, Type: "knight"}, // 敵前衛2
				{ID: 9, Team: "enemy", X: 750, Y: 225, Type: "mage"},   // 敵後衛（孤立）
			},
		}
		json.NewEncoder(w).Encode(response)
	})

	// ⑥ 戦闘結果・領地書き換えAPI (POST)
	apiMux.HandleFunc("/api/battle/result", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var req BattleResultRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")

		if req.Result == "player_win" {
			if territory, exists := WorldMap[req.TerritoryID]; exists {
				territory.Owner = "player"
				fmt.Printf("【Goログ】プレイヤー勝利！「%s」の所有権を変更。\n", territory.Name)
				json.NewEncoder(w).Encode(map[string]string{
					"status":  "success",
					"message": fmt.Sprintf("勝利！ 領地「%s」を奪還しました！", territory.Name),
				})
				return
			}
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "戦闘終了（領地変化なし）"})
	})

	// ⑦ 【セーブ用】現在のGoの状態をフロントに引き渡すAPI (GET)
	apiMux.HandleFunc("/api/sync/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"gold":            CurrentPlayer.Gold,
			"territory_owner": WorldMap["fort_front"].Owner,
		})
	})

	// ⑧ 【ロード用】PHPから読み込んだデータでGoのメモリを書き換えるAPI (POST)
	apiMux.HandleFunc("/api/sync/load", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var data SyncData
		json.NewDecoder(r.Body).Decode(&data)

		CurrentPlayer.Gold = data.Gold
		WorldMap["fort_front"].Owner = data.TerritoryOwner

		fmt.Printf("【Goログ】セーブデータから同期完了 (資金: %dG, 領地所有者: %s)\n", data.Gold, data.TerritoryOwner)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// ⑨ 【リセット用】データを初期状態に戻すAPI (POST)
	apiMux.HandleFunc("/api/sync/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		// 初期値（1000G、敵軍領）に戻す
		CurrentPlayer.Gold = 1000
		WorldMap["fort_front"].Owner = "enemy"

		fmt.Println("【Goログ】ゲームが初期化されました (資金: 1000G, 領地: 敵軍領)")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "データを初期化しました"})
	})


	fmt.Println("=========================================================")
	fmt.Println(" ヴァーレントゥーガWeb（Go側コア）ポート:8080 で起動中...")
	fmt.Println("=========================================================")

	// 【最強のCORSラッパー】
	// 全てのリクエスト（静的ファイル配信含む）の「最外殻」で確実にCORSヘッダーを掴んで返す設定
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// OPTIONS（Preflight）は即座に返す
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 安全に前方一致を確認し、/api/ から始まっていれば apiMux へ
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			apiMux.ServeHTTP(w, r)
		} else {
			http.DefaultServeMux.ServeHTTP(w, r)
		}
	})

	// サーバーの起動命令は、この最後の1回だけにまとめます
	log.Fatal(http.ListenAndServe(":8080", finalHandler))
}
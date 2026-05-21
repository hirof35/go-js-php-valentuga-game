<?php
// save.php
header("Content-Type: application/json");
// 異なるポート(8080)からの通信（CORS）を許可する設定
header("Access-Control-Allow-Origin: *");
header("Access-Control-Allow-Headers: Content-Type");

$save_file = 'save_data.json';
$method = $_SERVER['REQUEST_METHOD'];

// ロード処理 (GET)
if ($method === 'GET') {
    if (file_exists($save_file)) {
        echo file_get_contents($save_file);
    } else {
        echo json_encode([
            "status" => "no_file",
            "gold" => 600,
            "territory_owner" => "enemy"
        ]);
    }
    exit;
}

// セーブ処理 (POST)
if ($method === 'POST') {
    $json_input = file_get_contents('php://input');
    $data = json_decode($json_input, true);
    
    if ($data) {
        file_put_contents($save_file, json_encode($data, JSON_PRETTY_PRINT));
        echo json_encode([
            "status" => "success",
            "message" => "データをサーバー（PHP）に保存しました！"
        ]);
    } else {
        http_response_code(400);
        echo json_encode(["status" => "error", "message" => "データが空、または不正です。"]);
    }
    exit;
}
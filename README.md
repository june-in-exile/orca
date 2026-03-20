# 🐋 Orca — Decentralized Video Infrastructure for Sui

## 簡介

Orca 是一個建立在 Walrus 之上的影片基礎設施，旨在簡化去中心化影片儲存與分發的流程。目前 Orca 正處於從本地處理轉向 **Walrus 原生架構** 的過渡階段（v2 遷移中），未來將整合 Seal 協議實現影片付費解鎖功能。

## 現有功能 (v2 Alpha)

目前版本專注於將 Walrus 作為核心儲存層：

- **非同步 Walrus 上傳**：接收 MP4 影片後，自動並行上傳至 Walrus Testnet。
- **Walrus 重新導向 (Redirect)**：透過 Orca Gateway 請求影片時，自動跳轉至 Walrus Aggregator 進行串流播放。
- **狀態追蹤**：提供 API 查詢影片上傳狀態與 Walrus `blobId`。
- **前端預覽**：內建簡單的 Web 介面進行上傳與播放測試。

---

## 快速開始

### 環境變數

```bash
ORCA_PORT=8080
ORCA_WALRUS_PUBLISHER_URL=https://publisher.walrus-testnet.walrus.space
ORCA_WALRUS_AGGREGATOR_URL=https://aggregator.walrus-testnet.walrus.space
ORCA_WALRUS_EPOCHS=1
ORCA_API_KEY=your_api_key
```

### 啟動服務

```bash
make run
```

---

## API 參考

### 1. 上傳影片
`POST /api/upload`
- **Body**: `multipart/form-data` (key: `video`)
- **Response**: 返回 `id` 與處理狀態。

### 2. 查詢狀態
`GET /api/status/{id}`
- **Response**: 返回影片的 `status`, `blob_id` 與 `blob_url`。

### 3. 串流播放
`GET /stream/{id}`
- **Behavior**: 暫時重新導向 (307 Redirect) 至 Walrus Aggregator。

---

## 發展路線 (Roadmap)

後續將逐步引入影片感知與付費功能：

### Phase 1: 影片感知與 HLS 切片 (進行中)
- 恢復 **FFmpeg** 轉碼邏輯。
- 將影片切分為 HLS (.m3u8 + .ts) 格式後分別存入 Walrus。

### Phase 2: Seal 整合 (計畫中)
- **部分加密策略**：影片前 N 秒維持明文，其餘片段使用 Seal 加密。
- **鏈上存取控制**：只有持有特定 Sui NFT 或支付 SUI 的用戶才能獲取 Seal 解密金鑰。

### Phase 3: Orca SDK
- 提供 `OrcaPlayer` 與 `OrcaUploader`，讓開發者只需幾行程式碼即可為 dApp 加入影片付費牆。

---

## 技術架構

```
[ 用戶端 ] 
    │
    ▼
[ Orca Gateway (Go) ] ─── (非同步) ───▶ [ Walrus Publisher ]
    │                                         │
    │ (Redirect)                              │ (Store Blobs)
    ▼                                         ▼
[ Walrus Aggregator ] ◀────────────────── [ Walrus Storage ]
```

---

## 開發者

Orca 專案致力於讓去中心化影片串流變得如同傳統 CDN 一樣流暢。

- **GitHub**: [github.com/anthropics/orca](https://github.com/anthropics/orca)
- **License**: MIT

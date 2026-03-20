# 🐋 Orca — Decentralized Video Paywall SDK for Sui

## 簡介

Orca 是建立在 Walrus + Seal 之上的影片付費解鎖 SDK。影片上傳時自動拆為預覽片段與完整版，預覽公開播放，完整版透過 Seal 加密存儲於 Walrus。觀眾付費後取得鏈上存取憑證，解密後即可觀看完整內容。開發者只需幾行程式碼即可為任何 dApp 加入影片付費牆功能。

## 現有功能 (v2 Alpha)

目前版本專注於將 Walrus 作為核心儲存層：

- **非同步 Walrus 上傳**：接收 MP4 影片後，自動並行上傳至 Walrus Testnet。
- **Walrus 重新導向 (Redirect)**：透過 Orca Gateway 請求影片時，自動跳轉至 Walrus Aggregator 進行串流播放。
- **狀態追蹤**：提供 API 查詢影片上傳狀態與 Walrus `blobId`。
- **前端預覽**：內建簡單的 Web 介面進行上傳與播放測試。

---

## 核心架構

### 付費解鎖流程

```
上傳時：
  MP4 → FFmpeg 擷取前 N 秒 → 預覽 MP4（明文）→ 存 Walrus (Blob A)
  MP4 → Seal 加密完整版    → 加密 MP4        → 存 Walrus (Blob B)
  → 鏈上建立 Video object（價格 + Blob IDs）

播放時：
  用戶進入 → 播放 Blob A（預覽，任何人可看）
  → 預覽結束 → 顯示付費牆
  → 付費 SUI → 鏈上 mint AccessPass
  → Seal 驗證 AccessPass → 解密 Blob B → 播放完整版
```

### 系統組件

```
[ 用戶端 ]
    │
    ▼
[ Orca SDK ]
    ├── @orca/uploader    擷取預覽 + Seal 加密 + 上傳 Walrus
    ├── @orca/sdk         付費 + 解密（與播放器解耦）
    └── @orca/contracts   Move 合約（定價 / 購買 / 驗證）
    │
    ├──── 寫入 ────▶ [ Walrus Publisher ] → [ Walrus Storage ]
    └──── 讀取 ────▶ [ Walrus Aggregator ] ← (Range Request 串流)
                         │
                   [ Sui 區塊鏈 ]
                   Video / AccessPass / Seal Policy
```

### 付費後的播放體驗

```
預覽模式：播放 Blob A（前 10 秒）
         → 預覽結束，顯示付費牆

付費後：  播放 Blob B（完整版，從頭開始）
         → 體驗類似電影預告片 → 購票 → 觀看正片
```

已購買的用戶重新進入時，SDK 偵測到鏈上 AccessPass，直接播放完整版。

---

## 鏈上合約設計

```move
module orca::paywall;

/// 影片資訊，creator 上傳時建立
public struct Video has key {
    id: UID,
    price: u64,                // 解鎖價格（MIST）
    creator: address,          // 收款地址
    preview_blob_id: String,   // 預覽版 Walrus Blob ID
    full_blob_id: String,      // 完整版 Walrus Blob ID（Seal 加密）
}

/// 購買憑證，付費後 mint，永久有效
public struct AccessPass has key {
    id: UID,
    video_id: ID,
}

/// 用戶付費 → mint AccessPass
public fun purchase(video: &Video, payment: Coin<SUI>, ctx: &mut TxContext): AccessPass;

/// Seal key server 驗證解密權限
entry fun seal_approve(id: vector<u8>, pass: &AccessPass, video: &Video);
```

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

- **Behavior**: 重新導向 (307 Redirect) 至 Walrus Aggregator。

---

## 發展路線 (Roadmap)

### Phase 1：Move 合約 + 雙 Blob 上傳（進行中）

- 部署 `orca::paywall` 合約至 Sui Testnet（Video / AccessPass / seal_approve）
- 上傳流程改為自動拆成兩個 blob：
  - Blob A：FFmpeg 擷取前 N 秒 → 明文存 Walrus
  - Blob B：完整 MP4 → Seal 加密後存 Walrus
- 鏈上建立 Video object，記錄兩個 Blob ID + 價格

**驗證方式：**

```bash
# 上傳影片，指定預覽 10 秒、價格 0.1 SUI
orca upload --file video.mp4 --preview 10 --price 100000000

# 預覽版可直接播放
curl $AGGREGATOR/v1/blobs/$PREVIEW_BLOB_ID -o preview.mp4

# 完整版是密文，無法直接播放
curl $AGGREGATOR/v1/blobs/$FULL_BLOB_ID -o encrypted.mp4
```

### Phase 2：付費解鎖 + Seal 解密

- 前端付費牆 UI：預覽播完 → 顯示價格 + 購買按鈕
- 錢包連接 → 呼叫 `purchase` → mint AccessPass
- Seal SDK 驗證 AccessPass → 取得解密金鑰 → 解密 Blob B
- 已購買用戶重新進入時自動偵測 AccessPass，直接播完整版

**驗證方式：**

- 未付費：只能看預覽，付費牆出現
- 付費後：完整版從頭播放
- 重新進入：已購買用戶直接看完整版

### Phase 3：Orca SDK 封裝

- `@orca/sdk`：付費 + 解密，與播放器完全解耦

  ```typescript
  import { Orca } from '@orca/sdk';

  const orca = new Orca({ network: 'mainnet' });

  // 檢查是否已購買
  const hasAccess = await orca.checkAccess(videoId, wallet);

  // 付費解鎖 → 拿到可播放的 URL
  const videoUrl = await orca.unlock(videoId, wallet);

  // 開發者愛用什麼播放器就用什麼
  videoElement.src = videoUrl;
  ```

- `@orca/uploader`：上傳 SDK（自動擷取預覽 + 加密 + 存 Walrus + 建鏈上 object）
- 文件 + 範例 dApp

### 之後逐步加入

- **Creator Dashboard**：上傳管理、收益統計、價格調整
- **批量訂閱**：一次付費解鎖 creator 所有影片
- **分潤機制**：嵌入者 / 推薦者可獲得銷售分成（鏈上自動分潤）
- **HLS 切片模式**：可選的進階模式，支援逐 segment 加密和多解析度

---

## 參考專案

| 專案 | 角色 | 備註 |
|---|---|---|
| [Walrus](https://docs.wal.app) | 去中心化儲存 + 原生串流 | Range Request 支援 MP4 串流播放 |
| [Seal](https://seal.mystenlabs.com) | 加密 + 存取控制 | Identity-based encryption + 鏈上 policy |
| [Seal Examples](https://github.com/MystenLabs/seal/tree/main/examples) | 參考實作 | Subscription pattern |
| [@mysten/seal](https://www.npmjs.com/package/@mysten/seal) | Seal TS SDK | 前端解密 |

---

## License

MIT

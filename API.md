# PayLock API Reference

本文檔提供 PayLock 後端服務的完整 API 規格，專為希望將「影片付費解鎖」功能整合進其 dApp 的開發者設計。

---

## 目錄

- [核心流程總覽](#核心流程總覽)
- [共用格式](#共用格式)
  - [影片狀態 (Status)](#影片狀態-status)
  - [Video 物件完整欄位](#video-物件完整欄位)
  - [錯誤回應格式](#錯誤回應格式)
- [API 端點](#api-端點)
  - [1. 上傳影片 — `POST /api/upload`](#1-上傳影片)
  - [2. 查詢影片狀態 — `GET /api/status/{id}`](#2-查詢影片狀態)
  - [3. 即時狀態追蹤 (SSE) — `GET /api/status/{id}/events`](#3-即時狀態追蹤-sse)
  - [4. 關聯鏈上物件 — `PUT /api/videos/{id}`](#4-關聯鏈上物件)
  - [5. 以鏈上物件 ID 查詢影片 — `GET /api/videos/by-object/{object_id}`](#5-以鏈上物件-id-查詢影片)
  - [6. 列出所有影片 — `GET /api/videos`](#6-列出所有影片)
  - [7. 刪除影片 — `DELETE /api/videos/{id}`](#7-刪除影片)
  - [8. 預覽串流 — `GET /stream/{id}/preview`](#8-預覽串流)
  - [9. 完整版串流 — `GET /stream/{id}/full`](#9-完整版串流)
  - [10. 系統配置 — `GET /api/config`](#10-系統配置)
  - [11. 手動重新索引 — `POST /api/reindex`](#11-手動重新索引)
- [付費解鎖整合指南](#付費解鎖整合指南)
  - [Step 1: 加密上傳（創作者端）](#step-1-加密上傳創作者端)
  - [Step 2: 鏈上發布（創作者端）](#step-2-鏈上發布創作者端)
  - [Step 3: 購買（觀眾端）](#step-3-購買觀眾端)
  - [Step 4: 解密播放（觀眾端）](#step-4-解密播放觀眾端)
  - [Move 合約參考](#move-合約參考-paylockgating)

---

## 核心流程總覽

### 免費影片

```
POST /api/upload (price=0)
    → 伺服器處理預覽 / 縮圖 / 完整版並上傳至 Walrus
    → GET /api/status/{id} 輪詢至 status=ready
    → GET /stream/{id} 播放
```

### 付費影片

```
POST /api/upload (price>0)
    → 伺服器處理預覽 / 縮圖並上傳至 Walrus
    → GET /api/status/{id}/events (SSE) 等待 status=ready
    → 前端以 Seal SDK 加密原始影片 → 上傳加密 Blob 至 Walrus → 取得 full_blob_id
    → 前端發 Sui 交易呼叫 gating::create_video (帶 price, preview_blob_id, full_blob_id, seal_namespace)
    → PUT /api/videos/{id} (回寫 sui_object_id + full_blob_id)
    → 購買者: purchase_and_transfer → seal_approve → Seal 解密 → 播放
```

---

## 共用格式

### 影片狀態 (Status)

| 值 | 說明 |
|----|------|
| `processing` | 上傳已接收，背景處理中 |
| `ready` | 預覽 / 完整版已上傳至 Walrus，可串流 |
| `failed` | 處理失敗，`error` 欄位包含原因 |

### Video 物件完整欄位

```json
{
  "id": "a1b2c3d4e5f6g7h8",
  "title": "My Video",
  "status": "ready",
  "price": 1000000000,
  "creator": "0xabc...",
  "thumbnail_blob_id": "...",
  "thumbnail_blob_url": "https://aggregator.../v1/blobs/...",
  "preview_blob_id": "...",
  "preview_blob_url": "https://aggregator.../v1/blobs/...",
  "full_blob_id": "...",
  "full_blob_url": "https://aggregator.../v1/blobs/...",
  "encrypted": true,
  "sui_object_id": "0x789...abc",
  "created_at": "2024-03-24T12:00:00Z",
  "error": ""
}
```

- `encrypted`: 付費影片為 `true`，表示 `full_blob_id` 指向 Seal 加密後的 Blob。
- `error`: 僅在 `status=failed` 時出現。
- 帶 `omitempty` 的欄位在值為空時不會出現在 response 中。

### 錯誤回應格式

所有錯誤統一回傳：

```json
{ "error": "描述訊息" }
```

---

## API 端點

### 1. 上傳影片

**`POST /api/upload`**

發起非同步上傳。伺服器驗證檔案後開始背景處理。

- **Content-Type**: `multipart/form-data`
- **大小上限**: 由 `PAYLOCK_MAX_FILE_SIZE_MB` 控制（預設 500 MB），超過回傳 `413`。
- **支援格式**: MP4 (`.mp4`), MOV (`.mov`), WebM (`.webm`), MKV (`.mkv`), AVI (`.avi`)。以 magic bytes 驗證，非副檔名。

| 參數 | 必填 | 說明 |
|------|------|------|
| `video` | 是 | 影片檔案 |
| `title` | 否 | 影片標題，未提供則自動產生 |
| `price` | 否 | 價格 (MIST, uint64)。`0` 或未提供 = 免費影片 |
| `creator` | 條件必填 | 創作者的 Sui 地址。`price > 0` 時必填 |

> **付費上傳限制**: `price > 0` 時，必須提供 `creator` 且伺服器必須啟用 FFmpeg (`PAYLOCK_ENABLE_FFMPEG=true`)，否則回傳 `400`。

**成功回應** (`202 Accepted`):

```json
{
  "id": "a1b2c3d4e5f6g7h8",
  "status": "processing"
}
```

**錯誤回應**:

| Status | 原因 |
|--------|------|
| `400` | 無法讀取檔案 / 格式不支援 / price 非正整數 / 付費上傳缺少 creator / 付費上傳但 FFmpeg 未啟用 |
| `413` | 檔案超過大小上限 |

---

### 2. 查詢影片狀態

**`GET /api/status/{id}`**

取得特定影片的完整 Metadata。

**成功回應** (`200 OK`): 回傳完整 Video 物件（見上方欄位定義）。

**錯誤回應**:

| Status | 原因 |
|--------|------|
| `400` | 缺少 video id |
| `404` | 影片不存在 |

---

### 3. 即時狀態追蹤 (SSE)

**`GET /api/status/{id}/events`**

Server-Sent Events 串流，每當影片狀態變更時推送完整 Video 物件。連線後立即推送一次目前狀態。適合用於上傳後等待處理完成。

```text
data: {"id":"...","status":"processing","title":"My Video","price":0,"created_at":"..."}

data: {"id":"...","status":"ready","preview_blob_id":"...","preview_blob_url":"...","full_blob_id":"...","full_blob_url":"...","created_at":"..."}
```

連線在 `status` 變為 `ready` 或 `failed` 後由伺服器關閉。

---

### 4. 關聯鏈上物件

**`PUT /api/videos/{id}`**

前端完成鏈上 `create_video` 交易後，將 Sui 物件 ID 與加密完整 Blob ID 寫回後端。

- **需要認證**: 須附帶 `X-Creator` header，值為影片的創作者 Sui 地址。

**Request Body** (`application/json`):

```json
{
  "sui_object_id": "0x789...abc",
  "full_blob_id": "blobId123"
}
```

| 欄位 | 必填 | 說明 |
|------|------|------|
| `sui_object_id` | 是 | 鏈上 Video shared object 的 ID |
| `full_blob_id` | 否 | 加密後完整 Blob 的 Walrus blob ID（付費影片應提供） |

**成功回應** (`200 OK`):

```json
{
  "status": "ok",
  "sui_object_id": "0x789...abc"
}
```

**錯誤回應**:

| Status | 原因 |
|--------|------|
| `400` | 缺少 video id / request body 無效 / `sui_object_id` 為空 |
| `403` | `X-Creator` 不符合影片的 creator |
| `404` | 影片不存在 |
| `409` | 該影片已綁定不同的 `sui_object_id` |

---

### 5. 以鏈上物件 ID 查詢影片

**`GET /api/videos/by-object/{object_id}`**

以 Sui 鏈上的 `sui_object_id` 查詢對應的影片 Metadata。

**成功回應** (`200 OK`): 回傳完整 Video 物件（見上方欄位定義）。

**錯誤回應**:

| Status | 原因            |
|--------|-----------------|
| `400`  | 缺少 object_id  |
| `404`  | 影片不存在      |

---

### 6. 列出所有影片

**`GET /api/videos`**

取得影片列表，按 `created_at` 降序排列（最新在前）。支援篩選與分頁。

**Query Parameters**:

| 參數 | 預設 | 說明 |
|------|------|------|
| `creator` | *(無)* | 按創作者 Sui 地址篩選 |
| `page` | `1` | 頁碼（從 1 開始） |
| `per_page` | `20` | 每頁筆數（上限 100） |

**成功回應** (`200 OK`):

```json
{
  "videos": [
    { "id": "...", "title": "...", "status": "ready", "price": 0, "thumbnail_blob_url": "...", "created_at": "..." },
    { "id": "...", "title": "...", "status": "ready", "price": 1000000000, "encrypted": true, "sui_object_id": "0x...", "created_at": "..." }
  ],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

---

### 7. 刪除影片

**`DELETE /api/videos/{id}`**

從後端 Metadata Store 中刪除該影片記錄。

- **需要認證**: 須附帶 `X-Creator` header，值為影片的創作者 Sui 地址。

> **注意**: 這不會刪除 Walrus 上的 Blob 或鏈上的 Video 物件。

**成功回應** (`200 OK`):

```json
{ "id": "...", "status": "deleted" }
```

**錯誤回應**:

| Status | 原因 |
|--------|------|
| `403` | `X-Creator` 不符合影片的 creator |
| `404` | 影片不存在 |

---

### 8. 預覽串流

**`GET /stream/{id}/preview`**

307 Redirect 至預覽版在 Walrus 上的公開 URL。任何人皆可存取。

```html
<video src="https://your-paylock-host/stream/{id}/preview"></video>
```

> **已棄用路徑**: `GET /stream/{id}` 仍可使用，會 307 Redirect 至 `/stream/{id}/preview` 並附帶 `Deprecation` header。預計 2026-09-23 移除。

**錯誤回應**:

| Status | 原因 |
|--------|------|
| `400` | 缺少 video id |
| `404` | 影片不存在 |
| `500` | 影片無 preview blob URL |
| `503` | 影片尚未就緒 (status != ready) |

---

### 9. 完整版串流

**`GET /stream/{id}/full`**

307 Redirect 至完整版 Blob URL。付費影片回傳的是加密後的 Blob，需前端 Seal 解密。

**錯誤回應**: 同預覽串流。

---

### 10. 系統配置

**`GET /api/config`**

取得後端環境配置。整合者應透過此 API 取得合約與 Walrus 端點，而非硬編碼。

**成功回應** (`200 OK`):

```json
{
  "gating_package_id": "0x...",
  "sui_network": "testnet",
  "walrus_publisher_url": "https://publisher.walrus-testnet.walrus.space",
  "walrus_aggregator_url": "https://aggregator.walrus-testnet.walrus.space"
}
```

---

### 11. 手動重新索引

**`POST /api/reindex`**

觸發從 Sui 鏈上重新掃描所有 Video 物件，將缺少的記錄補入本地 VideoStore。啟動時伺服器會自動執行一次，此端點供手動觸發。

- **需要認證**: 須附帶 `Authorization: Bearer <PAYLOCK_ADMIN_SECRET>` header。若未設定 `PAYLOCK_ADMIN_SECRET` 環境變數，此端點永遠回傳 `401`。

**成功回應** (`200 OK`):

```json
{
  "status": "ok",
  "chain_total": 120,
  "new_entries": 3
}
```

| 欄位           | 說明                         |
|----------------|------------------------------|
| `chain_total`  | 鏈上掃描到的 Video 物件總數  |
| `new_entries`  | 本次新增至本地 Store 的筆數  |

**錯誤回應**:

| Status | 原因                         |
|--------|------------------------------|
| `401`  | 缺少或無效的 admin secret    |
| `500`  | 鏈上掃描失敗                 |

---

## 付費解鎖整合指南

以下為外部開發者在前端實作付費影片完整流程的具體步驟。

### 前置準備

```bash
npm install @mysten/sui @mysten/seal
```

從 `GET /api/config` 取得 `gating_package_id` 和 Walrus 端點。

### Step 1: 加密上傳（創作者端）

伺服器處理完預覽後 (`status=ready`)，前端加密原始影片並上傳至 Walrus：

```js
import { SealClient } from '@mysten/seal';

// 1. 產生隨機 namespace (32 bytes) + nonce (5 bytes)
const namespace = crypto.getRandomValues(new Uint8Array(32));
const nonce = crypto.getRandomValues(new Uint8Array(5));
const id = toHex(new Uint8Array([...namespace, ...nonce]));

// 2. Seal 加密
const { encryptedObject } = await sealClient.encrypt({
  threshold: 1,
  packageId: gatingPackageId,
  id,
  data: new Uint8Array(fileData),
});

// 3. 上傳加密 Blob 至 Walrus
const res = await fetch(`${walrusPublisherUrl}/v1/blobs?epochs=5`, {
  method: 'PUT',
  body: encryptedObject,
});
const walrusData = await res.json();
const fullBlobId =
  walrusData.newlyCreated?.blobObject?.blobId ??
  walrusData.alreadyCertified?.blobId;
```

### Step 2: 鏈上發布（創作者端）

呼叫 `gating::create_video` 建立 Video shared object：

```js
import { Transaction } from '@mysten/sui/transactions';

const tx = new Transaction();
tx.moveCall({
  target: `${gatingPackageId}::gating::create_video`,
  arguments: [
    tx.pure.u64(priceMist),                        // price (MIST)
    tx.pure.string(previewBlobId),                  // 伺服器產生的 preview blob ID
    tx.pure.string(fullBlobId),                     // 加密後的 full blob ID
    tx.pure.vector('u8', Array.from(namespace)),    // 32-byte seal namespace
  ],
});
// 簽署並執行交易，從 transaction effects 取得新建 Video object ID
```

交易成功後，回寫後端：

```js
await fetch(`/api/videos/${videoId}`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    sui_object_id: suiObjectId,
    full_blob_id: fullBlobId,
  }),
});
```

### Step 3: 購買（觀眾端）

檢查是否已持有 AccessPass，若無則購買：

```js
// 查詢 AccessPass — 用 suiClient.getOwnedObjects 過濾 type
const { data } = await suiClient.getOwnedObjects({
  owner: buyerAddress,
  filter: {
    StructType: `${gatingPackageId}::gating::AccessPass`,
  },
  options: { showContent: true },
});

// 從結果中找到 video_id 匹配的 AccessPass
const accessPass = data.find(obj =>
  obj.data?.content?.fields?.video_id === videoSuiObjectId
);

if (!accessPass) {
  // 購買: 呼叫 purchase_and_transfer (自動退還多餘的 SUI)
  const tx = new Transaction();
  const [coin] = tx.splitCoins(tx.gas, [tx.pure.u64(video.price)]);
  tx.moveCall({
    target: `${gatingPackageId}::gating::purchase_and_transfer`,
    arguments: [
      tx.object(video.sui_object_id),  // &Video
      coin,                             // Coin<SUI>
    ],
  });
  // 簽署並執行
}
```

### Step 4: 解密播放（觀眾端）

取得 AccessPass 後，透過 Seal 解密影片：

```js
import { SealClient, SessionKey, EncryptedObject } from '@mysten/seal';

// 1. 建立 SessionKey (有效期 10 分鐘)
const sessionKey = await SessionKey.create({
  address: buyerAddress,
  packageId: gatingPackageId,
  ttlMin: 10,
  suiClient,
});

// 2. 簽署 personal message
const message = sessionKey.getPersonalMessage();
const { signature } = await wallet.signPersonalMessage({ message });
sessionKey.setPersonalMessageSignature(signature);

// 3. 下載加密 Blob
const encryptedRes = await fetch(video.full_blob_url);
const encryptedData = new Uint8Array(await encryptedRes.arrayBuffer());

// 4. 組裝 seal_approve 交易
const parsed = EncryptedObject.parse(encryptedData);
const tx = new Transaction();
tx.moveCall({
  target: `${gatingPackageId}::gating::seal_approve`,
  arguments: [
    tx.pure.vector('u8', fromHex(parsed.id)),  // Seal ID
    tx.object(accessPassId),                    // &AccessPass
    tx.object(video.sui_object_id),             // &Video
  ],
});
const txBytes = await tx.build({ client: suiClient, onlyTransactionKind: true });

// 5. Seal 解密
const decryptedBytes = await sealClient.decrypt({
  data: encryptedData,
  sessionKey,
  txBytes,
});

// 6. 播放
const blob = new Blob([decryptedBytes], { type: 'video/mp4' });
const url = URL.createObjectURL(blob);
videoElement.src = url;
videoElement.play();
```

### Move 合約參考 (`paylock::gating`)

| Function | 類型 | 說明 |
|----------|------|------|
| `create_video(price, preview_blob_id, full_blob_id, seal_namespace, ctx)` | public | 建立 Video shared object。price > 0 時 seal_namespace 不可為空 |
| `purchase_and_transfer(video, payment, ctx)` | entry | 購買影片，鑄造 AccessPass 並轉移給買家，自動退還多餘 SUI |
| `seal_approve(id, pass, video)` | entry | 驗證 AccessPass + Seal ID prefix，供 Seal key server 授權解密 |
| `seal_approve_owner(id, video, ctx)` | entry | 創作者自行解密（無需 AccessPass） |

**關鍵 Struct**:

```move
struct Video has key {
    id: UID,
    price: u64,
    creator: address,
    preview_blob_id: String,
    full_blob_id: String,
    seal_namespace: vector<u8>,
}

struct AccessPass has key, store {
    id: UID,
    video_id: ID,
}
```

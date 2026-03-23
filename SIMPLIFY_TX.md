# 重構：付費影片上傳從兩筆交易改為一筆

## 核心思路

用 client-generated 的隨機 32-byte namespace 取代 Video object ID 作為 Seal 加密前綴，打破循環依賴，讓整個流程只需一筆交易。

### 現行流程（2 筆交易）

```
TX1: create_video(price, '', '')  →  拿到 Video object ID
[用 object ID 做 Seal 加密 → 上傳 Walrus → 拿到 blob IDs]
TX2: update_preview_blob_id + update_full_blob_id
```

### 目標流程（1 筆交易）

```
[前端產生 random namespace → 用 namespace 做 Seal 加密 → 上傳 Walrus → 拿到 blob IDs]
TX: create_video(price, preview_blob_id, full_blob_id, seal_namespace)
```

---

## 實作順序

| 順序 | 工作項目 | 相依性 |
|------|---------|--------|
| 1 | Move 合約修改 + 測試 | 無 |
| 2 | 部署新合約、取得新 package ID | 依賴 1 |
| 3 | wallet.js 重構 | 依賴 2 |
| 4 | videos-view.js 上傳流程調整 | 依賴 3 |
| 5 | 後端清理（刪除 endpoint） | 依賴 3 |
| 6 | 環境變數更新 + 整合測試 | 依賴 2-5 |

---

## Step 1：Move 合約修改

**檔案：`contracts/paylock/sources/paywall.move`**

### 1a. Video struct 新增欄位

```move
public struct Video has key {
    id: UID,
    price: u64,
    creator: address,
    preview_blob_id: String,
    full_blob_id: String,
    seal_namespace: vector<u8>,  // 新增
}
```

### 1b. `create_video` 新增參數

```move
public fun create_video(
    price: u64,
    preview_blob_id: String,
    full_blob_id: String,
    seal_namespace: vector<u8>,  // 新增
    ctx: &mut TxContext,
)
```

加入斷言：若 `price > 0` 則 `vector::length(&seal_namespace) > 0`（防止空 namespace 讓任何 seal ID 通過驗證）。

### 1c. `seal_approve` 改用 `seal_namespace`

現行用 `object::id_bytes(video)` 做前綴比對，改為用 `video.seal_namespace`。

### 1d. 刪除兩個函式

- `update_preview_blob_id`
- `update_full_blob_id`

### 1e. 新增 accessor

```move
public fun video_seal_namespace(video: &Video): &vector<u8> { &video.seal_namespace }
```

---

## Step 2：合約測試更新

**檔案：`contracts/paylock/tests/paywall_tests.move`**

- 所有 `create_video` 呼叫新增第四個參數 `seal_namespace`（用固定 32-byte 值）
- 刪除 `test_update_full_blob_id` 和 `test_update_full_blob_id_not_creator` 測試
- `test_seal_approve_valid`：seal ID 的 prefix 改用 `seal_namespace`（而非 object ID）
- 新增測試：namespace 不匹配時 `seal_approve` 應 abort
- 新增測試：免費影片（`price=0`）的 `seal_namespace` 可以是空 vector

跑 `sui move test` 確認全通過後部署：

```bash
sui client publish --gas-budget 100000000
```

---

## Step 3：wallet.js 重構

**檔案：`cmd/paylock/web/wallet.js`**

### 3a. `encryptAndPublish` 改用隨機 namespace

```js
// 現行：先 createVideoOnChain 拿 object ID 再加密
// 改為：直接產生隨機 namespace
export async function encryptAndPublish(videoId, fileData, price, onProgress) {
  const namespace = crypto.getRandomValues(new Uint8Array(32));
  const nonce = crypto.getRandomValues(new Uint8Array(5));
  const id = toHex(new Uint8Array([...namespace, ...nonce]));

  if (onProgress) onProgress('encrypt');
  const { encryptedObject: encryptedBytes } = await sealClient.encrypt({
    threshold: 1, packageId: paywallPackageId, id,
    data: new Uint8Array(fileData),
  });

  if (onProgress) onProgress('walrus');
  // ... Walrus 上傳邏輯不變 ...

  return { namespace, fullBlobId };  // 回傳 namespace（不再回傳 suiObjectId）
}
```

### 3b. `createVideoOnChain` 新增 `sealNamespace` 參數

```js
export async function createVideoOnChain(videoId, price, previewBlobId, fullBlobId, sealNamespace) {
  // ...
  tx.moveCall({
    target: paywallPackageId + '::paywall::create_video',
    arguments: [
      tx.pure.u64(price),
      tx.pure.string(previewBlobId),
      tx.pure.string(fullBlobId),
      tx.pure.vector('u8', sealNamespace),  // 新增
    ],
  });
  // ... 簽名、等待、PUT /sui-object 不變
}
```

### 3c. 刪除 `updateBlobIds` 函式

整個函式不再需要。

---

## Step 4：videos-view.js 上傳流程調整

**檔案：`cmd/paylock/web/videos-view.js`**

`confirmUpload` 的付費影片分支改為：

```js
// 並行：server 上傳 preview + browser 加密上傳 full
const [video, encResult] = await Promise.all([
  pollUntilReady(data.id).then((v) => {
    uploadState.value = { ...uploadState.value, previewDone: true };
    return v;
  }),
  mod.encryptAndPublish(data.id, fileArrayBuffer, priceMist, (browserStep) => {
    uploadState.value = { ...uploadState.value, browserStep };
  }),
]);

// 一筆交易搞定（不再有 TX2）
uploadState.value = { ...uploadState.value, step: 'onchain', text: 'Creating video on-chain...' };
await mod.createVideoOnChain(
  data.id, priceMist, video.preview_blob_id, encResult.fullBlobId, encResult.namespace,
);
```

步驟文字 "Publishing blob IDs on-chain" → "Creating video on-chain"。

---

## Step 5：後端清理

### 刪除 `PUT /api/videos/{id}/full-blob` endpoint

- `cmd/paylock/main.go`：移除路由註冊
- `internal/handler/set_full_blob.go`：整個檔案刪除
- `internal/model/video.go`：刪除 `SetFullBlob` 方法

`PUT /api/videos/{id}/sui-object` 保留（仍需存 object ID 給購買流程用）。

跑 `make test` 確認後端無殘留引用。

---

## Step 6：向後相容性

PayLock 在 testnet 上，部署新合約後 package ID 會變。舊 Video 物件無法被新合約操作（type mismatch），直接視為廢棄。

- 更新 `PAYLOCK_PAYWALL_PACKAGE_ID` 環境變數
- 可選：清除 `data/videos.json` 中的舊記錄

---

## 注意事項

1. **空 namespace 防護**：合約中 `price > 0` 時必須 assert `seal_namespace` 非空，否則空 prefix 會讓任何 seal ID 通過
2. **`decryptVideo`（wallet.js）不需改**：seal ID 是從 EncryptedObject 裡 parse 出來的，合約端已改用 `seal_namespace` 比對，前端解密邏輯不受影響
3. **encrypt 步驟會更快**：不再有 TX1 的等待時間，`encryptAndPublish` 直接開始加密

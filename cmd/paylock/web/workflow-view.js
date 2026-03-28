import { html, useEffect, useRef } from './lib.js';
import mermaid from 'https://esm.sh/mermaid@11/dist/mermaid.esm.min.mjs';

const freeVideoFlow = `sequenceDiagram
    participant C as Client
    participant S as PayLock Server
    participant W as Walrus

    C->>S: POST /api/upload (video file)
    S->>S: Validate magic bytes & size
    S-->>C: 202 { id, status: "processing" }
    alt FFmpeg enabled
        S->>S: Extract preview clip + thumbnail
    end
    S->>W: Store preview blob
    S->>W: Store full blob
    S->>W: Store thumbnail (if generated)
    S-->>S: Status → ready
    C->>S: GET /api/videos/{id}
    S-->>C: 200 { preview_blob_url, full_blob_url, ... }`;

const paidVideoFlow = `sequenceDiagram
    participant C as Client
    participant S as PayLock Server
    participant W as Walrus
    participant Seal as Seal Service
    participant Sui as Sui Chain

    C->>S: POST /api/upload (video, price > 0)
    S->>S: Validate + FFmpeg extract preview & thumbnail
    S->>W: Store preview blob
    S->>W: Store thumbnail
    S-->>C: 202 { id, status: "processing", preview_blob_id }
    C->>Seal: sealClient.encrypt(full video)
    Seal-->>C: Encrypted full blob
    C->>W: Upload encrypted full blob
    C->>Sui: gating::create_video (blob IDs, namespace)
    C->>S: PATCH /api/videos/{id}/link (sui_object_id, full_blob_id)
    S->>S: Link on-chain object → local record
    S-->>S: Status → ready`;

const purchaseFlow = `sequenceDiagram
    participant B as Buyer
    participant Sui as Sui Chain
    participant Seal as Seal Service
    participant W as Walrus

    B->>Sui: purchase_and_transfer (pay SUI)
    Sui-->>B: AccessPass minted
    B->>B: Create Seal SessionKey
    B->>Sui: seal_approve (AccessPass)
    B->>Seal: sealClient.decrypt(encrypted blob, session)
    Seal->>Sui: Verify seal_approve (AccessPass)
    Sui-->>Seal: Approval confirmed
    Seal-->>B: Decryption keys
    B->>W: Fetch encrypted full blob
    B->>B: Decrypt & play video`;

export function WorkflowView() {
  const containerRef = useRef(null);

  useEffect(() => {
    mermaid.initialize({ startOnLoad: false, theme: 'dark' });
    if (containerRef.current) {
      mermaid.run({
        nodes: containerRef.current.querySelectorAll('.mermaid'),
      });
    }
  }, []);

  return html`
    <div class="view active" ref=${containerRef}>
      <div class="header-action">
        <h2>PayLock Workflow</h2>
      </div>

      <div class="workflow-section">
        <h3>1. Free Videos (price = 0)</h3>
        <p>Client uploads a video, server extracts preview/thumbnail and stores on Walrus.</p>
        <div class="mermaid">${freeVideoFlow}</div>
      </div>

      <div class="workflow-section">
        <h3>2. Paid Videos (price > 0)</h3>
        <p>Client uploads via Seal and registers the video on Sui.</p>
        <div class="mermaid">${paidVideoFlow}</div>
      </div>

      <div class="workflow-section">
        <h3>3. Purchase Flow</h3>
        <p>Buyer pays, gets AccessPass, decrypts the video using Seal.</p>
        <div class="mermaid">${purchaseFlow}</div>
      </div>
    </div>
  `;
}

import { html, useState, useEffect } from './lib.js';
import { walletState, navGeneration } from './state.js';
import { UploadSection } from './upload-section.js';
import { VideoCard, deleteVideo } from './video-card.js';

export function MyVideosView() {
  const [videos, setVideos] = useState([]);
  const [loadError, setLoadError] = useState(null);
  const wallet = walletState.value;
  const generation = navGeneration.value;

  useEffect(() => {
    if (!wallet.connected || !wallet.address) return;
    let cancelled = false;

    fetch('/api/videos?creator=' + encodeURIComponent(wallet.address))
      .then((res) => {
        if (!res.ok) throw new Error('Failed to load');
        return res.json();
      })
      .then((data) => { if (!cancelled) { setVideos(data.videos || []); setLoadError(null); } })
      .catch(() => { if (!cancelled) setLoadError('Cannot connect to server.'); });

    return () => { cancelled = true; };
  }, [generation, wallet.address, wallet.connected]);

  if (!wallet.connected) {
    return html`
      <div class="view active">
        <div class="empty-state">
          <p>Connect your wallet to manage your videos.</p>
        </div>
      </div>
    `;
  }

  const refreshVideos = () => {
    fetch('/api/videos?creator=' + encodeURIComponent(wallet.address))
      .then((res) => res.json())
      .then((data) => setVideos(data.videos || []))
      .catch(() => {});
  };

  return html`
    <div class="view active">
      <${UploadSection} />

      <h2 style="margin-bottom: 1rem;">My Videos</h2>
      <div>
        ${loadError
          ? html`<div class="empty-state"><p>${loadError}</p></div>`
          : videos.length === 0
            ? html`<div class="empty-state"><p>No videos yet. Drag and drop a file or click Select Video.</p></div>`
            : html`
                <div class="video-grid">
                  ${videos.map((v) => html`<${VideoCard} key=${v.id} video=${v} showDelete=${true} onDeleted=${refreshVideos} />`)}
                </div>
              `}
      </div>
    </div>
  `;
}

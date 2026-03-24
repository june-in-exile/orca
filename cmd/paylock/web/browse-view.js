import { html, useState, useEffect } from './lib.js';
import { navGeneration } from './state.js';
import { VideoCard } from './video-card.js';

export function BrowseView() {
  const [videos, setVideos] = useState([]);
  const [loadError, setLoadError] = useState(null);
  const generation = navGeneration.value;

  useEffect(() => {
    let cancelled = false;

    fetch('/api/videos')
      .then((res) => {
        if (!res.ok) throw new Error('Failed to load');
        return res.json();
      })
      .then((data) => { if (!cancelled) { setVideos(data.videos || []); setLoadError(null); } })
      .catch(() => { if (!cancelled) setLoadError('Cannot connect to server.'); });

    return () => { cancelled = true; };
  }, [generation]);

  return html`
    <div class="view active">
      <h2 style="margin-bottom: 1rem;">Browse Videos</h2>
      <div>
        ${loadError
          ? html`<div class="empty-state"><p>${loadError}</p></div>`
          : videos.length === 0
            ? html`<div class="empty-state"><p>No videos available yet.</p></div>`
            : html`
                <div class="video-grid">
                  ${videos.map((v) => html`<${VideoCard} key=${v.id} video=${v} showDelete=${false} />`)}
                </div>
              `}
      </div>
    </div>
  `;
}

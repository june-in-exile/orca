import { html, render, useState, useEffect } from './lib.js';
import {
  currentView, walletState, toastState,
  navigate, loadWallet, stageNewFile,
} from './state.js';
import { VideosView } from './videos-view.js';
import { PlayerView } from './player-view.js';

function Header() {
  const wallet = walletState.value;
  const shortAddr = wallet.address
    ? wallet.address.slice(0, 6) + '...' + wallet.address.slice(-4)
    : '';

  async function handleWalletClick() {
    const mod = await loadWallet();
    mod.connectWallet();
  }

  const btnLabel = !wallet.available
    ? (wallet.error || 'Wallet Unavailable')
    : wallet.error
      ? wallet.error
      : wallet.connected ? 'Disconnect' : 'Connect Wallet';

  return html`
    <header>
      <div class="logo" onclick=${() => navigate('videos')}>
        <span class="logo-text">Pay</span><span class="logo-box">Lock</span>
      </div>
      <div class="wallet-area">
        ${wallet.connected && html`
          <div class="wallet-info" style="display:flex;">
            <span class="wallet-dot"></span>
            <span class="wallet-balance">${wallet.balance || '0 SUI'}</span>
            <span class="wallet-addr">${shortAddr}</span>
          </div>
        `}
        <button
          class=${wallet.connected ? 'wallet-btn connected' : 'wallet-btn connect'}
          disabled=${!wallet.available}
          style=${!wallet.available ? 'opacity:0.5;cursor:not-allowed' : ''}
          onclick=${handleWalletClick}
        >
          ${btnLabel}
        </button>
      </div>
    </header>
  `;
}

function Toast() {
  const t = toastState.value;
  if (!t) return html`<div class="toast"></div>`;
  return html`
    <div class=${`toast ${t.type}${t.visible ? ' visible' : ''}`}>
      ${t.message}
    </div>
  `;
}

function DragOverlay({ active }) {
  return html`
    <div class=${`upload-overlay${active ? ' active' : ''}`}>
      <div class="upload-overlay-text">Drop MP4 to select</div>
    </div>
  `;
}

function App() {
  const view = currentView.value;
  const [dragging, setDragging] = useState(false);

  // Global drag-and-drop
  useEffect(() => {
    let dragCounter = 0;

    function onDragEnter(e) { e.preventDefault(); dragCounter++; setDragging(true); }
    function onDragLeave(e) { e.preventDefault(); dragCounter--; if (dragCounter <= 0) { dragCounter = 0; setDragging(false); } }
    function onDragOver(e) { e.preventDefault(); }
    function onDrop(e) {
      e.preventDefault();
      dragCounter = 0;
      setDragging(false);
      if (e.dataTransfer.files.length > 0) stageNewFile(e.dataTransfer.files[0]);
    }

    window.addEventListener('dragenter', onDragEnter);
    window.addEventListener('dragleave', onDragLeave);
    window.addEventListener('dragover', onDragOver);
    window.addEventListener('drop', onDrop);

    return () => {
      window.removeEventListener('dragenter', onDragEnter);
      window.removeEventListener('dragleave', onDragLeave);
      window.removeEventListener('dragover', onDragOver);
      window.removeEventListener('drop', onDrop);
    };
  }, []);

  // Client-side router
  useEffect(() => {
    function handleRoute() {
      const path = window.location.pathname;
      if (path.startsWith('/play/')) {
        navigate('player', { id: path.slice(6) }, false);
      } else {
        navigate('videos', {}, false);
      }
    }

    window.addEventListener('popstate', handleRoute);
    handleRoute();
    return () => window.removeEventListener('popstate', handleRoute);
  }, []);

  // Eagerly start wallet loading (non-blocking)
  useEffect(() => { loadWallet().catch(() => {}); }, []);

  return html`
    <${Header} />
    <${DragOverlay} active=${dragging} />
    <main>
      ${view === 'videos' && html`<${VideosView} />`}
      ${view === 'player' && html`<${PlayerView} />`}
    </main>
    <${Toast} />
  `;
}

render(html`<${App} />`, document.getElementById('app'));

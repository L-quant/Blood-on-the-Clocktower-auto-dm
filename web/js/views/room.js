// ============================================
// 房间设置页面 (选择剧本、设定人数、选座位)
// ============================================

import { store } from '../store.js';
import { wsSendCommand, refreshState } from '../api.js';
import { toast } from '../utils.js';
import { navigate } from '../app.js';
import { EDITIONS, getComposition } from '../roles.js';

let stateUnsub = null;
let settingsDebounce = null;

export function renderRoom() {
  const roomId = store.get('roomId');
  const shortCode = roomId ? roomId.substring(0, 8).toUpperCase() : '---';
  const isHost = store.get('isHost');

  // Get settings from gameState if available, else use store defaults
  const gs = store.get('gameState');
  const edition = (gs && gs.edition) || store.get('selectedEdition') || 'tb';
  const playerCount = (gs && gs.max_players) || store.get('selectedPlayerCount') || 7;
  const comp = getComposition(playerCount);

  return `
    <div class="room-page">
      <header class="room-header">
        <span class="room-title">🩸 房间设置</span>
        <div class="room-code">
          <span>邀请码：</span>
          <span class="code" id="room-code-text">${shortCode}</span>
          <button class="copy-btn" id="copy-code-btn" title="复制完整邀请码">📋 复制</button>
        </div>
        <div class="room-actions">
          <button class="btn btn-ghost btn-sm" id="back-home-btn">返回</button>
        </div>
      </header>

      <div class="room-content">
        <!-- Edition Selection -->
        <div class="edition-section">
          <h3>📖 选择剧本</h3>
          ${isHost ? `
          <div class="edition-cards" id="edition-cards">
            ${EDITIONS.map(ed => `
              <div class="edition-card ${edition === ed.id ? 'selected' : ''}" data-edition="${ed.id}">
                <div class="edition-icon">${ed.icon}</div>
                <div class="edition-name">${ed.name}</div>
                <div class="edition-desc">${ed.desc}</div>
              </div>
            `).join('')}
          </div>
          ` : `
          <div class="edition-cards">
            ${EDITIONS.filter(ed => ed.id === edition).map(ed => `
              <div class="edition-card selected">
                <div class="edition-icon">${ed.icon}</div>
                <div class="edition-name">${ed.name}</div>
                <div class="edition-desc">${ed.desc}</div>
              </div>
            `).join('')}
            <div style="color:var(--text-secondary); font-size:0.85rem; margin-top:8px;">
              由房主选择剧本
            </div>
          </div>
          `}
        </div>

        <!-- Player Count -->
        <div class="player-count-section">
          <h3>👥 设定人数</h3>
          ${isHost ? `
          <div class="slider-container">
            <span style="color:var(--text-secondary)">5</span>
            <input type="range" min="5" max="15" value="${playerCount}" id="player-count-slider">
            <span style="color:var(--text-secondary)">15</span>
            <span class="slider-value" id="player-count-value">${playerCount}</span>
          </div>
          ` : `
          <div class="slider-container">
            <span class="slider-value" style="font-size:1.5rem;">${playerCount} 人</span>
            <span style="color:var(--text-secondary); font-size:0.85rem; margin-left:12px;">由房主设定</span>
          </div>
          `}
          <div class="composition-info" id="composition-info">
            <div class="comp-item"><span class="comp-label">村民</span><span class="comp-value">${comp.townsfolk}</span></div>
            <div class="comp-item"><span class="comp-label">外来者</span><span class="comp-value">${comp.outsider}</span></div>
            <div class="comp-item"><span class="comp-label">爪牙</span><span class="comp-value">${comp.minion}</span></div>
            <div class="comp-item"><span class="comp-label">恶魔</span><span class="comp-value">${comp.demon}</span></div>
          </div>
        </div>

        <!-- Seats -->
        <div class="seats-section">
          <h3>💺 选择座位</h3>
          <div class="seats-grid" id="seats-grid">
            ${renderSeats(playerCount)}
          </div>
        </div>

        <!-- Start Button -->
        <div style="text-align:center; padding: 20px;">
          ${isHost ? `
          <button class="btn btn-primary btn-lg" id="start-game-btn" disabled>
            🎮 开始游戏
          </button>
          ` : `
          <button class="btn btn-primary btn-lg" id="start-game-btn" disabled>
            ⏳ 等待房主开始
          </button>
          `}
          <div style="color:var(--text-secondary); font-size:0.85rem; margin-top:8px;" id="start-hint">
            等待所有玩家入座...
          </div>
        </div>
      </div>
    </div>
  `;
}

function renderSeats(count) {
  const gs = store.get('gameState');
  const players = gs ? gs.players || {} : {};
  const myId = store.get('userId');
  let html = '';

  for (let i = 1; i <= count; i++) {
    let occupant = null;
    for (const p of Object.values(players)) {
      if (p.seat_number === i && !p.is_dm) {
        occupant = p;
        break;
      }
    }

    const isMine = occupant && occupant.user_id === myId;
    const stateClass = occupant ? (isMine ? 'occupied mine' : 'occupied') : 'empty';

    html += `
      <div class="seat-slot ${stateClass}" data-seat="${i}">
        <span class="seat-number">#${i}</span>
        ${occupant
          ? `<div class="seat-avatar">👤</div>
             <div class="seat-name">${occupant.name || '玩家'}</div>`
          : `<button class="seat-sit-btn" data-claim-seat="${i}">坐下</button>`
        }
      </div>
    `;
  }
  return html;
}

function sendSettingsToServer(edition, maxPlayers) {
  clearTimeout(settingsDebounce);
  settingsDebounce = setTimeout(async () => {
    const roomId = store.get('roomId');
    if (!roomId) return;
    try {
      await wsSendCommand(roomId, 'room_settings', {
        edition: edition,
        max_players: String(maxPlayers),
      });
    } catch (err) {
      console.error('[Room] Failed to sync settings:', err);
    }
  }, 300);
}

function bindCommonEvents() {
  const roomId = store.get('roomId');

  document.getElementById('copy-code-btn')?.addEventListener('click', () => {
    const fullCode = store.get('roomId');
    navigator.clipboard.writeText(fullCode).then(() => {
      toast('邀请码已复制！', 'success');
    }).catch(() => {
      const el = document.createElement('textarea');
      el.value = fullCode;
      document.body.appendChild(el);
      el.select();
      document.execCommand('copy');
      document.body.removeChild(el);
      toast('邀请码已复制！', 'success');
    });
  });

  document.getElementById('back-home-btn')?.addEventListener('click', () => {
    navigate('home');
  });

  document.getElementById('seats-grid')?.addEventListener('click', async (e) => {
    const claimBtn = e.target.closest('[data-claim-seat]');
    if (!claimBtn) return;
    const seatNum = claimBtn.dataset.claimSeat;
    try {
      await wsSendCommand(roomId, 'claim_seat', { seat_number: seatNum });
      toast(`已入座 #${seatNum}`, 'success');
      refreshState();
    } catch (err) {
      toast(err.message || '入座失败', 'error');
    }
  });
}

export function mountRoom() {
  const roomId = store.get('roomId');
  const isHost = store.get('isHost');

  refreshState();
  bindCommonEvents();

  if (isHost) {
    // Edition selection - host only
    document.querySelectorAll('.edition-card').forEach(card => {
      card.addEventListener('click', () => {
        const ed = card.dataset.edition;
        store.set('selectedEdition', ed);
        document.querySelectorAll('.edition-card').forEach(c => c.classList.remove('selected'));
        card.classList.add('selected');
        sendSettingsToServer(ed, store.get('selectedPlayerCount'));
      });
    });

    // Player count slider - host only
    const slider = document.getElementById('player-count-slider');
    if (slider) {
      slider.addEventListener('input', () => {
        const count = parseInt(slider.value);
        store.set('selectedPlayerCount', count);
        document.getElementById('player-count-value').textContent = count;
        const comp = getComposition(count);
        document.getElementById('composition-info').innerHTML = `
          <div class="comp-item"><span class="comp-label">村民</span><span class="comp-value">${comp.townsfolk}</span></div>
          <div class="comp-item"><span class="comp-label">外来者</span><span class="comp-value">${comp.outsider}</span></div>
          <div class="comp-item"><span class="comp-label">爪牙</span><span class="comp-value">${comp.minion}</span></div>
          <div class="comp-item"><span class="comp-label">恶魔</span><span class="comp-value">${comp.demon}</span></div>
        `;
        updateSeatsGrid(count);
        sendSettingsToServer(store.get('selectedEdition'), count);
      });
    }

    // Start game - host only
    document.getElementById('start-game-btn')?.addEventListener('click', async () => {
      const btn = document.getElementById('start-game-btn');
      btn.disabled = true;
      btn.textContent = '启动中...';
      try {
        await wsSendCommand(roomId, 'start_game', {});
        toast('游戏开始！', 'success');
      } catch (err) {
        toast(err.message || '启动失败', 'error');
        btn.disabled = false;
        btn.textContent = '🎮 开始游戏';
      }
    });
  }

  // State subscription - updates UI for all players
  stateUnsub = store.subscribe('gameState', (gs) => {
    if (!gs) return;

    if (gs.phase && gs.phase !== 'lobby') {
      navigate('game');
      return;
    }

    const serverEdition = gs.edition || 'tb';
    const serverMaxPlayers = gs.max_players || 7;

    if (!isHost) {
      // Non-host: re-render entire page to sync settings
      store.update({ selectedEdition: serverEdition, selectedPlayerCount: serverMaxPlayers });
      const app = document.getElementById('app');
      if (app) {
        const scrollY = window.scrollY;
        app.innerHTML = renderRoom();
        bindCommonEvents();
        // Re-subscribe since we replaced the DOM
        window.scrollTo(0, scrollY);
      }
    } else {
      // Host: just update dynamic parts
      updateSeatsGrid(store.get('selectedPlayerCount'));
      updateStartButton(gs);
    }
  });
}

function updateSeatsGrid(count) {
  const grid = document.getElementById('seats-grid');
  if (grid) {
    grid.innerHTML = renderSeats(count);
  }
}

function updateStartButton(gs) {
  const btn = document.getElementById('start-game-btn');
  const hint = document.getElementById('start-hint');
  if (!btn || !hint) return;

  const isHost = store.get('isHost');
  const players = gs.players || {};
  const playerCount = Object.values(players).filter(p => !p.is_dm).length;

  if (!isHost) {
    btn.disabled = true;
    btn.textContent = '⏳ 等待房主开始';
    hint.textContent = `等待房主开始游戏... (${playerCount} 人已加入)`;
  } else if (playerCount < 5) {
    btn.disabled = true;
    hint.textContent = `至少需要 5 名玩家 (当前 ${playerCount} 人)`;
  } else {
    btn.disabled = false;
    hint.textContent = `${playerCount} 名玩家已就绪`;
  }
}

export function unmountRoom() {
  if (stateUnsub) {
    stateUnsub();
    stateUnsub = null;
  }
  clearTimeout(settingsDebounce);
}

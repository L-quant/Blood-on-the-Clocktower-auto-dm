// ============================================
// 主页 (创建房间 / 加入房间)
// ============================================

import { store } from '../store.js';
import { api, wsConnect, wsSubscribe, wsSendCommand } from '../api.js';
import { toast } from '../utils.js';
import { navigate } from '../app.js';

export function renderHome() {
  const name = store.get('userEmail') || '用户';

  return `
    <div class="home-page">
      <header class="home-header">
        <h1>🩸 血染钟楼</h1>
        <div class="user-info">
          <span>👤 ${name}</span>
          <button class="btn btn-ghost btn-sm" id="logout-btn">退出</button>
        </div>
      </header>
      <div class="home-content">
        <div class="home-grid">
          <div class="card card-interactive home-card" id="create-room-card">
            <div class="icon">🏠</div>
            <h2>创建房间</h2>
            <p>创建一个新的游戏房间，邀请朋友加入。选择剧本和人数后开始游戏。</p>
            <button class="btn btn-primary" id="create-room-btn">创建房间</button>
          </div>
          <div class="card home-card">
            <div class="icon">🚪</div>
            <h2>加入房间</h2>
            <p>输入房间邀请码，加入已创建的游戏房间。</p>
            <div class="join-input-group">
              <input class="input" type="text" id="join-room-input" placeholder="输入房间邀请码">
              <button class="btn btn-primary" id="join-room-btn">加入</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}

export function mountHome() {
  // Logout
  document.getElementById('logout-btn').addEventListener('click', () => {
    store.clearAuth();
    navigate('auth');
  });

  // Create room
  document.getElementById('create-room-btn').addEventListener('click', async () => {
    const btn = document.getElementById('create-room-btn');
    btn.disabled = true;
    btn.textContent = '创建中...';

    try {
      const result = await api.createRoom();
      const roomId = result.room_id;

      store.update({ roomId, isHost: true });

      // Connect WebSocket
      wsConnect();

      // Join the room via API first
      await api.joinRoom(roomId);

      // Subscribe to room events
      setTimeout(() => {
        wsSubscribe(roomId);
        // Send join command
        wsSendCommand(roomId, 'join', { name: store.get('userEmail') || '房主' }).catch(() => {});
      }, 500);

      toast('房间创建成功！', 'success');
      navigate('room');
    } catch (err) {
      toast(err.message || '创建房间失败', 'error');
      btn.disabled = false;
      btn.textContent = '创建房间';
    }
  });

  // Join room
  document.getElementById('join-room-btn').addEventListener('click', async () => {
    const input = document.getElementById('join-room-input');
    const roomId = input.value.trim();
    if (!roomId) {
      toast('请输入房间邀请码', 'warning');
      return;
    }

    const btn = document.getElementById('join-room-btn');
    btn.disabled = true;
    btn.textContent = '加入中...';

    try {
      await api.joinRoom(roomId);
      store.update({ roomId, isHost: false });

      // Connect WebSocket & subscribe
      wsConnect();
      setTimeout(() => {
        wsSubscribe(roomId);
        wsSendCommand(roomId, 'join', { name: store.get('userEmail') || '玩家' }).catch(() => {});
      }, 500);

      toast('加入房间成功！', 'success');
      navigate('room');
    } catch (err) {
      toast(err.message || '加入房间失败', 'error');
      btn.disabled = false;
      btn.textContent = '加入';
    }
  });

  // Enter key for join input
  document.getElementById('join-room-input').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      document.getElementById('join-room-btn').click();
    }
  });
}

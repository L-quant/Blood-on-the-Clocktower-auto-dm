// ============================================
// API 客户端 (REST + WebSocket)
// ============================================

import { store } from './store.js';
import { toast, generateId } from './utils.js';

const API_BASE = window.BOTC_API_BASE || (window.location.origin === 'null' ? 'http://localhost:8080' : '');
const WS_BASE = window.BOTC_WS_BASE || (window.location.protocol === 'https:' ? `wss://${window.location.host}` : `ws://${window.location.host || 'localhost:8080'}`);

// ---- REST API ----

async function request(method, path, body = null) {
  const headers = { 'Content-Type': 'application/json' };
  const token = store.get('token');
  console.log(`[API] ${method} ${path} | token=${token ? token.substring(0, 20) + '...' : '(none)'}`);
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);

  const resp = await fetch(`${API_BASE}${path}`, opts);

  if (!resp.ok) {
    const text = await resp.text();
    console.error(`[API] ${method} ${path} failed: ${resp.status} ${text}`);

    // If 401, token is invalid/expired - clear auth and redirect to login
    if (resp.status === 401 && !path.includes('/auth/')) {
      console.warn('[API] Token expired or invalid, clearing auth...');
      store.clearAuth();
      toast('登录已过期，请重新登录', 'warning');
      // Force reload to go back to login page
      setTimeout(() => { window.location.hash = 'auth'; window.location.reload(); }, 500);
    }

    throw new Error(text || `HTTP ${resp.status}`);
  }

  // Parse response - try JSON first
  const text = await resp.text();
  console.log(`[API] ${method} ${path} response:`, text.substring(0, 200));
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

export const api = {
  register(email, password) {
    return request('POST', '/v1/auth/register', { email, password });
  },

  login(email, password) {
    return request('POST', '/v1/auth/login', { email, password });
  },

  quickLogin(name) {
    return request('POST', '/v1/auth/quick', { name });
  },

  createRoom() {
    return request('POST', '/v1/rooms');
  },

  joinRoom(roomId) {
    return request('POST', `/v1/rooms/${roomId}/join`);
  },

  getState(roomId) {
    return request('GET', `/v1/rooms/${roomId}/state`);
  },

  getEvents(roomId, afterSeq = 0) {
    return request('GET', `/v1/rooms/${roomId}/events?after_seq=${afterSeq}`);
  },

  getReplay(roomId, toSeq, viewer) {
    let url = `/v1/rooms/${roomId}/replay?to_seq=${toSeq}`;
    if (viewer) url += `&viewer=${viewer}`;
    return request('GET', url);
  },
};


// ---- WebSocket Client ----

let ws = null;
let wsReconnectTimer = null;
let wsPingTimer = null;
let wsRequestCallbacks = new Map();
let lastSeq = 0;

// generateId imported from utils.js

export function wsConnect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }

  const token = store.get('token');
  const userId = store.get('userId');
  let url = `${WS_BASE}/ws`;

  if (token) {
    url += `?token=${encodeURIComponent(token)}`;
  } else if (userId) {
    url += `?player_id=${encodeURIComponent(userId)}`;
  }

  ws = new WebSocket(url);

  ws.onopen = () => {
    console.log('[WS] Connected');
    store.set('connected', true);

    // Start ping
    clearInterval(wsPingTimer);
    wsPingTimer = setInterval(() => wsSend({ type: 'ping' }), 25000);

    // Re-subscribe if in a room
    const roomId = store.get('roomId');
    if (roomId) {
      wsSubscribe(roomId);
    }
  };

  ws.onmessage = (evt) => {
    try {
      const msg = JSON.parse(evt.data);
      handleWSMessage(msg);
    } catch (e) {
      console.error('[WS] Parse error:', e);
    }
  };

  ws.onclose = () => {
    console.log('[WS] Disconnected');
    store.set('connected', false);
    clearInterval(wsPingTimer);

    // Reconnect after 2s
    clearTimeout(wsReconnectTimer);
    wsReconnectTimer = setTimeout(() => wsConnect(), 2000);
  };

  ws.onerror = (err) => {
    console.error('[WS] Error:', err);
  };
}

export function wsDisconnect() {
  clearTimeout(wsReconnectTimer);
  clearInterval(wsPingTimer);
  if (ws) {
    ws.onclose = null;
    ws.close();
    ws = null;
  }
  store.set('connected', false);
}

function wsSend(msg) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg));
  }
}

export function wsSubscribe(roomId) {
  lastSeq = 0;
  wsSend({
    type: 'subscribe',
    payload: { room_id: roomId, last_seq: 0 },
  });
}

export function wsSendCommand(roomId, type, data = {}) {
  const commandId = generateId();
  const idempotencyKey = generateId();

  return new Promise((resolve, reject) => {
    wsRequestCallbacks.set(commandId, { resolve, reject, timeout: setTimeout(() => {
      wsRequestCallbacks.delete(commandId);
      reject(new Error('Command timeout'));
    }, 15000) });

    wsSend({
      type: 'command',
      request_id: commandId,
      payload: {
        command_id: commandId,
        idempotency_key: idempotencyKey,
        room_id: roomId,
        type: type,
        last_seen_seq: lastSeq,
        data: data,
      },
    });
  });
}

// ---- WS Message Handler ----

function handleWSMessage(msg) {
  switch (msg.type) {
    case 'pong':
      break;

    case 'subscribed':
      store.set('subscribed', true);
      console.log('[WS] Subscribed to room');
      break;

    case 'event':
      handleGameEvent(msg.payload);
      break;

    case 'command_result': {
      const cb = wsRequestCallbacks.get(msg.payload.command_id);
      if (cb) {
        clearTimeout(cb.timeout);
        wsRequestCallbacks.delete(msg.payload.command_id);
        if (msg.payload.status === 'accepted' || msg.payload.status === 'ok') {
          cb.resolve(msg.payload);
        } else {
          cb.reject(new Error(msg.payload.reason || 'Command rejected'));
        }
      }
      // Update lastSeq
      if (msg.payload.applied_seq_to) {
        lastSeq = Math.max(lastSeq, msg.payload.applied_seq_to);
      }
      break;
    }

    case 'error':
      console.error('[WS] Server error:', msg.payload);
      toast(msg.payload.message || '服务器错误', 'error');
      break;

    default:
      console.log('[WS] Unknown message type:', msg.type);
  }
}

// ---- Game Event Processing ----

// These are the events we render & handle
function handleGameEvent(event) {
  if (!event) return;

  // Update lastSeq
  if (event.seq) {
    lastSeq = Math.max(lastSeq, event.seq);
  }

  const data = typeof event.data === 'string' ? JSON.parse(event.data) : (event.data || {});
  const eventType = event.event_type || event.type;

  switch (eventType) {
    case 'player.joined':
      store.addMessage({ type: 'system', text: `${data.name || '玩家'} 加入了房间` });
      refreshState();
      break;

    case 'player.left':
      store.addMessage({ type: 'system', text: `玩家离开了房间` });
      refreshState();
      break;

    case 'seat.claimed':
      refreshState();
      break;

    case 'room.settings.changed':
      console.log('[WS] room.settings.changed:', data);
      refreshState();
      break;

    case 'game.started':
      store.addMessage({ type: 'announcement', text: '🩸 游戏开始！' });
      store.set('currentView', 'game');
      refreshState();
      break;

    case 'role.assigned': {
      const myId = store.get('userId');
      if (data.user_id === myId && data.role) {
        store.addMessage({ type: 'system', text: `你的角色是：${data.role}` });
      }
      refreshState();
      break;
    }

    case 'phase.first_night':
      store.addMessage({ type: 'announcement', text: '🌙 第一个夜晚降临...' });
      refreshState();
      break;

    case 'phase.night':
      store.addMessage({ type: 'announcement', text: '🌙 夜幕降临...' });
      refreshState();
      break;

    case 'phase.day':
      store.addMessage({ type: 'announcement', text: '☀️ 天亮了！' });
      store.update({ nightActionPending: false, nightResult: '' });
      refreshState();
      break;

    case 'phase.nomination':
      store.addMessage({ type: 'announcement', text: '📢 提名阶段开始' });
      refreshState();
      break;

    case 'public.chat':
      store.addMessage({
        type: 'public',
        sender: data.sender_name || '未知',
        senderSeat: data.sender_seat,
        text: data.message,
      });
      break;

    case 'whisper.sent':
      store.addMessage({
        type: 'whisper',
        sender: data.sender_name || '未知',
        text: data.message,
        to: data.to_user_id,
      });
      break;

    case 'nomination.created':
      store.addMessage({
        type: 'system',
        text: `提名发起！座位${data.nominator_seat} → 座位${data.nominee_seat}`,
      });
      refreshState();
      break;

    case 'defense.ended':
      store.addMessage({ type: 'system', text: '辩护结束，开始投票！' });
      refreshState();
      break;

    case 'vote.cast':
      store.addMessage({
        type: 'system',
        text: `座位${data.voter_seat} 投出了 ${data.vote === 'yes' ? '✅ 赞成' : '❌ 反对'} 票`,
      });
      refreshState();
      break;

    case 'nomination.resolved':
      store.addMessage({
        type: 'announcement',
        text: data.result === 'executed' ? '⚖️ 提名通过！处决执行！' :
              data.result === 'cancelled' ? '❌ 提名被取消' : '⚖️ 提名未通过',
      });
      refreshState();
      break;

    case 'execution.resolved':
      if (data.result === 'executed') {
        store.addMessage({ type: 'death', text: `💀 一名玩家被处决！` });
      }
      refreshState();
      break;

    case 'player.died':
      store.addMessage({
        type: 'death',
        text: `💀 玩家死亡 (${data.cause === 'demon' ? '恶魔袭击' :
          data.cause === 'execution' ? '处决' :
          data.cause === 'virgin_ability' ? '贞洁者能力' :
          data.cause === 'slayer' ? '杀手射击' : data.cause})`,
      });
      refreshState();
      break;

    case 'slayer.shot':
      store.addMessage({
        type: 'announcement',
        text: `🔫 杀手开枪了！`,
      });
      refreshState();
      break;

    case 'ability.resolved': {
      // Show result to the player who used the ability
      if (data.result) {
        store.set('nightResult', data.result);
        store.addMessage({ type: 'system', text: `能力结果：${data.result}` });
      }
      refreshState();
      break;
    }

    case 'game.ended':
      store.addMessage({
        type: 'announcement',
        text: `🏆 游戏结束！${data.winner === 'good' ? '善良阵营' : '邪恶阵营'}获胜！原因：${data.reason}`,
      });
      refreshState();
      break;

    default:
      // For events we don't handle specifically, still refresh state
      refreshState();
  }
}

// Fetch latest projected state from the server
let refreshDebounce = null;
function refreshState() {
  clearTimeout(refreshDebounce);
  refreshDebounce = setTimeout(async () => {
    const roomId = store.get('roomId');
    if (!roomId) return;
    try {
      const state = await api.getState(roomId);
      store.set('gameState', state);

      // Check if we need to handle night action
      checkNightAction(state);
    } catch (e) {
      console.error('[API] Failed to refresh state:', e);
    }
  }, 100);
}

function checkNightAction(state) {
  if (!state) return;
  const phase = state.phase;
  if (phase !== 'first_night' && phase !== 'night') {
    store.update({ nightActionPending: false });
    return;
  }

  const myPlayer = store.getMyPlayer();
  if (!myPlayer || !myPlayer.role) return;

  // Check if there's a pending action for me
  const actions = state.night_actions || [];
  const currentIdx = state.current_action || 0;

  if (currentIdx < actions.length) {
    const currentAction = actions[currentIdx];
    if (currentAction.user_id === store.get('userId') && !currentAction.completed) {
      store.update({
        nightActionPending: true,
        nightActionRole: myPlayer.role,
      });
      return;
    }
  }

  store.update({ nightActionPending: false });
}

export { refreshState };

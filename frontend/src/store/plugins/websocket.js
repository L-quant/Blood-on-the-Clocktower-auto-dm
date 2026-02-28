// Vuex 插件：WebSocket 连接管理、命令发送、重连
//
// [IN]  services/ApiService（REST 状态同步与认证）
// [IN]  ws_game_events.js（事件处理）
// [IN]  ws_state_sync.js（状态同步）
// [OUT] store/index.js（插件注册）
// [POS] 前后端实时通信桥梁

/**
 * WebSocket plugin for Vuex
 * Bridges WebSocket events to store mutations.
 *
 * Backend protocol:
 *   - Connect: ws://host/ws?token=<jwt>
 *   - Subscribe: { type: "subscribe", payload: { room_id, last_seq } }
 *   - Command:   { type: "command",   payload: { command_id, room_id, type, data } }
 *   - Ping:      { type: "ping" }
 *
 * Backend events arrive as:
 *   { type: "event", payload: { event_type, data, seq, server_ts } }
 */

import apiService from "../../services/ApiService";
import { processGameEvent } from "./ws_game_events";
import { syncRoomState } from "./ws_state_sync";

const WS_URL = process.env.VUE_APP_WS_URL || 'ws://localhost:8080/ws';

class WebSocketManager {
  constructor(store) {
    this._store = store;
    this._socket = null;
    this._pingTimer = null;
    this._reconnectTimer = null;
    this._reconnectDelay = 1000;
    this._maxReconnectDelay = 30000;
    this._lastSeq = 0;
    this._pingInterval = 25000;
    this._roomId = null;
    this._pendingRequests = {};
  }

  connect(roomId) {
    this.disconnect();
    const token = apiService.token;
    if (!token) return;

    this._roomId = roomId;
    const wsUrl = `${WS_URL}?token=${encodeURIComponent(token)}`;
    this._socket = new WebSocket(wsUrl);
    this._reconnectDelay = 1000;

    this._socket.onopen = () => {
      this._store.commit('setConnected', true);
      this._store.commit('setReconnecting', false);
      this._send('subscribe', { room_id: roomId, last_seq: this._lastSeq });
      this._startPing();
    };

    this._socket.onmessage = (event) => {
      this._handleMessage(event.data);
    };

    this._socket.onclose = (event) => {
      this._socket = null;
      this._stopPing();
      this._pendingRequests = {};
      this._store.commit('setConnected', false);
      this._store.commit('vote/setVotePending', false);
      if (event.code !== 1000 && this._roomId) {
        this._store.commit('setReconnecting', true);
        this._scheduleReconnect();
      }
    };

    this._socket.onerror = () => {};
  }

  disconnect() {
    this._stopPing();
    clearTimeout(this._reconnectTimer);
    this._roomId = null;
    this._store.commit('setReconnecting', false);
    if (this._socket) {
      this._socket.close(1000);
      this._socket = null;
    }
    this._store.commit('setConnected', false);
  }

  send(command, data) {
    if (this._socket && this._socket.readyState === WebSocket.OPEN) {
      const requestId = Math.random().toString(36).substr(2);
      this._pendingRequests[requestId] = command;
      this._socket.send(JSON.stringify({
        type: 'command',
        request_id: requestId,
        payload: {
          command_id: Math.random().toString(36).substr(2),
          room_id: this._roomId,
          type: command,
          data: data
        }
      }));
    }
  }

  _send(type, payload) {
    if (this._socket && this._socket.readyState === WebSocket.OPEN) {
      this._socket.send(JSON.stringify({
        type, request_id: Math.random().toString(36).substr(2), payload
      }));
    }
  }

  _handleMessage(raw) {
    let parsed;
    try { parsed = JSON.parse(raw); } catch (_e) { return; }
    const type = parsed.type;

    switch (type) {
      case 'subscribed':
        this._store.commit('setReconnecting', false);
        this.send('join', { name: 'Player' });
        setTimeout(() => this._fetchRoomState(), 500);
        break;
      case 'pong':
        this._handlePong(parsed.payload);
        break;
      case 'error':
        break;
      case 'command_result':
        this._handleCommandResult(parsed);
        break;
      case 'event': {
        let pe;
        if (typeof parsed.payload === 'string') {
          try { pe = JSON.parse(parsed.payload); } catch (_e) { return; }
        } else {
          pe = parsed.payload;
        }
        if (pe && pe.seq) this._lastSeq = Math.max(this._lastSeq, pe.seq);
        processGameEvent(pe, this._store);
        break;
      }
      default:
        break;
    }
  }

  _handleCommandResult(parsed) {
    let result = parsed.payload;
    if (typeof result === 'string') {
      try { result = JSON.parse(result); } catch (_e) { return; }
    }
    const reqId = parsed.request_id || (result && result.request_id);
    const cmdType = this._pendingRequests[reqId];
    if (reqId) delete this._pendingRequests[reqId];
    if (result && result.status === 'rejected') {
      if (cmdType === 'vote') this._store.commit('vote/setVotePending', false);
      console.warn(`Command [${cmdType}] rejected:`, result.reason);
    }
  }

  _fetchRoomState() {
    if (!this._roomId) return;
    apiService.getRoomState(this._roomId).then(state => {
      syncRoomState(state, this._store);
    }).catch(() => {});
  }

  _startPing() {
    this._stopPing();
    this._pingTimer = setInterval(() => {
      this._send('ping', { timestamp: Date.now() });
    }, this._pingInterval);
  }

  _stopPing() {
    if (this._pingTimer) { clearInterval(this._pingTimer); this._pingTimer = null; }
  }

  _handlePong(payload) {
    if (payload && payload.timestamp) {
      this._store.commit('setLatency', Date.now() - payload.timestamp);
    }
  }

  _scheduleReconnect() {
    this._reconnectTimer = setTimeout(() => {
      if (!this._store.state.connected && this._roomId) this.connect(this._roomId);
    }, this._reconnectDelay);
    this._reconnectDelay = Math.min(this._reconnectDelay * 2, this._maxReconnectDelay);
  }
}

export default store => {
  const ws = new WebSocketManager(store);
  store.$ws = ws;
  store.subscribe(({ type, payload }) => {
    switch (type) {
      case 'setRoomId':
        if (payload) { ws.connect(payload); } else { ws.disconnect(); }
        break;
      case 'disconnect':
        ws.disconnect();
        break;
      case 'sendCommand':
        if (payload && payload.type) ws.send(payload.type, payload.data || {});
        break;
    }
  });
};

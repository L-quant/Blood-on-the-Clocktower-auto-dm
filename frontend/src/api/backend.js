/**
 * Backend API Service
 * 连接到 Go 后端的 API 和 WebSocket 服务
 */

const API_BASE = process.env.VUE_APP_API_URL || 'http://localhost:8080';
const WS_BASE = process.env.VUE_APP_WS_URL || 'ws://localhost:8080';

class BackendAPI {
  constructor() {
    this._token = localStorage.getItem('botc_token') || '';
    this._playerId = localStorage.getItem('botc_player_id') || '';
    this._roomId = '';
    this._ws = null;
    this._reconnectTimer = null;
    this._listeners = new Map();
    this._lastSeq = 0;
  }

  /**
   * 生成或获取玩家 ID
   */
  getPlayerId() {
    if (!this._playerId) {
      this._playerId = Math.random().toString(36).substr(2, 10);
      localStorage.setItem('botc_player_id', this._playerId);
    }
    return this._playerId;
  }

  /**
   * 创建房间
   * @returns {Promise<{room_id: string}>}
   */
  async createRoom() {
    const response = await fetch(`${API_BASE}/v1/rooms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(this._token ? { 'Authorization': `Bearer ${this._token}` } : {})
      },
      body: JSON.stringify({
        player_id: this.getPlayerId()
      })
    });
    if (!response.ok) {
      throw new Error(`Failed to create room: ${response.status}`);
    }
    const data = await response.json();
    this._roomId = data.room_id;
    return data;
  }

  /**
   * 加入房间
   * @param {string} roomId
   * @returns {Promise<Object>}
   */
  async joinRoom(roomId) {
    const response = await fetch(`${API_BASE}/v1/rooms/${roomId}/join`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(this._token ? { 'Authorization': `Bearer ${this._token}` } : {})
      },
      body: JSON.stringify({
        player_id: this.getPlayerId()
      })
    });
    if (!response.ok) {
      throw new Error(`Failed to join room: ${response.status}`);
    }
    const data = await response.json();
    this._roomId = roomId;
    return data;
  }

  /**
   * 获取房间状态
   * @param {string} roomId
   * @returns {Promise<Object>}
   */
  async getRoomState(roomId) {
    const response = await fetch(`${API_BASE}/v1/rooms/${roomId}`, {
      headers: {
        ...(this._token ? { 'Authorization': `Bearer ${this._token}` } : {})
      }
    });
    if (!response.ok) {
      throw new Error(`Failed to get room state: ${response.status}`);
    }
    return response.json();
  }

  /**
   * 获取事件流
   * @param {string} roomId
   * @param {number} afterSeq
   * @returns {Promise<Object[]>}
   */
  async getEvents(roomId, afterSeq = 0) {
    const response = await fetch(`${API_BASE}/v1/rooms/${roomId}/events?after_seq=${afterSeq}`, {
      headers: {
        ...(this._token ? { 'Authorization': `Bearer ${this._token}` } : {})
      }
    });
    if (!response.ok) {
      throw new Error(`Failed to get events: ${response.status}`);
    }
    return response.json();
  }

  /**
   * 发送命令
   * @param {string} roomId
   * @param {string} commandType
   * @param {Object} data
   * @returns {Promise<Object>}
   */
  async sendCommand(roomId, commandType, data = {}) {
    const commandId = Math.random().toString(36).substr(2, 10);
    const response = await fetch(`${API_BASE}/v1/rooms/${roomId}/commands`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(this._token ? { 'Authorization': `Bearer ${this._token}` } : {})
      },
      body: JSON.stringify({
        command_id: commandId,
        type: commandType,
        player_id: this.getPlayerId(),
        data
      })
    });
    if (!response.ok) {
      throw new Error(`Failed to send command: ${response.status}`);
    }
    return response.json();
  }

  /**
   * 连接 WebSocket
   * @param {string} roomId
   * @param {Function} onMessage
   * @param {Function} onConnect
   * @param {Function} onDisconnect
   */
  connectWebSocket(roomId, { onMessage, onConnect, onDisconnect, onError }) {
    this._roomId = roomId;
    
    const wsUrl = `${WS_BASE}/ws?player_id=${this.getPlayerId()}`;
    
    if (this._ws) {
      this._ws.close();
    }

    this._ws = new WebSocket(wsUrl);

    this._ws.onopen = () => {
      console.log('[WS] Connected');
      // 订阅房间
      this._send({
        type: 'subscribe',
        request_id: Math.random().toString(36).substr(2),
        payload: {
          room_id: roomId,
          last_seq: this._lastSeq
        }
      });
      if (onConnect) onConnect();
    };

    this._ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        console.log('[WS] Message:', message);
        
        // 更新 last_seq
        if (message.seq) {
          this._lastSeq = Math.max(this._lastSeq, message.seq);
        }
        
        if (onMessage) onMessage(message);
        
        // 触发特定类型的监听器
        const listeners = this._listeners.get(message.type) || [];
        listeners.forEach(cb => cb(message));
      } catch (err) {
        console.error('[WS] Failed to parse message:', err);
      }
    };

    this._ws.onclose = (event) => {
      console.log('[WS] Disconnected:', event.code, event.reason);
      this._ws = null;
      
      if (event.code !== 1000 && this._roomId) {
        // 非正常关闭，尝试重连
        this._reconnectTimer = setTimeout(() => {
          this.connectWebSocket(roomId, { onMessage, onConnect, onDisconnect, onError });
        }, 3000);
      }
      
      if (onDisconnect) onDisconnect(event);
    };

    this._ws.onerror = (error) => {
      console.error('[WS] Error:', error);
      if (onError) onError(error);
    };
  }

  /**
   * 断开 WebSocket
   */
  disconnectWebSocket() {
    clearTimeout(this._reconnectTimer);
    this._roomId = '';
    if (this._ws) {
      this._ws.close(1000);
      this._ws = null;
    }
  }

  /**
   * 通过 WebSocket 发送消息
   * @param {Object} message
   */
  _send(message) {
    if (this._ws && this._ws.readyState === WebSocket.OPEN) {
      this._ws.send(JSON.stringify(message));
    }
  }

  /**
   * 发送命令通过 WebSocket
   * @param {string} commandType
   * @param {Object} data
   */
  sendWSCommand(commandType, data = {}) {
    this._send({
      type: 'command',
      request_id: Math.random().toString(36).substr(2),
      payload: {
        command_id: Math.random().toString(36).substr(2),
        type: commandType,
        data
      }
    });
  }

  /**
   * 添加事件监听器
   * @param {string} eventType
   * @param {Function} callback
   */
  on(eventType, callback) {
    if (!this._listeners.has(eventType)) {
      this._listeners.set(eventType, []);
    }
    this._listeners.get(eventType).push(callback);
  }

  /**
   * 移除事件监听器
   * @param {string} eventType
   * @param {Function} callback
   */
  off(eventType, callback) {
    const listeners = this._listeners.get(eventType) || [];
    const index = listeners.indexOf(callback);
    if (index > -1) {
      listeners.splice(index, 1);
    }
  }
}

// 单例导出
export const backendAPI = new BackendAPI();
export default backendAPI;

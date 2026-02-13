// ============================================
// 响应式状态管理 Store
// ============================================

// Support multi-account testing: ?as=2 uses separate localStorage namespace
const urlParams = new URLSearchParams(window.location.search);
const STORAGE_SUFFIX = urlParams.get('as') ? `_${urlParams.get('as')}` : '';
const KEY_TOKEN = 'botc_token' + STORAGE_SUFFIX;
const KEY_USER_ID = 'botc_user_id' + STORAGE_SUFFIX;
const KEY_EMAIL = 'botc_email' + STORAGE_SUFFIX;

class Store {
  constructor() {
    this._state = {
      // Auth
      token: localStorage.getItem(KEY_TOKEN) || '',
      userId: localStorage.getItem(KEY_USER_ID) || '',
      userEmail: localStorage.getItem(KEY_EMAIL) || '',

      // Connection
      connected: false,
      subscribed: false,

      // Room
      roomId: '',
      isHost: false,

      // Game State (from backend projection)
      gameState: null,

      // Messages
      messages: [],

      // UI
      currentView: 'auth', // auth, home, room, game
      selectedEdition: 'tb',
      selectedPlayerCount: 7,

      // Night action
      nightActionPending: false,
      nightActionRole: '',
      nightActionTargets: [],
      nightResult: '',

      // Player selection (for abilities)
      selectedTargets: [],
    };

    this._listeners = new Map();
    this._globalListeners = new Set();
  }

  get(key) {
    return this._state[key];
  }

  getAll() {
    return { ...this._state };
  }

  set(key, value) {
    const old = this._state[key];
    this._state[key] = value;
    if (old !== value) {
      this._notify(key, value, old);
    }
  }

  // Update multiple keys at once
  update(updates) {
    const changed = [];
    for (const [key, value] of Object.entries(updates)) {
      const old = this._state[key];
      this._state[key] = value;
      if (old !== value) {
        changed.push([key, value, old]);
      }
    }
    // Notify after all updates
    for (const [key, value, old] of changed) {
      this._notify(key, value, old);
    }
  }

  subscribe(key, fn) {
    if (!this._listeners.has(key)) {
      this._listeners.set(key, new Set());
    }
    this._listeners.get(key).add(fn);
    return () => this._listeners.get(key).delete(fn);
  }

  // Global listener - called on any state change
  onAny(fn) {
    this._globalListeners.add(fn);
    return () => this._globalListeners.delete(fn);
  }

  _notify(key, value, old) {
    const listeners = this._listeners.get(key);
    if (listeners) {
      listeners.forEach(fn => fn(value, old, key));
    }
    this._globalListeners.forEach(fn => fn(key, value, old));
  }

  // Auth helpers
  setAuth(token, userId, email) {
    localStorage.setItem(KEY_TOKEN, token);
    localStorage.setItem(KEY_USER_ID, userId);
    localStorage.setItem(KEY_EMAIL, email || '');
    this.update({ token, userId, userEmail: email || '' });
  }

  clearAuth() {
    localStorage.removeItem(KEY_TOKEN);
    localStorage.removeItem(KEY_USER_ID);
    localStorage.removeItem(KEY_EMAIL);
    this.update({ token: '', userId: '', userEmail: '', currentView: 'auth' });
  }

  isLoggedIn() {
    return !!this._state.token;
  }

  // Game state helpers
  getMyPlayer() {
    const gs = this._state.gameState;
    if (!gs || !gs.players) return null;
    return gs.players[this._state.userId] || null;
  }

  getPlayerList() {
    const gs = this._state.gameState;
    if (!gs || !gs.players) return [];
    const players = Object.values(gs.players).filter(p => !p.is_dm);
    players.sort((a, b) => a.seat_number - b.seat_number);
    return players;
  }

  getPhase() {
    const gs = this._state.gameState;
    return gs ? gs.phase : 'lobby';
  }

  addMessage(msg) {
    const msgs = [...this._state.messages, msg];
    // Keep last 200 messages
    if (msgs.length > 200) msgs.splice(0, msgs.length - 200);
    this.set('messages', msgs);
  }
}

export const store = new Store();

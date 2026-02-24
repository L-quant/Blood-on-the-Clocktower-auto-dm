/**
 * ApiService - REST API client for the BOTC backend
 * Handles JWT authentication, room management, and game commands.
 *
 * Uses sessionStorage so each browser tab gets its own player identity.
 * This enables multi-player testing with multiple tabs.
 */

const API_BASE = process.env.VUE_APP_API_URL || 'http://localhost:8080';
const TOKEN_KEY = 'botc_token';
const USER_ID_KEY = 'botc_user_id';

class ApiService {
  constructor() {
    this._token = sessionStorage.getItem(TOKEN_KEY) || '';
    this._userId = sessionStorage.getItem(USER_ID_KEY) || '';
  }

  /** Returns true if we have a stored JWT token */
  get isAuthenticated() {
    return !!this._token;
  }

  /** Returns the current JWT token */
  get token() {
    return this._token;
  }

  /** Returns the current user ID */
  get userId() {
    return this._userId;
  }

  /**
   * Quick login — creates a temporary account with just a display name.
   * Backend: POST /v1/auth/quick { name }
   * Returns: { token, user_id, name }
   */
  async quickLogin(name) {
    const response = await fetch(`${API_BASE}/v1/auth/quick`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: name || 'Player' })
    });
    if (!response.ok) {
      const text = await response.text().catch(() => '');
      throw new Error(`Login failed: ${text}`);
    }
    const data = await response.json();
    this._token = data.token;
    this._userId = data.user_id;
    sessionStorage.setItem(TOKEN_KEY, this._token);
    sessionStorage.setItem(USER_ID_KEY, this._userId);
    return data;
  }

  /**
   * Ensure we have a valid auth token. If not, do quick login.
   */
  async ensureAuth() {
    if (this._token) return;
    const name = 'Player_' + Math.random().toString(36).substr(2, 4);
    await this.quickLogin(name);
  }

  /**
   * Internal fetch wrapper — injects Authorization header.
   */
  async _fetch(path, options = {}) {
    await this.ensureAuth();
    const response = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this._token}`,
        ...options.headers
      }
    });
    if (response.status === 401) {
      // Token expired — clear and re-auth
      this._token = '';
      sessionStorage.removeItem(TOKEN_KEY);
      await this.ensureAuth();
      // Retry once
      const retry = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this._token}`,
          ...options.headers
        }
      });
      if (!retry.ok) {
        const text = await retry.text().catch(() => '');
        throw new Error(`API error ${retry.status}: ${text}`);
      }
      return retry.json();
    }
    if (!response.ok) {
      const text = await response.text().catch(() => '');
      throw new Error(`API error ${response.status}: ${text}`);
    }
    return response.json();
  }

  /**
   * Create a new game room.
   * Backend: POST /v1/rooms (auth required)
   * Returns: { room_id }
   */
  async createRoom() {
    return this._fetch('/v1/rooms', {
      method: 'POST',
      body: JSON.stringify({})
    });
  }

  /**
   * Join an existing game room.
   * Backend: POST /v1/rooms/{room_id}/join (auth required)
   * Returns: { status: "joined" }
   */
  async joinRoom(roomId) {
    return this._fetch(`/v1/rooms/${roomId}/join`, {
      method: 'POST',
      body: JSON.stringify({})
    });
  }

  /**
   * Get current room state.
   * Backend: GET /v1/rooms/{room_id}/state (auth required)
   */
  async getRoomState(roomId) {
    return this._fetch(`/v1/rooms/${roomId}/state`);
  }

  /**
   * Fetch events after a given sequence number.
   * Backend: GET /v1/rooms/{room_id}/events?after_seq=N (auth required)
   */
  async getEvents(roomId, afterSeq = 0) {
    return this._fetch(`/v1/rooms/${roomId}/events?after_seq=${afterSeq}`);
  }

  /**
   * Ask the AI assistant.
   * Backend: POST /v1/rooms/{room_id}/assistant (may or may not exist yet)
   */
  async askAssistant(roomId, question, context = {}) {
    return this._fetch(`/v1/rooms/${roomId}/assistant`, {
      method: 'POST',
      body: JSON.stringify({
        question,
        context
      })
    });
  }

  /**
   * Clear stored auth data (for logout / leaving).
   */
  clearAuth() {
    this._token = '';
    this._userId = '';
    sessionStorage.removeItem(TOKEN_KEY);
    sessionStorage.removeItem(USER_ID_KEY);
  }
}

export const apiService = new ApiService();
export default apiService;

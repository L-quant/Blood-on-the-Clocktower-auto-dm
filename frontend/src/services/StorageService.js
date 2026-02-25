// localStorage 封装：全局与按房间 scope 的存储
//
// [OUT] 未被直接引用
// [POS] 存储服务层，提供结构化的 localStorage 访问

/**
 * StorageService - localStorage utility with room-scoped storage
 */

class StorageService {
  get(key, defaultValue = null) {
    try {
      const value = localStorage.getItem(key);
      return value ? JSON.parse(value) : defaultValue;
    } catch (e) {
      return defaultValue;
    }
  }

  set(key, value) {
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch (e) {
      // Storage full or unavailable
    }
  }

  remove(key) {
    try {
      localStorage.removeItem(key);
    } catch (e) {
      // ignore
    }
  }

  // Room-scoped storage
  getRoomData(roomId, key, defaultValue = null) {
    return this.get(`botc_${roomId}_${key}`, defaultValue);
  }

  setRoomData(roomId, key, value) {
    this.set(`botc_${roomId}_${key}`, value);
  }

  removeRoomData(roomId, key) {
    this.remove(`botc_${roomId}_${key}`);
  }

  // Annotations persistence
  getAnnotations(roomId) {
    return this.getRoomData(roomId, 'annotations', {});
  }

  saveAnnotations(roomId, annotations) {
    this.setRoomData(roomId, 'annotations', annotations);
  }

  // Settings persistence
  getSettings() {
    return this.get('botc_settings', {
      soundEnabled: true,
      animationsEnabled: true,
      locale: 'zh',
      reducedMotion: false
    });
  }

  saveSettings(settings) {
    this.set('botc_settings', settings);
  }

  // Player ID persistence
  getPlayerId() {
    return localStorage.getItem('botc_player_id') || '';
  }

  setPlayerId(id) {
    localStorage.setItem('botc_player_id', id);
  }
}

export const storageService = new StorageService();
export default storageService;

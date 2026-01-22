/**
 * 游戏模式切换器
 * 管理说书人模式和无说书人模式之间的切换
 */

import { GameMode } from './RolePrivacySystem';

/**
 * 游戏模式切换器类
 */
export default class ModeSwitcher {
  constructor() {
    this.currentMode = GameMode.STORYTELLER;
    this.gameStarted = false;
    this.modeChangeListeners = [];
  }

  /**
   * 获取当前游戏模式
   * @returns {string}
   */
  getCurrentMode() {
    return this.currentMode;
  }

  /**
   * 切换游戏模式
   * @param {string} newMode - 新的游戏模式
   * @returns {boolean} 是否切换成功
   */
  switchMode(newMode) {
    // 验证新模式是否有效
    if (!this._isValidMode(newMode)) {
      console.error(`[ModeSwitcher] Invalid game mode: ${newMode}`);
      return false;
    }

    // 检查是否可以切换模式
    if (!this.canSwitchMode()) {
      console.warn('[ModeSwitcher] Cannot switch mode after game has started');
      return false;
    }

    // 如果模式相同，无需切换
    if (newMode === this.currentMode) {
      console.log(`[ModeSwitcher] Already in ${newMode} mode`);
      return true;
    }

    const oldMode = this.currentMode;
    this.currentMode = newMode;

    console.log(`[ModeSwitcher] Switched from ${oldMode} to ${newMode}`);

    // 通知所有监听器
    this.notifyModeChange(newMode, oldMode);

    return true;
  }

  /**
   * 验证是否可以切换模式
   * @returns {boolean}
   */
  canSwitchMode() {
    // 游戏开始后不允许切换模式
    if (this.gameStarted) {
      return false;
    }

    return true;
  }

  /**
   * 标记游戏已开始
   */
  markGameStarted() {
    this.gameStarted = true;
    console.log('[ModeSwitcher] Game started - mode switching locked');
  }

  /**
   * 重置游戏状态（允许再次切换模式）
   */
  resetGameState() {
    this.gameStarted = false;
    console.log('[ModeSwitcher] Game state reset - mode switching unlocked');
  }

  /**
   * 通知模式变化
   * @param {string} newMode - 新模式
   * @param {string} oldMode - 旧模式
   */
  notifyModeChange(newMode, oldMode) {
    const event = {
      newMode,
      oldMode,
      timestamp: Date.now()
    };

    // 调用所有监听器
    this.modeChangeListeners.forEach(listener => {
      try {
        listener(event);
      } catch (error) {
        console.error('[ModeSwitcher] Error in mode change listener:', error);
      }
    });
  }

  /**
   * 添加模式变化监听器
   * @param {Function} listener - 监听器函数
   */
  addModeChangeListener(listener) {
    if (typeof listener === 'function') {
      this.modeChangeListeners.push(listener);
    }
  }

  /**
   * 移除模式变化监听器
   * @param {Function} listener - 监听器函数
   */
  removeModeChangeListener(listener) {
    const index = this.modeChangeListeners.indexOf(listener);
    if (index > -1) {
      this.modeChangeListeners.splice(index, 1);
    }
  }

  /**
   * 应用模式配置
   * @param {string} mode - 游戏模式
   */
  applyModeConfiguration(mode) {
    if (!this._isValidMode(mode)) {
      console.error(`[ModeSwitcher] Invalid mode for configuration: ${mode}`);
      return;
    }

    this.currentMode = mode;
    console.log(`[ModeSwitcher] Applied mode configuration: ${mode}`);
  }

  /**
   * 验证模式是否有效
   * @private
   * @param {string} mode - 游戏模式
   * @returns {boolean}
   */
  _isValidMode(mode) {
    return mode === GameMode.STORYTELLER || mode === GameMode.PLAYER_ONLY;
  }

  /**
   * 获取模式描述
   * @param {string} mode - 游戏模式
   * @returns {string}
   */
  getModeDescription(mode) {
    switch (mode) {
      case GameMode.STORYTELLER:
        return '说书人模式：需要一个说书人来管理游戏，说书人可以看到所有角色';
      case GameMode.PLAYER_ONLY:
        return '无说书人模式：系统自动管理游戏，每个玩家只能看到自己的角色';
      default:
        return '未知模式';
    }
  }

  /**
   * 检查是否为无说书人模式
   * @returns {boolean}
   */
  isPlayerOnlyMode() {
    return this.currentMode === GameMode.PLAYER_ONLY;
  }

  /**
   * 检查是否为说书人模式
   * @returns {boolean}
   */
  isStorytellerMode() {
    return this.currentMode === GameMode.STORYTELLER;
  }

  /**
   * 重置切换器
   */
  reset() {
    this.currentMode = GameMode.STORYTELLER;
    this.gameStarted = false;
    this.modeChangeListeners = [];
    console.log('[ModeSwitcher] Reset to default state');
  }
}

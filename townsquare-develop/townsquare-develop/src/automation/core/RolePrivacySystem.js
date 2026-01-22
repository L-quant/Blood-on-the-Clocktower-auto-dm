/**
 * 角色隐私保护系统
 * 负责管理无说书人模式下的角色隐私保护
 */

import PerspectiveFilter from './PerspectiveFilter';
import ModeSwitcher from './ModeSwitcher';

/**
 * 游戏模式枚举
 */
export const GameMode = {
  STORYTELLER: 'storyteller',    // 说书人模式
  PLAYER_ONLY: 'player-only'     // 无说书人模式
};

/**
 * 角色隐私保护系统类
 */
export default class RolePrivacySystem {
  constructor() {
    this.isEnabled = false;
    this.currentMode = GameMode.STORYTELLER;
    this.perspectiveFilter = null;
    this.modeSwitcher = null;
    this.accessLogs = [];
  }

  /**
   * 初始化隐私保护系统
   * @param {string} gameMode - 游戏模式
   */
  initialize(gameMode = GameMode.STORYTELLER) {
    this.currentMode = gameMode;
    this.perspectiveFilter = new PerspectiveFilter();
    this.modeSwitcher = new ModeSwitcher();
    
    // 如果是无说书人模式，自动启用隐私保护
    if (gameMode === GameMode.PLAYER_ONLY) {
      this.enablePrivacyProtection();
    }
    
    console.log(`[RolePrivacySystem] Initialized in ${gameMode} mode`);
  }

  /**
   * 启用隐私保护
   */
  enablePrivacyProtection() {
    this.isEnabled = true;
    console.log('[RolePrivacySystem] Privacy protection enabled');
  }

  /**
   * 禁用隐私保护
   */
  disablePrivacyProtection() {
    this.isEnabled = false;
    console.log('[RolePrivacySystem] Privacy protection disabled');
  }

  /**
   * 检查是否为无说书人模式
   * @returns {boolean}
   */
  isPlayerOnlyMode() {
    return this.currentMode === GameMode.PLAYER_ONLY;
  }

  /**
   * 过滤游戏状态（为特定玩家）
   * @param {object} gameState - 完整游戏状态
   * @param {string} playerId - 玩家ID
   * @returns {object} 过滤后的游戏状态
   */
  filterGameStateForPlayer(gameState, playerId) {
    if (!this.isEnabled || !this.isPlayerOnlyMode()) {
      // 说书人模式或隐私保护未启用，返回完整状态
      return gameState;
    }

    if (!this.perspectiveFilter) {
      console.error('[RolePrivacySystem] PerspectiveFilter not initialized');
      return gameState;
    }

    // 使用视角过滤器过滤状态
    return this.perspectiveFilter.filterGameState(gameState, playerId);
  }

  /**
   * 验证玩家是否可以访问目标玩家的角色
   * @param {string} playerId - 请求者玩家ID
   * @param {string} targetPlayerId - 目标玩家ID
   * @param {object} gameState - 游戏状态（可选，用于检查游戏是否结束）
   * @returns {boolean}
   */
  canPlayerAccessRole(playerId, targetPlayerId, gameState = null) {
    // 记录访问尝试
    this.logAccessAttempt(playerId, 'accessRole', targetPlayerId);

    // 说书人模式或隐私保护未启用，允许访问
    if (!this.isEnabled || !this.isPlayerOnlyMode()) {
      return true;
    }

    // 游戏结束时，所有角色公开
    if (gameState && gameState.phase === 'ended') {
      return true;
    }

    // 无说书人模式下，只能访问自己的角色
    const canAccess = playerId === targetPlayerId;
    
    if (!canAccess) {
      console.warn(`[RolePrivacySystem] Player ${playerId} attempted to access role of ${targetPlayerId}`);
      this.logSecurityViolation(playerId, 'unauthorized_role_access', targetPlayerId);
    }

    return canAccess;
  }

  /**
   * 切换游戏模式
   * @param {string} newMode - 新的游戏模式
   * @returns {boolean} 是否切换成功
   */
  switchMode(newMode) {
    if (!this.modeSwitcher) {
      console.error('[RolePrivacySystem] ModeSwitcher not initialized');
      return false;
    }

    const success = this.modeSwitcher.switchMode(newMode);
    
    if (success) {
      this.currentMode = newMode;
      
      // 根据新模式启用或禁用隐私保护
      if (newMode === GameMode.PLAYER_ONLY) {
        this.enablePrivacyProtection();
      } else {
        this.disablePrivacyProtection();
      }
    }

    return success;
  }

  /**
   * 获取当前游戏模式
   * @returns {string}
   */
  getCurrentMode() {
    return this.currentMode;
  }

  /**
   * 记录访问尝试
   * @param {string} playerId - 玩家ID
   * @param {string} action - 操作类型
   * @param {string} targetPlayerId - 目标玩家ID（可选）
   */
  logAccessAttempt(playerId, action, targetPlayerId = null) {
    const log = {
      timestamp: Date.now(),
      playerId,
      action,
      targetPlayerId,
      success: targetPlayerId ? playerId === targetPlayerId : true,
      type: 'access'
    };

    this.accessLogs.push(log);

    // 保持日志数量在合理范围内
    if (this.accessLogs.length > 1000) {
      this.accessLogs.shift();
    }
  }

  /**
   * 记录安全违规
   * @param {string} playerId - 玩家ID
   * @param {string} violationType - 违规类型
   * @param {string} details - 详细信息
   */
  logSecurityViolation(playerId, violationType, details = null) {
    const log = {
      timestamp: Date.now(),
      playerId,
      action: violationType,
      targetPlayerId: details,
      success: false,
      type: 'violation'
    };

    this.accessLogs.push(log);

    console.error(`[RolePrivacySystem] Security violation: ${violationType} by player ${playerId}`, details);

    // 保持日志数量在合理范围内
    if (this.accessLogs.length > 1000) {
      this.accessLogs.shift();
    }
  }

  /**
   * 获取访问日志
   * @param {object} options - 过滤选项 {limit, type, playerId}
   * @returns {Array}
   */
  getAccessLogs(options = {}) {
    const { limit = 100, type = null, playerId = null } = options;
    
    let logs = [...this.accessLogs];
    
    // 按类型过滤
    if (type) {
      logs = logs.filter(log => log.type === type);
    }
    
    // 按玩家ID过滤
    if (playerId) {
      logs = logs.filter(log => log.playerId === playerId);
    }
    
    return logs.slice(-limit);
  }

  /**
   * 获取安全违规日志
   * @param {number} limit - 返回的日志数量限制
   * @returns {Array}
   */
  getSecurityViolations(limit = 100) {
    return this.getAccessLogs({ limit, type: 'violation' });
  }

  /**
   * 清除访问日志
   */
  clearAccessLogs() {
    this.accessLogs = [];
  }

  /**
   * 解除隐私保护（游戏结束时）
   */
  revealAllRoles() {
    console.log('[RolePrivacySystem] Revealing all roles (game ended)');
    this.disablePrivacyProtection();
  }

  /**
   * 重置系统
   */
  reset() {
    this.isEnabled = false;
    this.currentMode = GameMode.STORYTELLER;
    this.accessLogs = [];
    console.log('[RolePrivacySystem] System reset');
  }
}

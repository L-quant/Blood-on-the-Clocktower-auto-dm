/**
 * Privacy Vuex模块
 * 管理无说书人模式下的隐私保护状态
 */

import { GameMode } from '@/automation/core/RolePrivacySystem';
import PlayerAuthenticator from '@/automation/core/PlayerAuthenticator';
import SeatManager from '@/automation/core/SeatManager';

const playerAuthenticator = new PlayerAuthenticator();
const seatManager = new SeatManager();

const state = () => ({
  gameMode: GameMode.STORYTELLER,
  myPlayerId: null,
  mySeatIndex: -1,
  myToken: null,
  isPrivacyEnabled: false,
  accessLogs: [],
  seatBindings: [],
  
  // 私密信息
  myRole: null,
  nightInformation: [],
  abilityResults: []
});

const getters = {
  /**
   * 是否为无说书人模式
   */
  isPlayerOnlyMode: (state) => {
    return state.gameMode === GameMode.PLAYER_ONLY;
  },

  /**
   * 是否可以看到指定玩家的角色
   */
  canSeeRole: (state) => (playerId) => {
    // 说书人模式下可以看到所有角色
    if (state.gameMode === GameMode.STORYTELLER) {
      return true;
    }

    // 无说书人模式下只能看到自己的角色
    return state.myPlayerId === playerId;
  },

  /**
   * 获取可见的玩家列表（过滤后）
   */
  getVisiblePlayers: (state, getters, rootState) => {
    const players = rootState.players.players || [];
    
    // 说书人模式下返回所有玩家
    if (state.gameMode === GameMode.STORYTELLER) {
      return players;
    }

    // 无说书人模式下过滤角色信息
    return players.map(player => ({
      ...player,
      role: player.id === state.myPlayerId ? player.role : {},
      roleCardVisible: player.id === state.myPlayerId
    }));
  },

  /**
   * 是否已认领座位
   */
  hasClaimedSeat: (state) => {
    return state.mySeatIndex >= 0 && state.myToken !== null;
  },

  /**
   * 获取可用座位列表
   */
  availableSeats: () => {
    return seatManager.getAvailableSeats();
  },

  /**
   * 获取所有座位状态
   */
  allSeats: () => {
    return seatManager.getAllSeats();
  }
};

const mutations = {
  /**
   * 设置游戏模式
   */
  SET_GAME_MODE(state, mode) {
    state.gameMode = mode;
    state.isPrivacyEnabled = mode === GameMode.PLAYER_ONLY;
  },

  /**
   * 设置玩家ID
   */
  SET_PLAYER_ID(state, playerId) {
    state.myPlayerId = playerId;
  },

  /**
   * 设置座位索引
   */
  SET_SEAT_INDEX(state, seatIndex) {
    state.mySeatIndex = seatIndex;
  },

  /**
   * 设置令牌
   */
  SET_TOKEN(state, token) {
    state.myToken = token;
  },

  /**
   * 启用隐私保护
   */
  ENABLE_PRIVACY(state) {
    state.isPrivacyEnabled = true;
  },

  /**
   * 禁用隐私保护
   */
  DISABLE_PRIVACY(state) {
    state.isPrivacyEnabled = false;
  },

  /**
   * 添加访问日志
   */
  ADD_ACCESS_LOG(state, log) {
    state.accessLogs.push({
      ...log,
      timestamp: Date.now()
    });

    // 保持日志数量在合理范围内
    if (state.accessLogs.length > 100) {
      state.accessLogs.shift();
    }
  },

  /**
   * 清除访问日志
   */
  CLEAR_ACCESS_LOGS(state) {
    state.accessLogs = [];
  },

  /**
   * 更新座位绑定
   */
  UPDATE_SEAT_BINDINGS(state, bindings) {
    state.seatBindings = bindings;
  },

  /**
   * 重置隐私状态
   */
  RESET_PRIVACY(state) {
    state.myPlayerId = null;
    state.mySeatIndex = -1;
    state.myToken = null;
    state.accessLogs = [];
    state.seatBindings = [];
    state.myRole = null;
    state.nightInformation = [];
    state.abilityResults = [];
  },

  /**
   * 设置我的角色
   */
  SET_MY_ROLE(state, role) {
    state.myRole = role;
  },

  /**
   * 添加夜间信息
   */
  ADD_NIGHT_INFORMATION(state, info) {
    state.nightInformation.push(info);
  },

  /**
   * 设置夜间信息列表
   */
  SET_NIGHT_INFORMATION(state, infoList) {
    state.nightInformation = infoList;
  },

  /**
   * 添加能力结果
   */
  ADD_ABILITY_RESULT(state, result) {
    state.abilityResults.push(result);
  },

  /**
   * 设置能力结果列表
   */
  SET_ABILITY_RESULTS(state, results) {
    state.abilityResults = results;
  },

  /**
   * 清除私密信息
   */
  CLEAR_PRIVATE_INFO(state) {
    state.myRole = null;
    state.nightInformation = [];
    state.abilityResults = [];
  }
};

const actions = {
  /**
   * 认领座位
   */
  async claimSeat({ commit, state }, { playerId, seatIndex }) {
    try {
      // 使用PlayerAuthenticator认领座位
      const result = await playerAuthenticator.claimSeat(playerId, seatIndex);

      if (result.success) {
        // 更新Vuex状态
        commit('SET_PLAYER_ID', playerId);
        commit('SET_SEAT_INDEX', seatIndex);
        commit('SET_TOKEN', result.token);

        // 更新SeatManager
        seatManager.bindPlayerToSeat(playerId, seatIndex);

        // 更新座位绑定列表
        commit('UPDATE_SEAT_BINDINGS', playerAuthenticator.getAllSeatBindings());

        console.log(`[Privacy] Successfully claimed seat ${seatIndex}`);
        return { success: true };
      } else {
        console.error(`[Privacy] Failed to claim seat: ${result.error}`);
        return { success: false, error: result.error };
      }
    } catch (error) {
      console.error('[Privacy] Error claiming seat:', error);
      return { success: false, error: error.message };
    }
  },

  /**
   * 释放座位
   */
  releaseSeat({ commit, state }) {
    if (state.myPlayerId) {
      playerAuthenticator.releaseSeat(state.myPlayerId);
      seatManager.unbindPlayer(state.myPlayerId);

      commit('RESET_PRIVACY');
      commit('UPDATE_SEAT_BINDINGS', playerAuthenticator.getAllSeatBindings());

      console.log('[Privacy] Released seat');
    }
  },

  /**
   * 切换游戏模式
   */
  switchGameMode({ commit, rootState }, newMode) {
    try {
      // 检查是否可以切换模式（游戏未开始）
      const automation = rootState.automation;
      if (automation && automation.isAutomationEnabled) {
        console.warn('[Privacy] Cannot switch mode while game is running');
        return { success: false, error: 'Game already started' };
      }

      commit('SET_GAME_MODE', newMode);

      console.log(`[Privacy] Switched to ${newMode} mode`);
      return { success: true };
    } catch (error) {
      console.error('[Privacy] Error switching mode:', error);
      return { success: false, error: error.message };
    }
  },

  /**
   * 处理断线重连
   */
  async handleReconnection({ commit }, { playerId, token }) {
    try {
      const result = await playerAuthenticator.handleReconnection(playerId, token);

      if (result.success) {
        commit('SET_PLAYER_ID', playerId);
        commit('SET_SEAT_INDEX', result.seatIndex);
        commit('SET_TOKEN', token);

        console.log('[Privacy] Reconnection successful');
        return { success: true };
      } else {
        console.error(`[Privacy] Reconnection failed: ${result.error}`);
        return { success: false, error: result.error };
      }
    } catch (error) {
      console.error('[Privacy] Error during reconnection:', error);
      return { success: false, error: error.message };
    }
  },

  /**
   * 清理过期令牌
   */
  cleanupExpiredTokens({ commit }) {
    const cleanedCount = playerAuthenticator.cleanupExpiredTokens();
    commit('UPDATE_SEAT_BINDINGS', playerAuthenticator.getAllSeatBindings());
    return cleanedCount;
  },

  /**
   * 记录访问尝试
   */
  logAccess({ commit }, { playerId, action, targetPlayerId }) {
    commit('ADD_ACCESS_LOG', {
      playerId,
      action,
      targetPlayerId
    });
  }
};

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions
};

/**
 * 游戏状态管理器
 * 负责管理游戏的整体状态和阶段转换
 */

import { IGameStateManager } from '../interfaces/IAutomatedStorytellerSystem';
import { GameState, GamePhase, Player } from '../types/GameTypes';
import { SystemStatus } from '../types/AutomationTypes';
import { deepClone } from '../utils/GameUtils';

export default class GameStateManager extends IGameStateManager {
  constructor(store) {
    super();
    this.store = store;
    this.currentState = new GameState();
    this.stateHistory = [];
    this.maxHistorySize = 50;
    
    // 定义有效的状态转换
    this.validTransitions = new Map([
      [GamePhase.SETUP, [GamePhase.FIRST_NIGHT]],
      [GamePhase.FIRST_NIGHT, [GamePhase.DAY]],
      [GamePhase.DAY, [GamePhase.NIGHT, GamePhase.ENDED]],
      [GamePhase.NIGHT, [GamePhase.DAY, GamePhase.ENDED]],
      [GamePhase.ENDED, []] // 游戏结束后不能转换到其他状态
    ]);
  }

  /**
   * 获取当前游戏状态
   * @returns {GameState}
   */
  getCurrentState() {
    return deepClone(this.currentState);
  }

  /**
   * 转换游戏阶段
   * @param {GamePhase} phase 目标阶段
   * @returns {Promise<void>}
   */
  async transitionToPhase(phase) {
    const currentPhase = this.currentState.phase;
    
    // 验证转换的合法性
    if (!this.validateTransition(currentPhase, phase)) {
      throw new Error(`Invalid transition from ${currentPhase} to ${phase}`);
    }

    try {
      // 保存当前状态到历史记录
      this.saveStateToHistory();
      
      // 执行阶段转换前的处理
      await this.beforePhaseTransition(currentPhase, phase);
      
      // 更新游戏状态
      this.currentState.phase = phase;
      this.currentState.timestamp = Date.now();
      
      // 特殊阶段处理
      if (phase === GamePhase.DAY && currentPhase !== GamePhase.SETUP) {
        this.currentState.day += 1;
      }
      
      // 执行阶段转换后的处理
      await this.afterPhaseTransition(currentPhase, phase);
      
      // 同步状态到Vuex store
      this.syncToStore();
      
      // 记录日志
      this.logStateTransition(currentPhase, phase);
      
    } catch (error) {
      // 转换失败，回滚状态
      this.rollbackToPreviousState();
      throw new Error(`Phase transition failed: ${error.message}`);
    }
  }

  /**
   * 更新玩家状态
   * @param {string} playerId 玩家ID
   * @param {object} state 状态更新
   */
  updatePlayerState(playerId, state) {
    const player = this.currentState.players.find(p => p.id === playerId);
    if (!player) {
      throw new Error(`Player not found: ${playerId}`);
    }

    // 保存状态到历史记录
    this.saveStateToHistory();

    // 更新玩家状态
    Object.assign(player, state);
    this.currentState.timestamp = Date.now();

    // 同步到store
    this.syncToStore();

    // 记录日志
    this.store.commit('automation/ADD_LOG', {
      level: 'debug',
      message: `Player ${player.name} state updated`,
      data: { playerId, state }
    });
  }

  /**
   * 验证状态转换的合法性
   * @param {GamePhase} from 源阶段
   * @param {GamePhase} to 目标阶段
   * @returns {boolean}
   */
  validateTransition(from, to) {
    const validTargets = this.validTransitions.get(from);
    return validTargets && validTargets.includes(to);
  }

  /**
   * 回滚到上一个状态
   */
  rollbackToPreviousState() {
    if (this.stateHistory.length === 0) {
      throw new Error('No previous state to rollback to');
    }

    const previousState = this.stateHistory.pop();
    this.currentState = deepClone(previousState);
    this.syncToStore();

    this.store.commit('automation/ADD_LOG', {
      level: 'warn',
      message: 'Game state rolled back to previous state'
    });
  }

  /**
   * 初始化游戏状态
   * @param {GameConfiguration} config 游戏配置
   * @param {Player[]} players 玩家列表
   */
  initializeGame(config, players) {
    this.currentState = new GameState();
    this.currentState.gameId = this.generateGameId();
    this.currentState.gameConfiguration = config;
    this.currentState.players = players.map(p => deepClone(p));
    this.currentState.phase = GamePhase.SETUP;
    this.currentState.day = 0;
    this.currentState.timestamp = Date.now();

    // 清空历史记录
    this.stateHistory = [];

    // 同步到store
    this.syncToStore();

    this.store.commit('automation/ADD_LOG', {
      level: 'info',
      message: `Game initialized with ${players.length} players`,
      data: { gameId: this.currentState.gameId }
    });
  }

  /**
   * 添加玩家到游戏
   * @param {Player} player 玩家
   */
  addPlayer(player) {
    if (this.currentState.phase !== GamePhase.SETUP) {
      throw new Error('Cannot add players after game has started');
    }

    if (this.currentState.players.find(p => p.id === player.id)) {
      throw new Error(`Player already exists: ${player.id}`);
    }

    this.saveStateToHistory();
    this.currentState.players.push(deepClone(player));
    this.currentState.timestamp = Date.now();
    this.syncToStore();

    this.store.commit('automation/ADD_LOG', {
      level: 'info',
      message: `Player ${player.name} added to game`
    });
  }

  /**
   * 移除玩家从游戏
   * @param {string} playerId 玩家ID
   */
  removePlayer(playerId) {
    if (this.currentState.phase !== GamePhase.SETUP) {
      throw new Error('Cannot remove players after game has started');
    }

    const playerIndex = this.currentState.players.findIndex(p => p.id === playerId);
    if (playerIndex === -1) {
      throw new Error(`Player not found: ${playerId}`);
    }

    this.saveStateToHistory();
    const removedPlayer = this.currentState.players.splice(playerIndex, 1)[0];
    this.currentState.timestamp = Date.now();
    this.syncToStore();

    this.store.commit('automation/ADD_LOG', {
      level: 'info',
      message: `Player ${removedPlayer.name} removed from game`
    });
  }

  /**
   * 获取存活玩家列表
   * @returns {Player[]}
   */
  getAlivePlayers() {
    return this.currentState.players.filter(p => p.isAlive);
  }

  /**
   * 获取死亡玩家列表
   * @returns {Player[]}
   */
  getDeadPlayers() {
    return this.currentState.players.filter(p => !p.isAlive);
  }

  /**
   * 获取恶人玩家列表
   * @returns {Player[]}
   */
  getEvilPlayers() {
    return this.currentState.players.filter(p => p.isEvil);
  }

  /**
   * 获取好人玩家列表
   * @returns {Player[]}
   */
  getGoodPlayers() {
    return this.currentState.players.filter(p => !p.isEvil);
  }

  // 私有方法

  /**
   * 阶段转换前的处理
   * @param {GamePhase} from 源阶段
   * @param {GamePhase} to 目标阶段
   */
  async beforePhaseTransition(from, to) {
    // 可以在这里添加阶段转换前的特殊逻辑
    this.store.commit('automation/ADD_LOG', {
      level: 'debug',
      message: `Preparing transition from ${from} to ${to}`
    });
  }

  /**
   * 阶段转换后的处理
   * @param {GamePhase} from 源阶段
   * @param {GamePhase} to 目标阶段
   */
  async afterPhaseTransition(from, to) {
    // 更新store中的当前阶段
    this.store.commit('automation/SET_CURRENT_PHASE', to);
    
    // 可以在这里添加阶段转换后的特殊逻辑
    this.store.commit('automation/ADD_LOG', {
      level: 'debug',
      message: `Completed transition from ${from} to ${to}`
    });
  }

  /**
   * 保存当前状态到历史记录
   */
  saveStateToHistory() {
    this.stateHistory.push(deepClone(this.currentState));
    
    // 限制历史记录大小
    if (this.stateHistory.length > this.maxHistorySize) {
      this.stateHistory.shift();
    }
  }

  /**
   * 同步状态到Vuex store
   */
  syncToStore() {
    // 这里可以将游戏状态同步到现有的Vuex store模块
    // 例如更新players模块和session模块
    
    if (this.store) {
      // 更新玩家信息到players模块
      this.store.commit('players/set', this.currentState.players);
      
      // 更新会话信息到session模块
      this.store.commit('session/setPlayerCount', this.currentState.players.length);
    }
  }

  /**
   * 记录状态转换日志
   * @param {GamePhase} from 源阶段
   * @param {GamePhase} to 目标阶段
   */
  logStateTransition(from, to) {
    this.store.commit('automation/ADD_LOG', {
      level: 'info',
      message: `Game phase transitioned from ${from} to ${to}`,
      data: {
        from,
        to,
        day: this.currentState.day,
        playerCount: this.currentState.players.length,
        aliveCount: this.getAlivePlayers().length
      }
    });
  }

  /**
   * 生成游戏ID
   * @returns {string}
   */
  generateGameId() {
    return 'game_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now().toString(36);
  }
}
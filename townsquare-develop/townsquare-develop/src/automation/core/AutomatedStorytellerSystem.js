/**
 * 自动化说书人系统主控制器
 * 协调所有子系统的工作，管理游戏的完整生命周期
 */

import GameStateManager from './GameStateManager';
import RoleAssigner from './RoleAssigner';
import NightActionProcessor from './NightActionProcessor';
import VotingManager from './VotingManager';
import VictoryConditionChecker from './VictoryConditionChecker';
import AIDecisionEngine from './AIDecisionEngine';
import AbilityResolver from './AbilityResolver';
import AbilityExecutor from './AbilityExecutor';
import StateSynchronizer from './StateSynchronizer';
import ConfigurationManager from './ConfigurationManager';
import RolePrivacySystem from './RolePrivacySystem';
import { SystemStatus } from '../types/AutomationTypes';
import { GamePhase } from '../types/GameTypes';

/**
 * 自动化说书人系统类
 */
export default class AutomatedStorytellerSystem {
  constructor() {
    // 系统状态
    this.status = SystemStatus.IDLE;
    this.isPaused = false;
    this.isInitialized = false;

    // 核心组件
    this.configManager = new ConfigurationManager();
    this.gameStateManager = new GameStateManager();
    this.roleAssigner = new RoleAssigner();
    this.abilityResolver = new AbilityResolver();
    this.abilityExecutor = new AbilityExecutor(this.abilityResolver);
    this.nightActionProcessor = null; // 需要在初始化时创建
    this.votingManager = null; // 需要在初始化时创建
    this.victoryConditionChecker = new VictoryConditionChecker();
    this.aiDecisionEngine = null; // 需要在初始化时创建
    this.stateSynchronizer = null; // 可选，用于多人游戏
    this.rolePrivacySystem = null; // 隐私保护系统

    // 当前配置
    this.currentConfig = null;

    // 错误处理
    this.errorHandlers = [];
    this.emergencyHandlers = [];
  }

  /**
   * 初始化系统
   * @param {object} gameConfig 游戏配置
   * @returns {Promise<void>}
   */
  async initialize(gameConfig) {
    try {
      this.status = SystemStatus.INITIALIZING;
      this.configManager.log('info', 'Initializing Automated Storyteller System', gameConfig);

      // 创建并验证配置
      this.currentConfig = this.configManager.createConfig(gameConfig);

      // 检查游戏模式，如果是无说书人模式，初始化隐私保护系统
      if (gameConfig.gameMode === 'player-only') {
        this.rolePrivacySystem = new RolePrivacySystem(this.gameStateManager);
        this.rolePrivacySystem.enablePrivacyProtection();
        this.configManager.log('info', 'Privacy protection enabled for player-only mode');
      }

      // 初始化AI决策引擎
      this.aiDecisionEngine = new AIDecisionEngine(
        this.gameStateManager,
        this.currentConfig.aiDifficulty
      );

      // 初始化夜间行动处理器
      this.nightActionProcessor = new NightActionProcessor(
        this.gameStateManager,
        this.abilityExecutor,
        this.victoryConditionChecker
      );

      // 初始化投票管理器
      this.votingManager = new VotingManager(
        this.gameStateManager,
        this.victoryConditionChecker
      );

      // 如果需要，初始化状态同步器
      if (gameConfig.enableSync) {
        this.stateSynchronizer = new StateSynchronizer(this.gameStateManager);
      }

      this.isInitialized = true;
      this.status = SystemStatus.IDLE;
      this.configManager.log('info', 'System initialized successfully');

      return { success: true, message: 'System initialized' };
    } catch (error) {
      this.status = SystemStatus.ERROR;
      this.configManager.log('error', 'Initialization failed', error);
      throw error;
    }
  }

  /**
   * 启动自动化游戏
   * @param {Array} players 玩家列表
   * @returns {Promise<object>}
   */
  async startAutomatedGame(players) {
    try {
      if (!this.isInitialized) {
        throw new Error('System not initialized. Call initialize() first.');
      }

      if (this.status === SystemStatus.RUNNING) {
        throw new Error('Game is already running');
      }

      this.status = SystemStatus.RUNNING;
      this.configManager.log('info', 'Starting automated game', { playerCount: players.length });

      // 1. 初始化游戏状态
      const gamePlayers = players.map(p => ({
        id: p.id || `player-${Math.random()}`,
        name: p.name,
        isAlive: true,
        isEvil: false,
        role: null,
        abilities: [],
        status: {},
        votes: 0,
        ghostVoteUsed: false
      }));

      await this.gameStateManager.initializeGame(
        {
          scriptType: this.currentConfig.scriptType,
          playerCount: players.length
        },
        gamePlayers
      );

      // 2. 分配角色
      const roleAssignments = await this._assignRoles();
      this.configManager.log('info', 'Roles assigned', { count: roleAssignments.length });

      // 3. 转换到第一夜
      await this.gameStateManager.transitionToPhase(GamePhase.FIRST_NIGHT);

      // 4. 处理第一夜
      await this._processFirstNight();

      // 5. 开始游戏循环
      await this._startGameLoop();

      return {
        success: true,
        message: 'Game started successfully',
        gameState: this.gameStateManager.getCurrentState()
      };
    } catch (error) {
      this.status = SystemStatus.ERROR;
      this.configManager.log('error', 'Failed to start game', error);
      throw error;
    }
  }

  /**
   * 暂停自动化
   */
  pauseAutomation() {
    if (this.status === SystemStatus.RUNNING) {
      this.isPaused = true;
      this.status = SystemStatus.PAUSED;
      this.configManager.log('info', 'Automation paused');
    }
  }

  /**
   * 恢复自动化
   */
  resumeAutomation() {
    if (this.status === SystemStatus.PAUSED) {
      this.isPaused = false;
      this.status = SystemStatus.RUNNING;
      this.configManager.log('info', 'Automation resumed');
    }
  }

  /**
   * 停止游戏
   */
  stopGame() {
    this.status = SystemStatus.IDLE;
    this.isPaused = false;
    this.configManager.log('info', 'Game stopped');
  }

  /**
   * 获取当前系统状态
   * @returns {object}
   */
  getSystemStatus() {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 如果启用了隐私保护，返回过滤后的状态
    let filteredGameState = gameState;
    if (this.rolePrivacySystem && this.rolePrivacySystem.isPrivacyEnabled()) {
      // 这里返回系统全知状态，具体的过滤由客户端请求时进行
      filteredGameState = gameState;
    }
    
    return {
      status: this.status,
      isPaused: this.isPaused,
      isInitialized: this.isInitialized,
      currentConfig: this.currentConfig,
      gameState: filteredGameState,
      privacyEnabled: this.rolePrivacySystem ? this.rolePrivacySystem.isPrivacyEnabled() : false,
      timestamp: Date.now()
    };
  }

  /**
   * 处理紧急情况
   * @param {string} emergencyType 紧急情况类型
   */
  handleEmergency(emergencyType) {
    this.configManager.log('warn', 'Emergency triggered', { type: emergencyType });

    switch (emergencyType) {
      case 'pause':
        this.pauseAutomation();
        break;
      case 'stop':
        this.stopGame();
        break;
      case 'rollback':
        this.gameStateManager.rollbackToPreviousState();
        break;
      default:
        this.configManager.log('error', 'Unknown emergency type', { type: emergencyType });
    }

    // 调用注册的紧急处理器
    this.emergencyHandlers.forEach(handler => {
      try {
        handler(emergencyType);
      } catch (error) {
        this.configManager.log('error', 'Emergency handler failed', error);
      }
    });
  }

  /**
   * 注册错误处理器
   * @param {Function} handler 错误处理函数
   */
  onError(handler) {
    this.errorHandlers.push(handler);
  }

  /**
   * 注册紧急情况处理器
   * @param {Function} handler 紧急情况处理函数
   */
  onEmergency(handler) {
    this.emergencyHandlers.push(handler);
  }

  /**
   * 获取AI决策建议
   * @param {object} context 决策上下文
   * @returns {Array} 决策建议列表
   */
  getAIDecisionSuggestions(context) {
    if (!this.aiDecisionEngine) {
      throw new Error('AI Decision Engine not initialized');
    }

    return this.aiDecisionEngine.generateDecisionSuggestions(context);
  }

  /**
   * 获取完整的决策报告
   * @param {object} context 决策上下文
   * @returns {object} 决策报告
   */
  getAIDecisionReport(context) {
    if (!this.aiDecisionEngine) {
      throw new Error('AI Decision Engine not initialized');
    }

    return this.aiDecisionEngine.getDecisionReport(context);
  }

  // 私有方法

  /**
   * 分配角色
   * @private
   */
  async _assignRoles() {
    const gameState = this.gameStateManager.getCurrentState();
    const playerCount = gameState.players.length;

    // 生成角色组合
    const roleComposition = this.roleAssigner.generateRoleComposition(
      playerCount,
      this.currentConfig.scriptType
    );

    // 分配角色给玩家
    const assignments = await this.roleAssigner.assignRolesToPlayers(
      gameState.players,
      roleComposition
    );

    // 更新游戏状态中的玩家角色
    assignments.forEach(assignment => {
      const player = gameState.players.find(p => p.id === assignment.playerId);
      if (player) {
        player.role = assignment.role;
        player.isEvil = assignment.role.team === 'evil' || assignment.role.team === 'demon' || assignment.role.team === 'minion';
        player.abilities = assignment.role.ability ? [assignment.role.ability] : [];
      }
    });

    return assignments;
  }

  /**
   * 处理第一夜
   * @private
   */
  async _processFirstNight() {
    this.configManager.log('info', 'Processing first night');

    try {
      await this.nightActionProcessor.startNightPhase();
      const result = await this.nightActionProcessor.completeNightPhase();

      this.configManager.log('info', 'First night completed', result);

      return result;
    } catch (error) {
      this.configManager.log('error', 'First night processing failed', error);
      throw error;
    }
  }

  /**
   * 开始游戏循环
   * @private
   */
  async _startGameLoop() {
    this.configManager.log('info', 'Starting game loop');

    while (this.status === SystemStatus.RUNNING && !this.isPaused) {
      const gameState = this.gameStateManager.getCurrentState();

      // 检查游戏是否结束
      if (gameState.phase === GamePhase.ENDED) {
        this.configManager.log('info', 'Game ended');
        this.status = SystemStatus.IDLE;
        break;
      }

      // 根据当前阶段执行相应操作
      try {
        if (gameState.phase === GamePhase.DAY) {
          await this._processDayPhase();
        } else if (gameState.phase === GamePhase.NIGHT) {
          await this._processNightPhase();
        }

        // 短暂延迟，避免过快循环
        await this._delay(100);
      } catch (error) {
        this.configManager.log('error', 'Game loop error', error);
        this._handleError(error);
      }
    }
  }

  /**
   * 处理白天阶段
   * @private
   */
  async _processDayPhase() {
    this.configManager.log('info', 'Processing day phase');

    try {
      // 这里可以添加自动化的白天流程
      // 例如：自动提名、自动投票等
      // 目前保持简单，等待手动操作

      // 如果配置了自动化级别，可以自动处理
      if (this.currentConfig.automationLevel === 'full_auto') {
        await this.votingManager.processDayPhase();
      }
    } catch (error) {
      this.configManager.log('error', 'Day phase processing failed', error);
      throw error;
    }
  }

  /**
   * 处理夜间阶段
   * @private
   */
  async _processNightPhase() {
    this.configManager.log('info', 'Processing night phase');

    try {
      await this.nightActionProcessor.startNightPhase();
      const result = await this.nightActionProcessor.completeNightPhase();

      this.configManager.log('info', 'Night phase completed', result);

      return result;
    } catch (error) {
      this.configManager.log('error', 'Night phase processing failed', error);
      throw error;
    }
  }

  /**
   * 延迟函数
   * @private
   */
  _delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * 处理错误
   * @private
   */
  _handleError(error) {
    this.configManager.log('error', 'Error occurred', error);

    // 调用注册的错误处理器
    this.errorHandlers.forEach(handler => {
      try {
        handler(error);
      } catch (handlerError) {
        this.configManager.log('error', 'Error handler failed', handlerError);
      }
    });

    // 根据错误类型决定是否暂停或停止
    if (error.critical) {
      this.handleEmergency('stop');
    } else {
      this.handleEmergency('pause');
    }
  }

  /**
   * 获取配置管理器
   * @returns {ConfigurationManager}
   */
  getConfigManager() {
    return this.configManager;
  }

  /**
   * 获取游戏状态管理器
   * @returns {GameStateManager}
   */
  getGameStateManager() {
    return this.gameStateManager;
  }

  /**
   * 获取AI决策引擎
   * @returns {AIDecisionEngine}
   */
  getAIDecisionEngine() {
    return this.aiDecisionEngine;
  }

  /**
   * 获取胜负判断器
   * @returns {VictoryConditionChecker}
   */
  getVictoryConditionChecker() {
    return this.victoryConditionChecker;
  }

  /**
   * 获取隐私保护系统
   * @returns {RolePrivacySystem|null}
   */
  getRolePrivacySystem() {
    return this.rolePrivacySystem;
  }

  /**
   * 获取过滤后的游戏状态（用于特定玩家）
   * @param {string} playerId 玩家ID
   * @returns {object} 过滤后的游戏状态
   */
  getFilteredGameState(playerId) {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 如果没有启用隐私保护，返回完整状态
    if (!this.rolePrivacySystem || !this.rolePrivacySystem.isPrivacyEnabled()) {
      return gameState;
    }
    
    // 返回过滤后的状态
    return this.rolePrivacySystem.getFilteredState(playerId);
  }
}

/**
 * 夜间行动处理器
 * 按顺序处理夜间阶段的所有角色能力
 */

import AbilityExecutor from './AbilityExecutor';
import { GamePhase } from '../types/GameTypes';
import { AbilityTiming, AbilityContext } from '../types/AbilityTypes';

/**
 * 夜间行动处理器类
 */
export default class NightActionProcessor {
  constructor(gameStateManager, abilityExecutor, victoryConditionChecker = null, options = {}) {
    this.gameStateManager = gameStateManager;
    this.abilityExecutor = abilityExecutor;
    this.victoryConditionChecker = victoryConditionChecker;
    this.nightActions = [];
    this.currentNightResult = null;
    this.privacyMode = options.privacyMode || false;
  }

  /**
   * 开始夜间阶段
   * @returns {Promise<void>}
   */
  async startNightPhase() {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 验证当前阶段
    if (gameState.phase !== GamePhase.NIGHT && gameState.phase !== GamePhase.FIRST_NIGHT) {
      throw new Error(`Cannot start night phase from ${gameState.phase}`);
    }

    // 清空之前的夜间行动
    this.nightActions = [];
    this.currentNightResult = {
      day: gameState.day,
      phase: gameState.phase,
      actions: [],
      deaths: [],
      statusChanges: [],
      startTime: Date.now(),
      privacyMode: this.privacyMode
    };

    if (this.privacyMode) {
      console.log(`Starting night phase for day ${gameState.day} (Privacy Mode)`);
    } else {
      console.log(`Starting night phase for day ${gameState.day}`);
    }
  }

  /**
   * 处理单个角色的夜间行动
   * @param {Role} role 角色
   * @param {Player} player 玩家
   * @param {object} actionData 行动数据（目标等）
   * @returns {Promise<ActionResult>} 行动结果
   */
  async processRoleAction(role, player, actionData = {}) {
    if (!player.isAlive) {
      return {
        success: false,
        message: 'Player is dead',
        player,
        role
      };
    }

    // 检查玩家是否有能力阻止标记
    if (player.status && player.status.abilityBlocked) {
      return {
        success: false,
        message: 'Ability is blocked',
        player,
        role
      };
    }

    try {
      // 获取角色能力
      const ability = this._getRoleAbility(role, player);
      
      if (!ability) {
        return {
          success: false,
          message: 'No ability found for role',
          player,
          role
        };
      }

      // 创建能力上下文
      const context = new AbilityContext({
        gameState: this.gameStateManager.getCurrentState(),
        player,
        targets: actionData.targets || [],
        phase: this.gameStateManager.getCurrentState().phase,
        day: this.gameStateManager.getCurrentState().day,
        metadata: actionData.metadata || {}
      });

      // 执行能力
      const result = await this.abilityExecutor.executeAbilityWithTransaction(ability, context);

      // 记录行动
      this._recordAction(player, role, result);

      return {
        success: result.isSuccess(),
        result,
        player,
        role
      };

    } catch (error) {
      console.error('Error processing role action:', error);
      return {
        success: false,
        message: error.message,
        error,
        player,
        role
      };
    }
  }

  /**
   * 获取夜间行动顺序
   * @param {Role[]} activeRoles 活跃的角色列表
   * @returns {Role[]} 排序后的角色列表
   */
  getNightOrder(activeRoles) {
    const gameState = this.gameStateManager.getCurrentState();
    const isFirstNight = gameState.phase === GamePhase.FIRST_NIGHT;

    // 根据夜间顺序排序
    return activeRoles.sort((a, b) => {
      const orderA = isFirstNight ? (a.firstNightOrder || a.firstNight || 999) : (a.otherNight || 999);
      const orderB = isFirstNight ? (b.firstNightOrder || b.firstNight || 999) : (b.otherNight || 999);
      
      return orderA - orderB;
    }).filter(role => {
      // 过滤掉没有夜间行动的角色
      const order = isFirstNight ? (role.firstNightOrder || role.firstNight) : role.otherNight;
      return order && order > 0;
    });
  }

  /**
   * 解决能力冲突
   * @param {AbilityConflict[]} conflicts 冲突列表
   * @returns {Resolution[]} 解决方案列表
   */
  resolveAbilityConflicts(conflicts) {
    const resolutions = [];

    for (const conflict of conflicts) {
      const resolution = this._resolveConflict(conflict);
      resolutions.push(resolution);
    }

    return resolutions;
  }

  /**
   * 完成夜间阶段
   * @returns {Promise<NightResult>} 夜间结果
   */
  async completeNightPhase() {
    if (!this.currentNightResult) {
      throw new Error('No active night phase');
    }

    // 收集所有死亡玩家
    const gameState = this.gameStateManager.getCurrentState();
    const deaths = gameState.players.filter(p => 
      !p.isAlive && !this.currentNightResult.deaths.find(d => d.id === p.id)
    );

    this.currentNightResult.deaths.push(...deaths);
    this.currentNightResult.endTime = Date.now();
    this.currentNightResult.duration = this.currentNightResult.endTime - this.currentNightResult.startTime;

    // 清理临时状态
    this._cleanupNightStatus(gameState);

    // 检查胜负条件
    if (this.victoryConditionChecker) {
      const victoryResult = await this.victoryConditionChecker.checkAndEndGame();
      if (victoryResult) {
        this.currentNightResult.gameEnded = true;
        this.currentNightResult.victoryResult = victoryResult;
      }
    }

    const result = { ...this.currentNightResult };
    this.currentNightResult = null;

    console.log(`Night phase completed. ${deaths.length} deaths.`);

    return result;
  }

  /**
   * 转换到白天阶段
   * @returns {Promise<void>}
   */
  async transitionToDayPhase() {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 验证当前阶段
    if (gameState.phase !== GamePhase.NIGHT && gameState.phase !== GamePhase.FIRST_NIGHT) {
      throw new Error(`Cannot transition to day from ${gameState.phase}`);
    }

    // 检查游戏是否已结束
    if (gameState.phase === GamePhase.ENDED) {
      console.log('Game has ended, skipping day phase transition');
      return;
    }

    try {
      // 转换到白天阶段
      await this.gameStateManager.transitionToPhase(GamePhase.DAY);
      
      console.log(`Transitioned to day phase (Day ${gameState.day + 1})`);
    } catch (error) {
      console.error('Error transitioning to day phase:', error);
      throw error;
    }
  }

  /**
   * 处理完整的夜间阶段
   * @param {object} nightActions 夜间行动配置 {playerId: {targets: [], metadata: {}}}
   * @returns {Promise<NightResult>} 夜间结果
   */
  async processNightPhase(nightActions = {}) {
    await this.startNightPhase();

    const gameState = this.gameStateManager.getCurrentState();
    let alivePlayers = gameState.players.filter(p => p.isAlive);

    // 获取所有活跃角色
    let activeRoles = alivePlayers.map(p => p.role);

    // 获取夜间行动顺序
    let orderedRoles = this.getNightOrder(activeRoles);

    // 按顺序处理每个角色的行动
    for (const role of orderedRoles) {
      // 重新获取当前游戏状态（因为之前的行动可能改变了状态）
      const currentState = this.gameStateManager.getCurrentState();
      const player = currentState.players.find(p => p.role.id === role.id);
      
      if (!player) {
        continue;
      }

      // 检查玩家是否在本轮中死亡
      if (!player.isAlive) {
        console.log(`Player ${player.name} died during night, skipping remaining action`);
        continue;
      }

      // 获取该玩家的行动数据
      const actionData = nightActions[player.id] || {};

      // 处理行动
      await this.processRoleAction(role, player, actionData);

      // 检查是否有玩家在此行动后死亡，并移除他们的后续行动
      const updatedState = this.gameStateManager.getCurrentState();
      const newDeaths = updatedState.players.filter(p => 
        !p.isAlive && alivePlayers.find(ap => ap.id === p.id)
      );

      if (newDeaths.length > 0) {
        console.log(`${newDeaths.length} player(s) died, removing their remaining actions`);
        
        // 更新活跃玩家列表
        alivePlayers = updatedState.players.filter(p => p.isAlive);
        
        // 重新计算剩余的行动顺序（移除死亡玩家）
        activeRoles = alivePlayers.map(p => p.role);
        orderedRoles = this.getNightOrder(activeRoles);
      }
    }

    // 完成夜间阶段并转换到白天
    const nightResult = await this.completeNightPhase();
    
    // 如果游戏已结束，不转换到白天阶段
    if (!nightResult.gameEnded) {
      // 自动转换到白天阶段
      await this.transitionToDayPhase();
    }
    
    return nightResult;
  }

  /**
   * 获取角色能力
   * @private
   * @param {Role} role 角色
   * @param {Player} player 玩家
   * @returns {Ability} 能力对象
   */
  _getRoleAbility(role, player) {
    // 这里应该从角色定义中获取能力
    // 简化实现：返回角色的ability属性
    if (role.ability) {
      return role.ability;
    }

    // 如果玩家有自定义能力
    if (player.abilities && player.abilities.length > 0) {
      return player.abilities[0];
    }

    return null;
  }

  /**
   * 记录行动
   * @private
   * @param {Player} player 玩家
   * @param {Role} role 角色
   * @param {AbilityResult} result 结果
   */
  _recordAction(player, role, result) {
    const action = {
      player: {
        id: player.id,
        name: player.name
      },
      role: {
        id: role.id,
        name: role.name
      },
      result: {
        status: result.status,
        message: result.message
      },
      timestamp: Date.now(),
      // 在隐私模式下，标记为私密信息
      isPrivate: this.privacyMode
    };

    this.nightActions.push(action);

    if (this.currentNightResult) {
      this.currentNightResult.actions.push(action);
    }
  }

  /**
   * 解决单个冲突
   * @private
   * @param {AbilityConflict} conflict 冲突
   * @returns {Resolution} 解决方案
   */
  _resolveConflict(conflict) {
    const { type, abilities, players } = conflict;

    // 根据冲突类型解决
    switch (type) {
      case 'protection_vs_kill':
        // 保护优先于杀死
        return {
          type,
          winner: abilities.find(a => a.type === 'protection'),
          loser: abilities.find(a => a.type === 'killing'),
          reason: 'Protection takes priority over killing'
        };

      case 'drunk_vs_ability':
        // 醉酒使能力无效
        return {
          type,
          winner: abilities.find(a => a.type === 'drunk'),
          loser: abilities.find(a => a.type !== 'drunk'),
          reason: 'Drunk status invalidates ability'
        };

      case 'poison_vs_ability':
        // 中毒使能力无效
        return {
          type,
          winner: abilities.find(a => a.type === 'poison'),
          loser: abilities.find(a => a.type !== 'poison'),
          reason: 'Poison status invalidates ability'
        };

      case 'priority_conflict':
        // 按优先级排序
        const sorted = abilities.sort((a, b) => (b.priority || 0) - (a.priority || 0));
        return {
          type,
          winner: sorted[0],
          loser: sorted[1],
          reason: 'Higher priority ability wins'
        };

      default:
        // 默认：第一个能力获胜
        return {
          type,
          winner: abilities[0],
          loser: abilities[1],
          reason: 'Default resolution: first ability wins'
        };
    }
  }

  /**
   * 清理夜间状态
   * @private
   * @param {GameState} gameState 游戏状态
   */
  _cleanupNightStatus(gameState) {
    // 清理一次性状态效果
    gameState.players.forEach(player => {
      if (player.status) {
        // 清理保护状态
        if (player.status.protected && player.status.protectedUntil === 'one_night') {
          delete player.status.protected;
          delete player.status.protectedUntil;
        }

        // 清理能力阻止状态
        if (player.status.abilityBlocked) {
          delete player.status.abilityBlocked;
        }
      }
    });
  }

  /**
   * 获取夜间行动历史
   * @param {string} playerId 玩家ID（可选，用于隐私模式）
   * @returns {Array} 行动历史
   */
  getNightActionHistory(playerId = null) {
    // 如果是隐私模式且指定了玩家ID，只返回该玩家相关的行动
    if (this.privacyMode && playerId) {
      return this.nightActions.filter(action => 
        action.player.id === playerId || !action.isPrivate
      );
    }
    
    return [...this.nightActions];
  }

  /**
   * 清除夜间行动历史
   */
  clearNightActionHistory() {
    this.nightActions = [];
  }

  /**
   * 获取当前夜间结果
   * @param {string} playerId 玩家ID（可选，用于隐私模式）
   * @returns {object|null} 当前夜间结果
   */
  getCurrentNightResult(playerId = null) {
    if (!this.currentNightResult) {
      return null;
    }
    
    const result = { ...this.currentNightResult };
    
    // 如果是隐私模式且指定了玩家ID，过滤行动
    if (this.privacyMode && playerId) {
      result.actions = result.actions.filter(action => 
        action.player.id === playerId || !action.isPrivate
      );
    }
    
    return result;
  }

  /**
   * 检查是否有活跃的夜间阶段
   * @returns {boolean}
   */
  hasActiveNightPhase() {
    return this.currentNightResult !== null;
  }

  /**
   * 获取夜间行动统计
   * @returns {object} 统计信息
   */
  getNightActionStats() {
    const total = this.nightActions.length;
    const successful = this.nightActions.filter(a => a.result.status === 'success').length;
    const failed = total - successful;

    const roleUsage = {};
    this.nightActions.forEach(action => {
      const roleId = action.role.id;
      if (!roleUsage[roleId]) {
        roleUsage[roleId] = { total: 0, successful: 0, failed: 0 };
      }
      roleUsage[roleId].total++;
      if (action.result.status === 'success') {
        roleUsage[roleId].successful++;
      } else {
        roleUsage[roleId].failed++;
      }
    });

    return {
      total,
      successful,
      failed,
      successRate: total > 0 ? (successful / total * 100).toFixed(2) : 0,
      roleUsage
    };
  }

  /**
   * 验证夜间行动配置
   * @param {object} nightActions 夜间行动配置
   * @returns {object} {valid: boolean, errors: string[]}
   */
  validateNightActions(nightActions) {
    const errors = [];
    const gameState = this.gameStateManager.getCurrentState();

    Object.keys(nightActions).forEach(playerId => {
      const player = gameState.players.find(p => p.id === playerId);
      
      if (!player) {
        errors.push(`Player ${playerId} not found`);
        return;
      }

      if (!player.isAlive) {
        errors.push(`Player ${playerId} is dead`);
      }

      const actionData = nightActions[playerId];
      
      if (actionData.targets) {
        actionData.targets.forEach((target, index) => {
          if (!target || !target.id) {
            errors.push(`Invalid target at index ${index} for player ${playerId}`);
          }
        });
      }
    });

    return {
      valid: errors.length === 0,
      errors
    };
  }
}

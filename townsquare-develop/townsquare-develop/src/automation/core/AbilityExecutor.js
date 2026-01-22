/**
 * 能力执行管理器
 * 处理能力对游戏状态的影响，实现事务性执行和错误恢复
 */

import AbilityResolver from './AbilityResolver';
import { AbilityResultStatus } from '../types/AbilityTypes';

/**
 * 能力执行管理器类
 */
export default class AbilityExecutor {
  constructor(gameStateManager, abilityResolver) {
    this.gameStateManager = gameStateManager;
    this.abilityResolver = abilityResolver || new AbilityResolver(gameStateManager);
    this.executionHistory = [];
    this.maxHistorySize = 100;
  }

  /**
   * 执行能力（带事务性）
   * @param {Ability} ability 能力对象
   * @param {AbilityContext} context 能力上下文
   * @returns {Promise<AbilityResult>} 执行结果
   */
  async executeAbilityWithTransaction(ability, context) {
    // 保存当前状态快照
    const stateSnapshot = this._createStateSnapshot();

    try {
      // 解析能力
      const parsedAbility = this.abilityResolver.parseAbility(ability, context);

      // 执行能力
      const result = await this.abilityResolver.executeAbility(parsedAbility);

      // 如果执行成功，应用状态变更
      if (result.isSuccess()) {
        await this._applyStateChanges(result);
        this._recordExecution(ability, context, result, true);
        return result;
      } else {
        // 如果执行失败，回滚状态
        this._rollbackState(stateSnapshot);
        this._recordExecution(ability, context, result, false);
        return result;
      }

    } catch (error) {
      // 发生错误时回滚状态
      console.error('Error executing ability:', error);
      this._rollbackState(stateSnapshot);
      
      // 记录错误
      this._recordExecution(ability, context, null, false, error);

      // 返回失败结果
      return {
        status: AbilityResultStatus.FAILED,
        ability,
        player: context.player,
        message: `Execution failed: ${error.message}`,
        error
      };
    }
  }

  /**
   * 批量执行能力
   * @param {Array} abilityExecutions 能力执行列表 [{ability, context}]
   * @returns {Promise<Array>} 执行结果列表
   */
  async executeBatch(abilityExecutions) {
    const results = [];

    for (const execution of abilityExecutions) {
      try {
        const result = await this.executeAbilityWithTransaction(
          execution.ability,
          execution.context
        );
        results.push(result);
      } catch (error) {
        console.error('Error in batch execution:', error);
        results.push({
          status: AbilityResultStatus.FAILED,
          ability: execution.ability,
          player: execution.context.player,
          message: `Batch execution failed: ${error.message}`,
          error
        });
      }
    }

    return results;
  }

  /**
   * 应用状态变更到游戏状态
   * @private
   * @param {AbilityResult} result 能力执行结果
   */
  async _applyStateChanges(result) {
    if (!result.effects || result.effects.length === 0) {
      return;
    }

    const gameState = this.gameStateManager.getCurrentState();

    // 处理每个效果的状态变更
    for (const effect of result.effects) {
      if (!effect.success) {
        continue;
      }

      // 处理玩家死亡
      if (effect.killedPlayers && effect.killedPlayers.length > 0) {
        effect.killedPlayers.forEach(player => {
          this._updatePlayerInGameState(gameState, player.id, { isAlive: false });
          
          // 添加到死亡玩家列表
          if (!gameState.deadPlayers) {
            gameState.deadPlayers = [];
          }
          if (!gameState.deadPlayers.find(p => p.id === player.id)) {
            gameState.deadPlayers.push(player);
          }
        });
      }

      // 处理玩家复活
      if (effect.resurrectedPlayers && effect.resurrectedPlayers.length > 0) {
        effect.resurrectedPlayers.forEach(player => {
          this._updatePlayerInGameState(gameState, player.id, { isAlive: true });
          
          // 从死亡玩家列表移除
          if (gameState.deadPlayers) {
            gameState.deadPlayers = gameState.deadPlayers.filter(p => p.id !== player.id);
          }
        });
      }

      // 处理玩家状态变更（中毒、醉酒、保护等）
      if (effect.poisonedPlayers) {
        effect.poisonedPlayers.forEach(player => {
          this._updatePlayerStatus(gameState, player.id, { 
            poisoned: true,
            poisonedUntil: effect.duration 
          });
        });
      }

      if (effect.drunkPlayers) {
        effect.drunkPlayers.forEach(player => {
          this._updatePlayerStatus(gameState, player.id, { 
            drunk: true,
            drunkUntil: effect.duration 
          });
        });
      }

      if (effect.protectedPlayers) {
        effect.protectedPlayers.forEach(player => {
          this._updatePlayerStatus(gameState, player.id, { 
            protected: true,
            protectedUntil: effect.duration 
          });
        });
      }

      // 处理角色变更
      if (effect.changedPlayers) {
        effect.changedPlayers.forEach(player => {
          this._updatePlayerInGameState(gameState, player.id, { 
            role: effect.newRole,
            previousRole: player.previousRole 
          });
        });
      }

      // 处理投票修改
      if (effect.modifiedPlayers) {
        effect.modifiedPlayers.forEach(player => {
          this._updatePlayerStatus(gameState, player.id, { 
            voteModifier: effect.modifier 
          });
        });
      }
    }

    // 触发状态更新事件
    this._notifyStateChange(gameState);
  }

  /**
   * 更新游戏状态中的玩家信息
   * @private
   * @param {GameState} gameState 游戏状态
   * @param {string} playerId 玩家ID
   * @param {object} updates 更新内容
   */
  _updatePlayerInGameState(gameState, playerId, updates) {
    const player = gameState.players.find(p => p.id === playerId);
    if (player) {
      Object.assign(player, updates);
    }
  }

  /**
   * 更新玩家状态
   * @private
   * @param {GameState} gameState 游戏状态
   * @param {string} playerId 玩家ID
   * @param {object} statusUpdates 状态更新
   */
  _updatePlayerStatus(gameState, playerId, statusUpdates) {
    const player = gameState.players.find(p => p.id === playerId);
    if (player) {
      if (!player.status) {
        player.status = {};
      }
      Object.assign(player.status, statusUpdates);
    }
  }

  /**
   * 创建状态快照
   * @private
   * @returns {object} 状态快照
   */
  _createStateSnapshot() {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 深拷贝关键状态
    return {
      players: JSON.parse(JSON.stringify(gameState.players)),
      deadPlayers: JSON.parse(JSON.stringify(gameState.deadPlayers || [])),
      phase: gameState.phase,
      day: gameState.day,
      timestamp: Date.now()
    };
  }

  /**
   * 回滚状态
   * @private
   * @param {object} snapshot 状态快照
   */
  _rollbackState(snapshot) {
    if (!snapshot) {
      console.warn('No snapshot available for rollback');
      return;
    }

    try {
      const gameState = this.gameStateManager.getCurrentState();
      
      // 恢复玩家状态
      gameState.players = JSON.parse(JSON.stringify(snapshot.players));
      gameState.deadPlayers = JSON.parse(JSON.stringify(snapshot.deadPlayers));
      gameState.phase = snapshot.phase;
      gameState.day = snapshot.day;

      console.log('State rolled back successfully');
    } catch (error) {
      console.error('Error rolling back state:', error);
    }
  }

  /**
   * 记录执行历史
   * @private
   * @param {Ability} ability 能力
   * @param {AbilityContext} context 上下文
   * @param {AbilityResult} result 结果
   * @param {boolean} success 是否成功
   * @param {Error} error 错误（如果有）
   */
  _recordExecution(ability, context, result, success, error = null) {
    const record = {
      ability: ability.id,
      player: context.player.id,
      targets: context.targets.map(t => t.id),
      success,
      timestamp: Date.now(),
      day: context.gameState.day,
      phase: context.gameState.phase
    };

    if (result) {
      record.status = result.status;
      record.message = result.message;
    }

    if (error) {
      record.error = error.message;
    }

    this.executionHistory.push(record);

    // 限制历史记录大小
    if (this.executionHistory.length > this.maxHistorySize) {
      this.executionHistory.shift();
    }
  }

  /**
   * 通知状态变更
   * @private
   * @param {GameState} gameState 游戏状态
   */
  _notifyStateChange(gameState) {
    // 触发Vuex mutation或事件
    if (this.gameStateManager.notifyStateChange) {
      this.gameStateManager.notifyStateChange(gameState);
    }
  }

  /**
   * 获取执行历史
   * @param {object} filters 过滤条件
   * @returns {Array} 执行历史记录
   */
  getExecutionHistory(filters = {}) {
    let history = [...this.executionHistory];

    if (filters.playerId) {
      history = history.filter(record => record.player === filters.playerId);
    }

    if (filters.abilityId) {
      history = history.filter(record => record.ability === filters.abilityId);
    }

    if (filters.day !== undefined) {
      history = history.filter(record => record.day === filters.day);
    }

    if (filters.success !== undefined) {
      history = history.filter(record => record.success === filters.success);
    }

    return history;
  }

  /**
   * 清除执行历史
   */
  clearExecutionHistory() {
    this.executionHistory = [];
  }

  /**
   * 获取最近的执行记录
   * @param {number} count 数量
   * @returns {Array} 最近的执行记录
   */
  getRecentExecutions(count = 10) {
    return this.executionHistory.slice(-count);
  }

  /**
   * 验证能力执行的前置条件
   * @param {Ability} ability 能力
   * @param {AbilityContext} context 上下文
   * @returns {object} {valid: boolean, errors: string[]}
   */
  validatePreconditions(ability, context) {
    const errors = [];

    // 检查玩家是否存在
    if (!context.player) {
      errors.push('Player is required');
    }

    // 检查游戏状态是否存在
    if (!context.gameState) {
      errors.push('Game state is required');
    }

    // 检查能力是否可用
    if (ability && !this.abilityResolver.validateAbilityConditions(ability, context)) {
      errors.push('Ability conditions not met');
    }

    // 检查目标是否有效
    if (context.targets) {
      context.targets.forEach((target, index) => {
        if (!target || !target.id) {
          errors.push(`Invalid target at index ${index}`);
        }
      });
    }

    return {
      valid: errors.length === 0,
      errors
    };
  }

  /**
   * 尝试恢复失败的能力执行
   * @param {Ability} ability 能力
   * @param {AbilityContext} context 上下文
   * @param {number} maxRetries 最大重试次数
   * @returns {Promise<AbilityResult>} 执行结果
   */
  async executeWithRetry(ability, context, maxRetries = 3) {
    let lastError = null;
    
    for (let attempt = 0; attempt < maxRetries; attempt++) {
      try {
        const result = await this.executeAbilityWithTransaction(ability, context);
        
        if (result.isSuccess()) {
          return result;
        }

        // 如果是被阻止的状态（中毒、醉酒等），不重试
        if (result.isBlocked && result.isBlocked()) {
          return result;
        }

        lastError = new Error(result.message);
        
      } catch (error) {
        lastError = error;
        console.warn(`Attempt ${attempt + 1} failed:`, error.message);
      }

      // 等待一小段时间再重试
      if (attempt < maxRetries - 1) {
        await this._delay(100 * (attempt + 1));
      }
    }

    // 所有重试都失败
    return {
      status: AbilityResultStatus.FAILED,
      ability,
      player: context.player,
      message: `Failed after ${maxRetries} attempts: ${lastError.message}`,
      error: lastError
    };
  }

  /**
   * 延迟函数
   * @private
   * @param {number} ms 毫秒数
   * @returns {Promise}
   */
  _delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * 获取执行统计信息
   * @returns {object} 统计信息
   */
  getExecutionStats() {
    const total = this.executionHistory.length;
    const successful = this.executionHistory.filter(r => r.success).length;
    const failed = total - successful;

    const abilityUsage = {};
    this.executionHistory.forEach(record => {
      if (!abilityUsage[record.ability]) {
        abilityUsage[record.ability] = { total: 0, successful: 0, failed: 0 };
      }
      abilityUsage[record.ability].total++;
      if (record.success) {
        abilityUsage[record.ability].successful++;
      } else {
        abilityUsage[record.ability].failed++;
      }
    });

    return {
      total,
      successful,
      failed,
      successRate: total > 0 ? (successful / total * 100).toFixed(2) : 0,
      abilityUsage
    };
  }
}

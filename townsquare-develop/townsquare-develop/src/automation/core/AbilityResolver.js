/**
 * 能力解析器
 * 解析和执行各种角色能力
 */

import {
  AbilityType,
  AbilityTiming,
  EffectType,
  TargetType,
  AbilityResultStatus,
  AbilityContext,
  AbilityResult
} from '../types/AbilityTypes';
import { Team, GamePhase } from '../types/GameTypes';

/**
 * 能力解析器类
 */
export default class AbilityResolver {
  constructor(gameStateManager) {
    this.gameStateManager = gameStateManager;
    this.effectHandlers = this._initializeEffectHandlers();
  }

  /**
   * 解析角色能力
   * @param {Ability} ability 能力对象
   * @param {AbilityContext} context 能力上下文
   * @returns {object} 解析后的能力
   */
  parseAbility(ability, context) {
    if (!ability) {
      throw new Error('Ability is required');
    }

    if (!context) {
      throw new Error('Context is required');
    }

    // 验证能力使用条件
    const canUse = this.validateAbilityConditions(ability, context);
    
    if (!canUse) {
      return {
        ability,
        context,
        canUse: false,
        reason: 'Ability conditions not met',
        effects: []
      };
    }

    // 解析能力效果
    const parsedEffects = ability.effects.map(effect => ({
      ...effect,
      resolvedTargets: this._resolveTargets(effect.target, context)
    }));

    return {
      ability,
      context,
      canUse: true,
      effects: parsedEffects,
      priority: ability.priority
    };
  }

  /**
   * 执行能力效果
   * @param {object} parsedAbility 解析后的能力
   * @returns {Promise<AbilityResult>} 能力执行结果
   */
  async executeAbility(parsedAbility) {
    const { ability, context, canUse, effects } = parsedAbility;

    if (!canUse) {
      return new AbilityResult({
        status: AbilityResultStatus.INVALID,
        ability,
        player: context.player,
        message: 'Ability cannot be used'
      });
    }

    // 检查玩家状态（中毒、醉酒等）
    const playerStatus = this._checkPlayerStatus(context.player);
    if (playerStatus.blocked) {
      return new AbilityResult({
        status: playerStatus.status,
        ability,
        player: context.player,
        message: playerStatus.message
      });
    }

    try {
      // 执行所有效果
      const executedEffects = [];
      
      for (const effect of effects) {
        const effectResult = await this._executeEffect(effect, context);
        executedEffects.push(effectResult);
      }

      // 标记能力已使用
      if (ability.use) {
        ability.use();
      }

      // 处理副作用
      const result = new AbilityResult({
        status: AbilityResultStatus.SUCCESS,
        ability,
        player: context.player,
        targets: context.targets,
        effects: executedEffects,
        message: 'Ability executed successfully'
      });

      this.handleAbilitySideEffects(result);

      return result;

    } catch (error) {
      console.error('Error executing ability:', error);
      
      return new AbilityResult({
        status: AbilityResultStatus.FAILED,
        ability,
        player: context.player,
        message: `Ability execution failed: ${error.message}`
      });
    }
  }

  /**
   * 验证能力使用条件
   * @param {Ability} ability 能力对象
   * @param {AbilityContext} context 能力上下文
   * @returns {boolean} 是否可以使用
   */
  validateAbilityConditions(ability, context) {
    if (!ability || !context) {
      return false;
    }

    // 使用能力对象的canUse方法
    if (ability.canUse) {
      return ability.canUse(context);
    }

    // 默认验证
    return true;
  }

  /**
   * 处理能力的副作用
   * @param {AbilityResult} result 能力结果
   */
  handleAbilitySideEffects(result) {
    if (!result.isSuccess()) {
      return;
    }

    // 记录能力使用历史
    this._recordAbilityUsage(result);

    // 触发相关事件
    this._triggerAbilityEvents(result);

    // 更新游戏状态
    this._updateGameStateFromAbility(result);
  }

  /**
   * 初始化效果处理器
   * @private
   * @returns {object} 效果处理器映射
   */
  _initializeEffectHandlers() {
    return {
      [EffectType.LEARN_INFO]: this._handleLearnInfo.bind(this),
      [EffectType.KILL]: this._handleKill.bind(this),
      [EffectType.PROTECT]: this._handleProtect.bind(this),
      [EffectType.POISON]: this._handlePoison.bind(this),
      [EffectType.DRUNK]: this._handleDrunk.bind(this),
      [EffectType.MODIFY_VOTE]: this._handleModifyVote.bind(this),
      [EffectType.CHANGE_ROLE]: this._handleChangeRole.bind(this),
      [EffectType.RESURRECT]: this._handleResurrect.bind(this),
      [EffectType.REGISTER_AS]: this._handleRegisterAs.bind(this),
      [EffectType.BLOCK_ABILITY]: this._handleBlockAbility.bind(this)
    };
  }

  /**
   * 执行单个效果
   * @private
   * @param {object} effect 效果对象
   * @param {AbilityContext} context 上下文
   * @returns {Promise<object>} 效果结果
   */
  async _executeEffect(effect, context) {
    const handler = this.effectHandlers[effect.type];
    
    if (!handler) {
      console.warn(`No handler for effect type: ${effect.type}`);
      return {
        type: effect.type,
        success: false,
        message: 'Unknown effect type'
      };
    }

    try {
      const result = await handler(effect, context);
      return {
        type: effect.type,
        success: true,
        ...result
      };
    } catch (error) {
      console.error(`Error executing effect ${effect.type}:`, error);
      return {
        type: effect.type,
        success: false,
        message: error.message
      };
    }
  }

  /**
   * 解析目标
   * @private
   * @param {string} targetType 目标类型
   * @param {AbilityContext} context 上下文
   * @returns {Player[]} 目标玩家列表
   */
  _resolveTargets(targetType, context) {
    const gameState = context.gameState;
    
    switch (targetType) {
      case TargetType.SELF:
        return [context.player];
      
      case TargetType.SINGLE_PLAYER:
        return context.targets.slice(0, 1);
      
      case TargetType.MULTIPLE_PLAYERS:
        return context.targets;
      
      case TargetType.ALL_PLAYERS:
        return gameState.players;
      
      case TargetType.ALIVE_PLAYERS:
        return gameState.players.filter(p => p.isAlive);
      
      case TargetType.DEAD_PLAYERS:
        return gameState.players.filter(p => !p.isAlive);
      
      case TargetType.NEIGHBORS:
        return this._getNeighbors(context.player, gameState);
      
      default:
        return context.targets;
    }
  }

  /**
   * 获取邻居玩家
   * @private
   * @param {Player} player 玩家
   * @param {GameState} gameState 游戏状态
   * @returns {Player[]} 邻居列表
   */
  _getNeighbors(player, gameState) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const playerIndex = alivePlayers.findIndex(p => p.id === player.id);
    
    if (playerIndex === -1) {
      return [];
    }

    const leftIndex = (playerIndex - 1 + alivePlayers.length) % alivePlayers.length;
    const rightIndex = (playerIndex + 1) % alivePlayers.length;

    return [alivePlayers[leftIndex], alivePlayers[rightIndex]];
  }

  /**
   * 检查玩家状态
   * @private
   * @param {Player} player 玩家
   * @returns {object} 状态检查结果
   */
  _checkPlayerStatus(player) {
    if (!player) {
      return { blocked: true, status: AbilityResultStatus.INVALID, message: 'Player not found' };
    }

    if (player.status && player.status.poisoned) {
      return { blocked: true, status: AbilityResultStatus.POISONED, message: 'Player is poisoned' };
    }

    if (player.status && player.status.drunk) {
      return { blocked: true, status: AbilityResultStatus.DRUNK, message: 'Player is drunk' };
    }

    return { blocked: false };
  }

  // ========== 效果处理器 ==========

  /**
   * 处理学习信息效果
   * @private
   */
  async _handleLearnInfo(effect, context) {
    const information = this._generateInformation(effect, context);
    
    return {
      information,
      message: 'Information learned'
    };
  }

  /**
   * 处理杀死效果
   * @private
   */
  async _handleKill(effect, context) {
    const targets = effect.resolvedTargets || [];
    const killedPlayers = [];

    for (const target of targets) {
      // 检查保护状态
      if (target.status && target.status.protected) {
        continue;
      }

      // 检查特殊角色（士兵等）
      if (this._isImmuneToKill(target)) {
        continue;
      }

      target.isAlive = false;
      killedPlayers.push(target);
    }

    return {
      killedPlayers,
      message: `${killedPlayers.length} player(s) killed`
    };
  }

  /**
   * 处理保护效果
   * @private
   */
  async _handleProtect(effect, context) {
    const targets = effect.resolvedTargets || [];
    
    targets.forEach(target => {
      if (!target.status) {
        target.status = {};
      }
      target.status.protected = true;
      target.status.protectedUntil = effect.duration;
    });

    return {
      protectedPlayers: targets,
      message: `${targets.length} player(s) protected`
    };
  }

  /**
   * 处理中毒效果
   * @private
   */
  async _handlePoison(effect, context) {
    const targets = effect.resolvedTargets || [];
    
    targets.forEach(target => {
      if (!target.status) {
        target.status = {};
      }
      target.status.poisoned = true;
      target.status.poisonedUntil = effect.duration;
    });

    return {
      poisonedPlayers: targets,
      message: `${targets.length} player(s) poisoned`
    };
  }

  /**
   * 处理醉酒效果
   * @private
   */
  async _handleDrunk(effect, context) {
    const targets = effect.resolvedTargets || [];
    
    targets.forEach(target => {
      if (!target.status) {
        target.status = {};
      }
      target.status.drunk = true;
      target.status.drunkUntil = effect.duration;
    });

    return {
      drunkPlayers: targets,
      message: `${targets.length} player(s) drunk`
    };
  }

  /**
   * 处理修改投票效果
   * @private
   */
  async _handleModifyVote(effect, context) {
    const targets = effect.resolvedTargets || [];
    const modifier = effect.value || 0;
    
    targets.forEach(target => {
      if (!target.status) {
        target.status = {};
      }
      target.status.voteModifier = modifier;
    });

    return {
      modifiedPlayers: targets,
      modifier,
      message: `Vote modified by ${modifier}`
    };
  }

  /**
   * 处理改变角色效果
   * @private
   */
  async _handleChangeRole(effect, context) {
    const targets = effect.resolvedTargets || [];
    const newRole = effect.value;
    
    if (!newRole) {
      throw new Error('New role not specified');
    }

    targets.forEach(target => {
      target.previousRole = target.role;
      target.role = newRole;
    });

    return {
      changedPlayers: targets,
      newRole,
      message: 'Role changed'
    };
  }

  /**
   * 处理复活效果
   * @private
   */
  async _handleResurrect(effect, context) {
    const targets = effect.resolvedTargets || [];
    const resurrectedPlayers = [];

    targets.forEach(target => {
      if (!target.isAlive) {
        target.isAlive = true;
        resurrectedPlayers.push(target);
      }
    });

    return {
      resurrectedPlayers,
      message: `${resurrectedPlayers.length} player(s) resurrected`
    };
  }

  /**
   * 处理注册为效果（间谍、隐士）
   * @private
   */
  async _handleRegisterAs(effect, context) {
    const targets = effect.resolvedTargets || [];
    const registerAs = effect.value;
    
    targets.forEach(target => {
      if (!target.status) {
        target.status = {};
      }
      target.status.registerAs = registerAs;
    });

    return {
      affectedPlayers: targets,
      registerAs,
      message: `Registered as ${registerAs}`
    };
  }

  /**
   * 处理阻止能力效果
   * @private
   */
  async _handleBlockAbility(effect, context) {
    const targets = effect.resolvedTargets || [];
    
    targets.forEach(target => {
      if (!target.status) {
        target.status = {};
      }
      target.status.abilityBlocked = true;
    });

    return {
      blockedPlayers: targets,
      message: `${targets.length} ability(ies) blocked`
    };
  }

  // ========== 辅助方法 ==========

  /**
   * 生成信息
   * @private
   */
  _generateInformation(effect, context) {
    // 这里应该根据具体的能力类型生成信息
    // 例如：洗衣妇看到两个玩家中有一个是村民
    return {
      type: 'generic',
      data: effect.value,
      timestamp: Date.now()
    };
  }

  /**
   * 检查是否免疫杀死
   * @private
   */
  _isImmuneToKill(player) {
    // 士兵免疫恶魔杀死
    if (player.role && player.role.id === 'soldier') {
      return true;
    }

    return false;
  }

  /**
   * 记录能力使用历史
   * @private
   */
  _recordAbilityUsage(result) {
    // 记录到游戏状态中
    if (this.gameStateManager) {
      const gameState = this.gameStateManager.getCurrentState();
      if (!gameState.abilityHistory) {
        gameState.abilityHistory = [];
      }
      
      gameState.abilityHistory.push({
        ability: result.ability.id,
        player: result.player.id,
        targets: result.targets.map(t => t.id),
        timestamp: Date.now(),
        day: gameState.day
      });
    }
  }

  /**
   * 触发能力事件
   * @private
   */
  _triggerAbilityEvents(result) {
    // 触发相关事件，供其他系统监听
    // 例如：能力使用事件、玩家死亡事件等
  }

  /**
   * 从能力结果更新游戏状态
   * @private
   */
  _updateGameStateFromAbility(result) {
    if (!this.gameStateManager) {
      return;
    }

    // 更新玩家状态
    result.effects.forEach(effect => {
      if (effect.killedPlayers) {
        effect.killedPlayers.forEach(player => {
          this.gameStateManager.updatePlayerState(player.id, { isAlive: false });
        });
      }
    });
  }
}

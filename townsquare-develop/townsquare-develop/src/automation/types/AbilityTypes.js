/**
 * 能力相关的类型定义
 */

/**
 * 能力类型枚举
 */
export const AbilityType = {
  INFORMATION: 'information',      // 信息获取类（洗衣妇、图书管理员等）
  PROTECTION: 'protection',        // 保护类（僧侣、旅店老板等）
  KILLING: 'killing',              // 杀人类（小恶魔、刺客等）
  MANIPULATION: 'manipulation',    // 操控类（投毒者、女巫等）
  DETECTION: 'detection',          // 检测类（占卜师、梦想家等）
  PASSIVE: 'passive',              // 被动类（士兵、隐士等）
  TRIGGERED: 'triggered',          // 触发类（守鸦人、送葬者等）
  VOTING: 'voting'                 // 投票类（屠夫、处女等）
};

/**
 * 能力时机枚举
 */
export const AbilityTiming = {
  SETUP: 'setup',                  // 设置阶段
  FIRST_NIGHT: 'first_night',      // 第一个夜晚
  NIGHT: 'night',                  // 每个夜晚
  DAY: 'day',                      // 白天
  NOMINATION: 'nomination',        // 提名时
  EXECUTION: 'execution',          // 处决时
  DEATH: 'death',                  // 死亡时
  PASSIVE: 'passive'               // 被动持续
};

/**
 * 效果类型枚举
 */
export const EffectType = {
  LEARN_INFO: 'learn_info',        // 学习信息
  KILL: 'kill',                    // 杀死玩家
  PROTECT: 'protect',              // 保护玩家
  POISON: 'poison',                // 中毒
  DRUNK: 'drunk',                  // 醉酒
  MODIFY_VOTE: 'modify_vote',      // 修改投票
  CHANGE_ROLE: 'change_role',      // 改变角色
  RESURRECT: 'resurrect',          // 复活
  REGISTER_AS: 'register_as',      // 注册为（间谍、隐士）
  BLOCK_ABILITY: 'block_ability'   // 阻止能力
};

/**
 * 目标类型枚举
 */
export const TargetType = {
  SELF: 'self',                    // 自己
  SINGLE_PLAYER: 'single_player',  // 单个玩家
  MULTIPLE_PLAYERS: 'multiple_players', // 多个玩家
  ALL_PLAYERS: 'all_players',      // 所有玩家
  NEIGHBORS: 'neighbors',          // 邻居
  DEAD_PLAYERS: 'dead_players',    // 死亡玩家
  ALIVE_PLAYERS: 'alive_players',  // 存活玩家
  TEAM: 'team',                    // 团队
  ROLE_TYPE: 'role_type'           // 角色类型
};

/**
 * 效果持续时间枚举
 */
export const EffectDuration = {
  INSTANT: 'instant',              // 立即
  UNTIL_DUSK: 'until_dusk',        // 直到黄昏
  UNTIL_DAWN: 'until_dawn',        // 直到黎明
  ONE_DAY: 'one_day',              // 一天
  ONE_NIGHT: 'one_night',          // 一晚
  PERMANENT: 'permanent',          // 永久
  CONDITIONAL: 'conditional'       // 条件性
};

/**
 * 能力条件类型枚举
 */
export const AbilityConditionType = {
  PLAYER_ALIVE: 'player_alive',    // 玩家存活
  PLAYER_DEAD: 'player_dead',      // 玩家死亡
  FIRST_USE: 'first_use',          // 首次使用
  PHASE: 'phase',                  // 特定阶段
  ROLE_IN_PLAY: 'role_in_play',    // 角色在场
  PLAYER_COUNT: 'player_count',    // 玩家数量
  DAY_NUMBER: 'day_number',        // 天数
  CUSTOM: 'custom'                 // 自定义条件
};

/**
 * 能力结果状态枚举
 */
export const AbilityResultStatus = {
  SUCCESS: 'success',              // 成功
  FAILED: 'failed',                // 失败
  BLOCKED: 'blocked',              // 被阻止
  POISONED: 'poisoned',            // 中毒（无效）
  DRUNK: 'drunk',                  // 醉酒（无效）
  PROTECTED: 'protected',          // 被保护
  INVALID: 'invalid'               // 无效
};

/**
 * 能力定义类
 */
export class Ability {
  constructor(config) {
    this.id = config.id;
    this.name = config.name;
    this.description = config.description;
    this.type = config.type;
    this.timing = config.timing;
    this.conditions = config.conditions || [];
    this.effects = config.effects || [];
    this.priority = config.priority || 0;
    this.usesRemaining = config.usesRemaining || Infinity;
    this.metadata = config.metadata || {};
  }

  /**
   * 检查能力是否可以使用
   * @param {object} context 能力上下文
   * @returns {boolean}
   */
  canUse(context) {
    if (this.usesRemaining <= 0) {
      return false;
    }

    return this.conditions.every(condition => 
      this._evaluateCondition(condition, context)
    );
  }

  /**
   * 评估单个条件
   * @private
   * @param {object} condition 条件
   * @param {object} context 上下文
   * @returns {boolean}
   */
  _evaluateCondition(condition, context) {
    switch (condition.type) {
      case AbilityConditionType.PLAYER_ALIVE:
        return context.player && context.player.isAlive;
      
      case AbilityConditionType.PLAYER_DEAD:
        return context.player && !context.player.isAlive;
      
      case AbilityConditionType.FIRST_USE:
        return this.usesRemaining === Infinity || this.usesRemaining > 0;
      
      case AbilityConditionType.PHASE:
        return context.gameState && context.gameState.phase === condition.value;
      
      case AbilityConditionType.PLAYER_COUNT:
        return context.gameState && 
               context.gameState.players.filter(p => p.isAlive).length >= condition.value;
      
      case AbilityConditionType.CUSTOM:
        return condition.evaluator ? condition.evaluator(context) : true;
      
      default:
        return true;
    }
  }

  /**
   * 使用能力（减少使用次数）
   */
  use() {
    if (this.usesRemaining !== Infinity) {
      this.usesRemaining--;
    }
  }
}

/**
 * 能力效果类
 */
export class AbilityEffect {
  constructor(config) {
    this.type = config.type;
    this.target = config.target;
    this.value = config.value;
    this.duration = config.duration || EffectDuration.INSTANT;
    this.metadata = config.metadata || {};
  }
}

/**
 * 能力上下文类
 */
export class AbilityContext {
  constructor(config) {
    this.gameState = config.gameState;
    this.player = config.player;
    this.targets = config.targets || [];
    this.phase = config.phase;
    this.day = config.day;
    this.metadata = config.metadata || {};
  }
}

/**
 * 能力结果类
 */
export class AbilityResult {
  constructor(config) {
    this.status = config.status;
    this.ability = config.ability;
    this.player = config.player;
    this.targets = config.targets || [];
    this.effects = config.effects || [];
    this.information = config.information || null;
    this.message = config.message || '';
    this.metadata = config.metadata || {};
  }

  /**
   * 检查结果是否成功
   * @returns {boolean}
   */
  isSuccess() {
    return this.status === AbilityResultStatus.SUCCESS;
  }

  /**
   * 检查结果是否被阻止
   * @returns {boolean}
   */
  isBlocked() {
    return this.status === AbilityResultStatus.BLOCKED ||
           this.status === AbilityResultStatus.POISONED ||
           this.status === AbilityResultStatus.DRUNK ||
           this.status === AbilityResultStatus.PROTECTED;
  }
}

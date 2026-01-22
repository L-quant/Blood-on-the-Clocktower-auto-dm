/**
 * 自动化系统相关类型定义
 */

// 自动化级别枚举
export const AutomationLevel = {
  MANUAL: 'manual',
  SEMI_AUTO: 'semi_auto',
  FULL_AUTO: 'full_auto'
};

// AI难度枚举
export const AIDifficulty = {
  EASY: 'easy',
  MEDIUM: 'medium',
  HARD: 'hard',
  EXPERT: 'expert'
};

// 系统状态枚举
export const SystemStatus = {
  IDLE: 'idle',
  INITIALIZING: 'initializing',
  RUNNING: 'running',
  PAUSED: 'paused',
  ERROR: 'error'
};

// 决策上下文接口
export class DecisionContext {
  constructor(gameState, playerPerspective, availableActions) {
    this.gameState = gameState;
    this.playerPerspective = playerPerspective;
    this.availableActions = availableActions;
    this.timeRemaining = 0;
    this.riskTolerance = 0.5;
  }
}

// 决策建议接口
export class DecisionSuggestion {
  constructor(action, confidence, reasoning) {
    this.action = action;
    this.confidence = confidence; // 0-1
    this.reasoning = reasoning;
    this.expectedOutcome = null;
    this.risks = [];
    this.alternatives = [];
  }
}

// 能力效果接口
export class AbilityEffect {
  constructor(type, target, value) {
    this.type = type;
    this.target = target;
    this.value = value;
    this.duration = 'instant';
  }
}

// 能力解析结果接口
export class AbilityResult {
  constructor(success, effects = [], errors = []) {
    this.success = success;
    this.effects = effects;
    this.errors = errors;
    this.timestamp = Date.now();
  }
}

// 胜利结果接口
export class VictoryResult {
  constructor(isEnded = false, winner = null, reason = '') {
    this.isEnded = isEnded;
    this.winner = winner; // 'good', 'evil', 'draw'
    this.reason = reason;
    this.timestamp = Date.now();
  }
}

// 错误类型枚举
export const ErrorType = {
  NETWORK: 'network',
  VALIDATION: 'validation',
  GAME_LOGIC: 'game_logic',
  SYSTEM: 'system'
};

// 错误恢复策略枚举
export const RecoveryStrategy = {
  RETRY: 'retry',
  ROLLBACK: 'rollback',
  SKIP: 'skip',
  MANUAL_INTERVENTION: 'manual_intervention'
};
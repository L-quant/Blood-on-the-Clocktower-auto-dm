/**
 * 自动化说书人系统主入口
 * 导出所有核心组件和类型
 */

// 核心系统
export { default as AutomatedStorytellerSystem } from './core/AutomatedStorytellerSystem';

// 核心组件
export { default as GameStateManager } from './core/GameStateManager';
export { default as RoleAssigner } from './core/RoleAssigner';
export { default as NightActionProcessor } from './core/NightActionProcessor';
export { default as VotingManager } from './core/VotingManager';
export { default as VictoryConditionChecker } from './core/VictoryConditionChecker';
export { default as AIDecisionEngine } from './core/AIDecisionEngine';
export { default as AbilityResolver } from './core/AbilityResolver';
export { default as AbilityExecutor } from './core/AbilityExecutor';
export { default as StateSynchronizer } from './core/StateSynchronizer';
export { default as ConfigurationManager } from './core/ConfigurationManager';

// 工具类
export { default as GameUtils } from './utils/GameUtils';
export { default as RoleCompositionValidator } from './utils/RoleCompositionValidator';
export { default as DecisionFormatter } from './utils/DecisionFormatter';

// 数据定义
export { default as ScriptDefinitions } from './data/ScriptDefinitions';

// 类型定义
export * from './types/GameTypes';
export * from './types/AutomationTypes';
export * from './types/AbilityTypes';

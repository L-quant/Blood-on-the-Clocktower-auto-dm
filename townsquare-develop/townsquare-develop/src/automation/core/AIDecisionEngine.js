/**
 * AI决策引擎
 * 为恶人阵营提供智能决策支持
 */

import { Team, GamePhase } from '../types/GameTypes';
import DecisionFormatter from '../utils/DecisionFormatter';

/**
 * AI决策引擎类
 */
export default class AIDecisionEngine {
  constructor(gameStateManager, aiDifficulty = 'medium') {
    this.gameStateManager = gameStateManager;
    this.aiDifficulty = aiDifficulty; // 'easy', 'medium', 'hard'
    this.gameHistory = [];
  }

  /**
   * 分析当前游戏状态
   * @param {GameState} gameState 游戏状态
   * @returns {object} 游戏分析结果
   */
  analyzeGameState(gameState) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const aliveGood = alivePlayers.filter(p => !p.isEvil);
    const aliveEvil = alivePlayers.filter(p => p.isEvil);
    const aliveDemons = alivePlayers.filter(p => p.role && p.role.team === Team.DEMON);
    const aliveMinions = alivePlayers.filter(p => p.role && p.role.team === Team.MINION);

    // 计算游戏进度
    const totalPlayers = gameState.players.length;
    const deathRate = (totalPlayers - alivePlayers.length) / totalPlayers;
    const gameProgress = Math.min(gameState.day / 5, 1); // 假设平均5天结束

    // 评估当前局势
    const evilAdvantage = this._calculateEvilAdvantage(aliveGood.length, aliveEvil.length);
    const threatLevel = this._assessThreatLevel(gameState, aliveGood, aliveEvil);
    const suspicionLevels = this._calculateSuspicionLevels(gameState);

    return {
      phase: gameState.phase,
      day: gameState.day,
      playerCounts: {
        total: totalPlayers,
        alive: alivePlayers.length,
        aliveGood: aliveGood.length,
        aliveEvil: aliveEvil.length,
        aliveDemons: aliveDemons.length,
        aliveMinions: aliveMinions.length
      },
      gameProgress,
      deathRate,
      evilAdvantage,
      threatLevel,
      suspicionLevels,
      keyPlayers: this._identifyKeyPlayers(gameState),
      timestamp: Date.now()
    };
  }

  /**
   * 生成决策建议
   * @param {object} context 决策上下文
   * @returns {Array} 决策建议列表
   */
  generateDecisionSuggestions(context) {
    const { gameState, playerPerspective, availableActions, riskTolerance = 0.5 } = context;

    if (!playerPerspective || !playerPerspective.isEvil) {
      return [];
    }

    const analysis = this.analyzeGameState(gameState);
    const suggestions = [];

    // 根据游戏阶段生成不同的建议
    if (gameState.phase === GamePhase.NIGHT || gameState.phase === GamePhase.FIRST_NIGHT) {
      suggestions.push(...this._generateNightSuggestions(context, analysis));
    } else if (gameState.phase === GamePhase.DAY) {
      suggestions.push(...this._generateDaySuggestions(context, analysis));
    }

    // 根据AI难度调整建议质量
    return this._adjustSuggestionsByDifficulty(suggestions, riskTolerance);
  }

  /**
   * 评估决策的风险和收益
   * @param {object} decision 决策
   * @param {GameState} gameState 游戏状态
   * @returns {object} 决策评估结果
   */
  evaluateDecision(decision, gameState) {
    const analysis = this.analyzeGameState(gameState);
    
    // 评估收益
    const benefit = this._calculateBenefit(decision, gameState, analysis);
    
    // 评估风险
    const risk = this._calculateRisk(decision, gameState, analysis);
    
    // 计算期望值
    const expectedValue = benefit - risk;
    
    // 评估成功概率
    const successProbability = this._estimateSuccessProbability(decision, gameState, analysis);

    return {
      decision,
      benefit,
      risk,
      expectedValue,
      successProbability,
      recommendation: expectedValue > 0 ? 'recommended' : 'not_recommended',
      confidence: Math.abs(expectedValue) / 10, // 归一化到0-1
      timestamp: Date.now()
    };
  }

  /**
   * 学习和优化决策算法
   * @param {object} gameHistory 游戏历史
   * @param {object} outcome 游戏结果
   */
  learnFromGameOutcome(_gameHistory, outcome) {
    // 记录游戏历史
    this.gameHistory.push({
      history: _gameHistory,
      outcome,
      timestamp: Date.now()
    });

    // 限制历史记录大小
    if (this.gameHistory.length > 100) {
      this.gameHistory.shift();
    }

    // 这里可以实现更复杂的学习算法
    // 例如：分析成功和失败的决策模式
    console.log(`AI learned from game outcome: ${outcome.winner} won`);
  }

  /**
   * 设置AI难度
   * @param {string} difficulty 难度级别
   */
  setDifficulty(difficulty) {
    if (['easy', 'medium', 'hard'].includes(difficulty)) {
      this.aiDifficulty = difficulty;
    }
  }

  /**
   * 获取AI难度
   * @returns {string} 难度级别
   */
  getDifficulty() {
    return this.aiDifficulty;
  }

  /**
   * 格式化决策建议为人类可读的文本
   * @param {Array} suggestions 决策建议列表
   * @returns {string} 格式化的文本
   */
  formatSuggestions(suggestions) {
    return DecisionFormatter.formatDecisionSuggestions(suggestions);
  }

  /**
   * 格式化游戏分析结果
   * @param {object} analysis 游戏分析结果
   * @returns {string} 格式化的文本
   */
  formatAnalysis(analysis) {
    return DecisionFormatter.formatGameAnalysis(analysis);
  }

  /**
   * 格式化决策评估结果
   * @param {object} evaluation 决策评估结果
   * @returns {string} 格式化的文本
   */
  formatEvaluation(evaluation) {
    return DecisionFormatter.formatDecisionEvaluation(evaluation);
  }

  /**
   * 生成决策摘要
   * @param {Array} suggestions 决策建议列表
   * @returns {object} 决策摘要
   */
  generateSummary(suggestions) {
    return DecisionFormatter.generateDecisionSummary(suggestions);
  }

  /**
   * 获取完整的决策报告
   * @param {object} context 决策上下文
   * @returns {object} 完整的决策报告
   */
  getDecisionReport(context) {
    const analysis = this.analyzeGameState(context.gameState);
    const suggestions = this.generateDecisionSuggestions(context);
    const summary = this.generateSummary(suggestions);

    return {
      analysis,
      suggestions,
      summary,
      formattedAnalysis: this.formatAnalysis(analysis),
      formattedSuggestions: this.formatSuggestions(suggestions),
      timestamp: Date.now()
    };
  }

  // 私有方法

  /**
   * 计算恶人优势
   * @private
   */
  _calculateEvilAdvantage(goodCount, evilCount) {
    if (goodCount === 0) return 1;
    if (evilCount === 0) return -1;
    
    // 恶人优势 = (恶人数 / 好人数) - 理想比例
    const ratio = evilCount / goodCount;
    const idealRatio = 0.33; // 理想情况下恶人约占1/3
    
    return (ratio - idealRatio) / idealRatio;
  }

  /**
   * 评估威胁等级
   * @private
   */
  _assessThreatLevel(gameState, aliveGood, aliveEvil) {
    let threatLevel = 0;

    // 好人数量威胁
    if (aliveGood.length > aliveEvil.length * 2) {
      threatLevel += 0.3;
    }

    // 特定角色威胁（例如：占卜师、共情者等信息角色）
    const dangerousRoles = ['fortuneteller', 'empath', 'investigator', 'washerwoman', 'librarian'];
    const dangerousAlive = aliveGood.filter(p => 
      p.role && dangerousRoles.includes(p.role.id)
    );
    threatLevel += dangerousAlive.length * 0.2;

    // 游戏进度威胁
    if (gameState.day > 3 && aliveGood.length > aliveEvil.length) {
      threatLevel += 0.2;
    }

    return Math.min(threatLevel, 1);
  }

  /**
   * 计算怀疑等级
   * @private
   */
  _calculateSuspicionLevels(gameState) {
    const suspicionLevels = {};

    gameState.players.forEach(player => {
      let suspicion = 0;

      // 基于投票行为
      // 基于发言模式
      // 基于角色声明
      // 这里简化处理

      suspicionLevels[player.id] = suspicion;
    });

    return suspicionLevels;
  }

  /**
   * 识别关键玩家
   * @private
   */
  _identifyKeyPlayers(gameState) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const keyPlayers = [];

    // 识别信息角色
    const infoRoles = ['fortuneteller', 'empath', 'investigator', 'washerwoman', 'librarian'];
    alivePlayers.forEach(player => {
      if (player.role && infoRoles.includes(player.role.id)) {
        keyPlayers.push({
          player,
          reason: 'information_role',
          priority: 'high'
        });
      }
    });

    // 识别保护角色
    const protectionRoles = ['monk', 'soldier', 'ravenkeeper'];
    alivePlayers.forEach(player => {
      if (player.role && protectionRoles.includes(player.role.id)) {
        keyPlayers.push({
          player,
          reason: 'protection_role',
          priority: 'medium'
        });
      }
    });

    return keyPlayers;
  }

  /**
   * 生成夜间建议
   * @private
   */
  _generateNightSuggestions(context, analysis) {
    const { gameState, playerPerspective } = context;
    const suggestions = [];

    // 如果是恶魔，建议杀人目标
    if (playerPerspective.role && playerPerspective.role.team === Team.DEMON) {
      const targets = this._selectKillTargets(gameState, analysis);
      
      targets.forEach((target, index) => {
        suggestions.push({
          action: {
            type: 'kill',
            target: target.player,
            phase: 'night'
          },
          confidence: target.confidence,
          reasoning: target.reasoning,
          expectedOutcome: {
            description: `Eliminate ${target.player.name}`,
            impact: target.impact
          },
          risks: target.risks,
          alternatives: index === 0 ? targets.slice(1).map(t => ({
            action: { type: 'kill', target: t.player },
            reason: t.reasoning
          })) : [],
          priority: index + 1
        });
      });
    }

    // 如果是爪牙，建议支持行动
    if (playerPerspective.role && playerPerspective.role.team === Team.MINION) {
      suggestions.push(...this._generateMinionSuggestions(context, analysis));
    }

    return suggestions;
  }

  /**
   * 生成白天建议
   * @private
   */
  _generateDaySuggestions(context, analysis) {
    const { gameState, playerPerspective } = context;
    const suggestions = [];

    // 建议提名目标
    const nominationTargets = this._selectNominationTargets(gameState, analysis, playerPerspective);
    
    nominationTargets.forEach((target, index) => {
      suggestions.push({
        action: {
          type: 'nominate',
          target: target.player,
          phase: 'day'
        },
        confidence: target.confidence,
        reasoning: target.reasoning,
        expectedOutcome: {
          description: `Nominate ${target.player.name} for execution`,
          impact: target.impact
        },
        risks: target.risks,
        alternatives: [],
        priority: index + 1
      });
    });

    // 建议投票策略
    suggestions.push({
      action: {
        type: 'voting_strategy',
        strategy: this._determineVotingStrategy(gameState, analysis, playerPerspective)
      },
      confidence: 0.7,
      reasoning: 'Recommended voting pattern to avoid suspicion',
      expectedOutcome: {
        description: 'Maintain cover while influencing votes',
        impact: 'medium'
      },
      risks: [{ type: 'exposure', level: 'low' }],
      alternatives: [],
      priority: 2
    });

    return suggestions;
  }

  /**
   * 选择杀人目标
   * @private
   */
  _selectKillTargets(gameState, analysis) {
    const alivePlayers = gameState.players.filter(p => p.isAlive && !p.isEvil);
    const targets = [];

    // 优先级1：信息角色
    const infoRoles = ['fortuneteller', 'empath', 'investigator'];
    alivePlayers.forEach(player => {
      if (player.role && infoRoles.includes(player.role.id)) {
        targets.push({
          player,
          confidence: 0.9,
          reasoning: `${player.name} is an information role (${player.role.name}) and poses high threat`,
          impact: 'high',
          risks: [{ type: 'protection', level: 'medium' }]
        });
      }
    });

    // 优先级2：有影响力的玩家
    // 这里简化处理，随机选择
    if (targets.length < 3) {
      const remaining = alivePlayers.filter(p => !targets.find(t => t.player.id === p.id));
      remaining.slice(0, 3 - targets.length).forEach(player => {
        targets.push({
          player,
          confidence: 0.6,
          reasoning: `${player.name} is a potential threat`,
          impact: 'medium',
          risks: [{ type: 'suspicion', level: 'low' }]
        });
      });
    }

    return targets.slice(0, 3);
  }

  /**
   * 生成爪牙建议
   * @private
   */
  _generateMinionSuggestions(context, analysis) {
    const suggestions = [];

    // 根据具体爪牙角色生成建议
    // 这里简化处理

    suggestions.push({
      action: {
        type: 'support_demon',
        description: 'Provide cover for demon'
      },
      confidence: 0.7,
      reasoning: 'Support demon by creating confusion',
      expectedOutcome: {
        description: 'Protect demon identity',
        impact: 'medium'
      },
      risks: [{ type: 'exposure', level: 'medium' }],
      alternatives: [],
      priority: 1
    });

    return suggestions;
  }

  /**
   * 选择提名目标
   * @private
   */
  _selectNominationTargets(gameState, analysis, playerPerspective) {
    const alivePlayers = gameState.players.filter(p => p.isAlive && p.id !== playerPerspective.id);
    const targets = [];

    // 优先提名好人
    const goodPlayers = alivePlayers.filter(p => !p.isEvil);
    
    goodPlayers.slice(0, 2).forEach(player => {
      targets.push({
        player,
        confidence: 0.7,
        reasoning: `Nominate ${player.name} to eliminate good player`,
        impact: 'high',
        risks: [{ type: 'counter_nomination', level: 'medium' }]
      });
    });

    return targets;
  }

  /**
   * 确定投票策略
   * @private
   */
  _determineVotingStrategy(gameState, analysis, playerPerspective) {
    // 根据局势确定投票策略
    if (analysis.evilAdvantage > 0.2) {
      return 'aggressive'; // 积极投票处决好人
    } else if (analysis.threatLevel > 0.7) {
      return 'defensive'; // 保守投票，避免暴露
    } else {
      return 'balanced'; // 平衡策略
    }
  }

  /**
   * 根据难度调整建议
   * @private
   */
  _adjustSuggestionsByDifficulty(suggestions, riskTolerance) {
    if (this.aiDifficulty === 'easy') {
      // 简单难度：减少建议数量，降低准确性
      return suggestions.slice(0, 2).map(s => ({
        ...s,
        confidence: s.confidence * 0.7
      }));
    } else if (this.aiDifficulty === 'hard') {
      // 困难难度：提供更多建议，提高准确性
      return suggestions.map(s => ({
        ...s,
        confidence: Math.min(s.confidence * 1.2, 1)
      }));
    }
    
    // 中等难度：保持原样
    return suggestions;
  }

  /**
   * 计算收益
   * @private
   */
  _calculateBenefit(decision, gameState, analysis) {
    let benefit = 0;

    if (decision.action.type === 'kill') {
      // 杀死好人的收益
      benefit += 5;
      
      // 如果是关键角色，额外收益
      const target = decision.action.target;
      if (target.role && ['fortuneteller', 'empath'].includes(target.role.id)) {
        benefit += 3;
      }
    } else if (decision.action.type === 'nominate') {
      // 提名的收益
      benefit += 3;
    }

    return benefit;
  }

  /**
   * 计算风险
   * @private
   */
  _calculateRisk(decision, gameState, analysis) {
    let risk = 0;

    if (decision.action.type === 'kill') {
      // 杀人的风险
      risk += 2;
      
      // 如果目标可能被保护
      if (decision.risks && decision.risks.some(r => r.type === 'protection')) {
        risk += 2;
      }
    } else if (decision.action.type === 'nominate') {
      // 提名的风险
      risk += 1;
      
      // 如果可能被反提名
      if (decision.risks && decision.risks.some(r => r.type === 'counter_nomination')) {
        risk += 2;
      }
    }

    return risk;
  }

  /**
   * 估计成功概率
   * @private
   */
  _estimateSuccessProbability(decision, gameState, analysis) {
    let probability = 0.5; // 基础概率

    // 根据决策类型调整
    if (decision.action.type === 'kill') {
      probability = 0.8; // 杀人通常成功率较高
      
      // 如果有保护风险，降低概率
      if (decision.risks && decision.risks.some(r => r.type === 'protection')) {
        probability -= 0.3;
      }
    } else if (decision.action.type === 'nominate') {
      probability = 0.6; // 提名成功率中等
    }

    // 根据局势调整
    if (analysis.evilAdvantage > 0) {
      probability += 0.1;
    } else {
      probability -= 0.1;
    }

    return Math.max(0, Math.min(1, probability));
  }
}

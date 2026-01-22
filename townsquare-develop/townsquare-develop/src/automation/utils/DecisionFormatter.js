/**
 * 决策格式化工具
 * 用于格式化AI决策建议的输出
 */

/**
 * 格式化决策建议为人类可读的文本
 * @param {object} suggestion 决策建议
 * @returns {string} 格式化的文本
 */
export function formatDecisionSuggestion(suggestion) {
  const lines = [];
  
  // 标题
  lines.push(`=== 决策建议 #${suggestion.priority} ===`);
  lines.push('');
  
  // 行动描述
  lines.push(`行动: ${formatAction(suggestion.action)}`);
  lines.push(`置信度: ${(suggestion.confidence * 100).toFixed(0)}%`);
  lines.push('');
  
  // 理由
  lines.push(`理由: ${suggestion.reasoning}`);
  lines.push('');
  
  // 预期结果
  if (suggestion.expectedOutcome) {
    lines.push('预期结果:');
    lines.push(`  ${suggestion.expectedOutcome.description}`);
    lines.push(`  影响程度: ${suggestion.expectedOutcome.impact}`);
    lines.push('');
  }
  
  // 风险评估
  if (suggestion.risks && suggestion.risks.length > 0) {
    lines.push('风险评估:');
    suggestion.risks.forEach(risk => {
      lines.push(`  - ${risk.type}: ${risk.level}`);
    });
    lines.push('');
  }
  
  // 替代方案
  if (suggestion.alternatives && suggestion.alternatives.length > 0) {
    lines.push('替代方案:');
    suggestion.alternatives.forEach((alt, index) => {
      lines.push(`  ${index + 1}. ${formatAction(alt.action)} - ${alt.reason}`);
    });
    lines.push('');
  }
  
  return lines.join('\n');
}

/**
 * 格式化行动描述
 * @param {object} action 行动对象
 * @returns {string} 格式化的行动描述
 */
export function formatAction(action) {
  switch (action.type) {
    case 'kill':
      return `杀死 ${action.target.name}`;
    case 'nominate':
      return `提名 ${action.target.name}`;
    case 'voting_strategy':
      return `投票策略: ${formatVotingStrategy(action.strategy)}`;
    case 'support_demon':
      return action.description || '支持恶魔';
    default:
      return action.description || action.type;
  }
}

/**
 * 格式化投票策略
 * @param {string} strategy 策略类型
 * @returns {string} 格式化的策略描述
 */
export function formatVotingStrategy(strategy) {
  const strategies = {
    aggressive: '积极投票（主动推动处决好人）',
    defensive: '保守投票（避免暴露身份）',
    balanced: '平衡策略（根据局势灵活调整）'
  };
  
  return strategies[strategy] || strategy;
}

/**
 * 格式化多个决策建议
 * @param {Array} suggestions 决策建议列表
 * @returns {string} 格式化的文本
 */
export function formatDecisionSuggestions(suggestions) {
  if (!suggestions || suggestions.length === 0) {
    return '暂无可用的决策建议';
  }
  
  const lines = [];
  lines.push('╔════════════════════════════════════════╗');
  lines.push('║        AI 决策建议                     ║');
  lines.push('╚════════════════════════════════════════╝');
  lines.push('');
  
  suggestions.forEach((suggestion, index) => {
    if (index > 0) {
      lines.push('----------------------------------------');
      lines.push('');
    }
    lines.push(formatDecisionSuggestion(suggestion));
  });
  
  return lines.join('\n');
}

/**
 * 格式化游戏分析结果
 * @param {object} analysis 游戏分析结果
 * @returns {string} 格式化的文本
 */
export function formatGameAnalysis(analysis) {
  const lines = [];
  
  lines.push('=== 游戏状态分析 ===');
  lines.push('');
  
  // 基本信息
  lines.push(`游戏阶段: ${analysis.phase}`);
  lines.push(`当前天数: 第 ${analysis.day} 天`);
  lines.push('');
  
  // 玩家统计
  lines.push('玩家统计:');
  lines.push(`  总人数: ${analysis.playerCounts.total}`);
  lines.push(`  存活: ${analysis.playerCounts.alive}`);
  lines.push(`  好人: ${analysis.playerCounts.aliveGood}`);
  lines.push(`  恶人: ${analysis.playerCounts.aliveEvil}`);
  lines.push(`  恶魔: ${analysis.playerCounts.aliveDemons}`);
  lines.push(`  爪牙: ${analysis.playerCounts.aliveMinions}`);
  lines.push('');
  
  // 局势评估
  lines.push('局势评估:');
  lines.push(`  游戏进度: ${(analysis.gameProgress * 100).toFixed(0)}%`);
  lines.push(`  死亡率: ${(analysis.deathRate * 100).toFixed(0)}%`);
  lines.push(`  恶人优势: ${formatAdvantage(analysis.evilAdvantage)}`);
  lines.push(`  威胁等级: ${formatThreatLevel(analysis.threatLevel)}`);
  lines.push('');
  
  // 关键玩家
  if (analysis.keyPlayers && analysis.keyPlayers.length > 0) {
    lines.push('关键玩家:');
    analysis.keyPlayers.forEach(kp => {
      lines.push(`  - ${kp.player.name} (${kp.reason}, 优先级: ${kp.priority})`);
    });
    lines.push('');
  }
  
  return lines.join('\n');
}

/**
 * 格式化优势值
 * @param {number} advantage 优势值
 * @returns {string} 格式化的优势描述
 */
export function formatAdvantage(advantage) {
  if (advantage > 0.3) return '恶人占优';
  if (advantage > 0.1) return '恶人略优';
  if (advantage > -0.1) return '势均力敌';
  if (advantage > -0.3) return '好人略优';
  return '好人占优';
}

/**
 * 格式化威胁等级
 * @param {number} threatLevel 威胁等级 (0-1)
 * @returns {string} 格式化的威胁描述
 */
export function formatThreatLevel(threatLevel) {
  if (threatLevel > 0.7) return '极高';
  if (threatLevel > 0.5) return '高';
  if (threatLevel > 0.3) return '中等';
  if (threatLevel > 0.1) return '低';
  return '极低';
}

/**
 * 格式化决策评估结果
 * @param {object} evaluation 决策评估结果
 * @returns {string} 格式化的文本
 */
export function formatDecisionEvaluation(evaluation) {
  const lines = [];
  
  lines.push('=== 决策评估 ===');
  lines.push('');
  
  lines.push(`行动: ${formatAction(evaluation.decision.action)}`);
  lines.push('');
  
  lines.push('评估结果:');
  lines.push(`  收益: ${evaluation.benefit.toFixed(1)}`);
  lines.push(`  风险: ${evaluation.risk.toFixed(1)}`);
  lines.push(`  期望值: ${evaluation.expectedValue.toFixed(1)}`);
  lines.push(`  成功概率: ${(evaluation.successProbability * 100).toFixed(0)}%`);
  lines.push(`  推荐度: ${evaluation.recommendation === 'recommended' ? '推荐' : '不推荐'}`);
  lines.push(`  置信度: ${(evaluation.confidence * 100).toFixed(0)}%`);
  lines.push('');
  
  return lines.join('\n');
}

/**
 * 生成决策摘要
 * @param {Array} suggestions 决策建议列表
 * @returns {object} 决策摘要
 */
export function generateDecisionSummary(suggestions) {
  if (!suggestions || suggestions.length === 0) {
    return {
      totalSuggestions: 0,
      bestSuggestion: null,
      averageConfidence: 0,
      riskLevels: {}
    };
  }
  
  // 找出最佳建议（置信度最高）
  const bestSuggestion = suggestions.reduce((best, current) => 
    current.confidence > best.confidence ? current : best
  );
  
  // 计算平均置信度
  const averageConfidence = suggestions.reduce((sum, s) => sum + s.confidence, 0) / suggestions.length;
  
  // 统计风险等级
  const riskLevels = {};
  suggestions.forEach(s => {
    if (s.risks) {
      s.risks.forEach(risk => {
        riskLevels[risk.level] = (riskLevels[risk.level] || 0) + 1;
      });
    }
  });
  
  return {
    totalSuggestions: suggestions.length,
    bestSuggestion,
    averageConfidence,
    riskLevels
  };
}

export default {
  formatDecisionSuggestion,
  formatAction,
  formatVotingStrategy,
  formatDecisionSuggestions,
  formatGameAnalysis,
  formatAdvantage,
  formatThreatLevel,
  formatDecisionEvaluation,
  generateDecisionSummary
};

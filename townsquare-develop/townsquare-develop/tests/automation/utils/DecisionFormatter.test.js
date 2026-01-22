/**
 * 决策格式化工具测试
 */

import DecisionFormatter from '../../../src/automation/utils/DecisionFormatter';
import { GamePhase } from '../../../src/automation/types/GameTypes';

describe('DecisionFormatter', () => {
  describe('formatAction', () => {
    test('应该格式化杀人行动', () => {
      const action = {
        type: 'kill',
        target: { name: 'Player 1' }
      };

      const formatted = DecisionFormatter.formatAction(action);

      expect(formatted).toBe('杀死 Player 1');
    });

    test('应该格式化提名行动', () => {
      const action = {
        type: 'nominate',
        target: { name: 'Player 2' }
      };

      const formatted = DecisionFormatter.formatAction(action);

      expect(formatted).toBe('提名 Player 2');
    });

    test('应该格式化投票策略', () => {
      const action = {
        type: 'voting_strategy',
        strategy: 'aggressive'
      };

      const formatted = DecisionFormatter.formatAction(action);

      expect(formatted).toContain('投票策略');
      expect(formatted).toContain('积极投票');
    });

    test('应该处理未知行动类型', () => {
      const action = {
        type: 'unknown',
        description: 'Custom action'
      };

      const formatted = DecisionFormatter.formatAction(action);

      expect(formatted).toBe('Custom action');
    });
  });

  describe('formatVotingStrategy', () => {
    test('应该格式化积极策略', () => {
      const formatted = DecisionFormatter.formatVotingStrategy('aggressive');
      expect(formatted).toContain('积极');
    });

    test('应该格式化保守策略', () => {
      const formatted = DecisionFormatter.formatVotingStrategy('defensive');
      expect(formatted).toContain('保守');
    });

    test('应该格式化平衡策略', () => {
      const formatted = DecisionFormatter.formatVotingStrategy('balanced');
      expect(formatted).toContain('平衡');
    });

    test('应该处理未知策略', () => {
      const formatted = DecisionFormatter.formatVotingStrategy('unknown');
      expect(formatted).toBe('unknown');
    });
  });

  describe('formatDecisionSuggestion', () => {
    test('应该格式化完整的决策建议', () => {
      const suggestion = {
        action: {
          type: 'kill',
          target: { name: 'Player 1' }
        },
        confidence: 0.85,
        reasoning: 'Player 1 is a threat',
        expectedOutcome: {
          description: 'Eliminate threat',
          impact: 'high'
        },
        risks: [
          { type: 'protection', level: 'medium' }
        ],
        alternatives: [
          {
            action: { type: 'kill', target: { name: 'Player 2' } },
            reason: 'Alternative target'
          }
        ],
        priority: 1
      };

      const formatted = DecisionFormatter.formatDecisionSuggestion(suggestion);

      expect(formatted).toContain('决策建议 #1');
      expect(formatted).toContain('杀死 Player 1');
      expect(formatted).toContain('85%');
      expect(formatted).toContain('Player 1 is a threat');
      expect(formatted).toContain('预期结果');
      expect(formatted).toContain('风险评估');
      expect(formatted).toContain('替代方案');
    });

    test('应该处理没有风险的建议', () => {
      const suggestion = {
        action: { type: 'kill', target: { name: 'Player 1' } },
        confidence: 0.9,
        reasoning: 'Safe target',
        risks: [],
        alternatives: [],
        priority: 1
      };

      const formatted = DecisionFormatter.formatDecisionSuggestion(suggestion);

      expect(formatted).toContain('决策建议');
      expect(formatted).not.toContain('风险评估:');
    });

    test('应该处理没有替代方案的建议', () => {
      const suggestion = {
        action: { type: 'kill', target: { name: 'Player 1' } },
        confidence: 0.9,
        reasoning: 'Only option',
        risks: [],
        alternatives: [],
        priority: 1
      };

      const formatted = DecisionFormatter.formatDecisionSuggestion(suggestion);

      expect(formatted).toContain('决策建议');
      expect(formatted).not.toContain('替代方案:');
    });
  });

  describe('formatDecisionSuggestions', () => {
    test('应该格式化多个决策建议', () => {
      const suggestions = [
        {
          action: { type: 'kill', target: { name: 'Player 1' } },
          confidence: 0.9,
          reasoning: 'Best option',
          risks: [],
          alternatives: [],
          priority: 1
        },
        {
          action: { type: 'kill', target: { name: 'Player 2' } },
          confidence: 0.7,
          reasoning: 'Second option',
          risks: [],
          alternatives: [],
          priority: 2
        }
      ];

      const formatted = DecisionFormatter.formatDecisionSuggestions(suggestions);

      expect(formatted).toContain('AI 决策建议');
      expect(formatted).toContain('Player 1');
      expect(formatted).toContain('Player 2');
      expect(formatted).toContain('----------------------------------------');
    });

    test('应该处理空建议列表', () => {
      const formatted = DecisionFormatter.formatDecisionSuggestions([]);

      expect(formatted).toBe('暂无可用的决策建议');
    });

    test('应该处理null建议', () => {
      const formatted = DecisionFormatter.formatDecisionSuggestions(null);

      expect(formatted).toBe('暂无可用的决策建议');
    });
  });

  describe('formatGameAnalysis', () => {
    test('应该格式化游戏分析结果', () => {
      const analysis = {
        phase: GamePhase.DAY,
        day: 2,
        playerCounts: {
          total: 10,
          alive: 8,
          aliveGood: 6,
          aliveEvil: 2,
          aliveDemons: 1,
          aliveMinions: 1
        },
        gameProgress: 0.4,
        deathRate: 0.2,
        evilAdvantage: 0.15,
        threatLevel: 0.6,
        keyPlayers: [
          {
            player: { name: 'Player 1' },
            reason: 'information_role',
            priority: 'high'
          }
        ]
      };

      const formatted = DecisionFormatter.formatGameAnalysis(analysis);

      expect(formatted).toContain('游戏状态分析');
      expect(formatted).toContain('第 2 天');
      expect(formatted).toContain('总人数: 10');
      expect(formatted).toContain('存活: 8');
      expect(formatted).toContain('好人: 6');
      expect(formatted).toContain('恶人: 2');
      expect(formatted).toContain('游戏进度: 40%');
      expect(formatted).toContain('死亡率: 20%');
      expect(formatted).toContain('关键玩家');
      expect(formatted).toContain('Player 1');
    });

    test('应该处理没有关键玩家的情况', () => {
      const analysis = {
        phase: GamePhase.NIGHT,
        day: 1,
        playerCounts: {
          total: 5,
          alive: 5,
          aliveGood: 3,
          aliveEvil: 2,
          aliveDemons: 1,
          aliveMinions: 1
        },
        gameProgress: 0.2,
        deathRate: 0,
        evilAdvantage: 0,
        threatLevel: 0.3,
        keyPlayers: []
      };

      const formatted = DecisionFormatter.formatGameAnalysis(analysis);

      expect(formatted).toContain('游戏状态分析');
      expect(formatted).not.toContain('关键玩家:');
    });
  });

  describe('formatAdvantage', () => {
    test('应该格式化恶人占优', () => {
      expect(DecisionFormatter.formatAdvantage(0.4)).toBe('恶人占优');
    });

    test('应该格式化恶人略优', () => {
      expect(DecisionFormatter.formatAdvantage(0.2)).toBe('恶人略优');
    });

    test('应该格式化势均力敌', () => {
      expect(DecisionFormatter.formatAdvantage(0)).toBe('势均力敌');
    });

    test('应该格式化好人略优', () => {
      expect(DecisionFormatter.formatAdvantage(-0.2)).toBe('好人略优');
    });

    test('应该格式化好人占优', () => {
      expect(DecisionFormatter.formatAdvantage(-0.4)).toBe('好人占优');
    });
  });

  describe('formatThreatLevel', () => {
    test('应该格式化极高威胁', () => {
      expect(DecisionFormatter.formatThreatLevel(0.8)).toBe('极高');
    });

    test('应该格式化高威胁', () => {
      expect(DecisionFormatter.formatThreatLevel(0.6)).toBe('高');
    });

    test('应该格式化中等威胁', () => {
      expect(DecisionFormatter.formatThreatLevel(0.4)).toBe('中等');
    });

    test('应该格式化低威胁', () => {
      expect(DecisionFormatter.formatThreatLevel(0.2)).toBe('低');
    });

    test('应该格式化极低威胁', () => {
      expect(DecisionFormatter.formatThreatLevel(0.05)).toBe('极低');
    });
  });

  describe('formatDecisionEvaluation', () => {
    test('应该格式化决策评估结果', () => {
      const evaluation = {
        decision: {
          action: { type: 'kill', target: { name: 'Player 1' } }
        },
        benefit: 8.5,
        risk: 3.2,
        expectedValue: 5.3,
        successProbability: 0.75,
        recommendation: 'recommended',
        confidence: 0.85
      };

      const formatted = DecisionFormatter.formatDecisionEvaluation(evaluation);

      expect(formatted).toContain('决策评估');
      expect(formatted).toContain('杀死 Player 1');
      expect(formatted).toContain('收益: 8.5');
      expect(formatted).toContain('风险: 3.2');
      expect(formatted).toContain('期望值: 5.3');
      expect(formatted).toContain('成功概率: 75%');
      expect(formatted).toContain('推荐');
      expect(formatted).toContain('置信度: 85%');
    });

    test('应该格式化不推荐的决策', () => {
      const evaluation = {
        decision: {
          action: { type: 'nominate', target: { name: 'Player 2' } }
        },
        benefit: 2.0,
        risk: 5.0,
        expectedValue: -3.0,
        successProbability: 0.3,
        recommendation: 'not_recommended',
        confidence: 0.4
      };

      const formatted = DecisionFormatter.formatDecisionEvaluation(evaluation);

      expect(formatted).toContain('不推荐');
    });
  });

  describe('generateDecisionSummary', () => {
    test('应该生成决策摘要', () => {
      const suggestions = [
        {
          action: { type: 'kill', target: { name: 'Player 1' } },
          confidence: 0.9,
          risks: [{ type: 'protection', level: 'low' }]
        },
        {
          action: { type: 'kill', target: { name: 'Player 2' } },
          confidence: 0.7,
          risks: [{ type: 'suspicion', level: 'medium' }]
        },
        {
          action: { type: 'nominate', target: { name: 'Player 3' } },
          confidence: 0.6,
          risks: [{ type: 'counter_nomination', level: 'high' }]
        }
      ];

      const summary = DecisionFormatter.generateDecisionSummary(suggestions);

      expect(summary.totalSuggestions).toBe(3);
      expect(summary.bestSuggestion).toBeDefined();
      expect(summary.bestSuggestion.confidence).toBe(0.9);
      expect(summary.averageConfidence).toBeCloseTo(0.733, 2);
      expect(summary.riskLevels).toEqual({
        low: 1,
        medium: 1,
        high: 1
      });
    });

    test('应该处理空建议列表', () => {
      const summary = DecisionFormatter.generateDecisionSummary([]);

      expect(summary.totalSuggestions).toBe(0);
      expect(summary.bestSuggestion).toBeNull();
      expect(summary.averageConfidence).toBe(0);
      expect(summary.riskLevels).toEqual({});
    });

    test('应该处理null建议', () => {
      const summary = DecisionFormatter.generateDecisionSummary(null);

      expect(summary.totalSuggestions).toBe(0);
      expect(summary.bestSuggestion).toBeNull();
    });

    test('应该处理没有风险的建议', () => {
      const suggestions = [
        {
          action: { type: 'kill', target: { name: 'Player 1' } },
          confidence: 0.8,
          risks: []
        }
      ];

      const summary = DecisionFormatter.generateDecisionSummary(suggestions);

      expect(summary.totalSuggestions).toBe(1);
      expect(summary.riskLevels).toEqual({});
    });
  });
});

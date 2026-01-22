/**
 * AI决策引擎测试
 */

import AIDecisionEngine from '../../../src/automation/core/AIDecisionEngine';
import GameStateManager from '../../../src/automation/core/GameStateManager';
import { GamePhase, Team } from '../../../src/automation/types/GameTypes';

describe('AIDecisionEngine', () => {
  let aiEngine;
  let gameStateManager;
  let mockGameState;

  beforeEach(() => {
    gameStateManager = new GameStateManager();
    aiEngine = new AIDecisionEngine(gameStateManager, 'medium');

    // 创建模拟游戏状态
    mockGameState = {
      gameId: 'test-game',
      phase: GamePhase.DAY,
      day: 1,
      players: [
        {
          id: 'p1',
          name: 'Player 1',
          isAlive: true,
          isEvil: false,
          role: { id: 'fortuneteller', name: 'Fortune Teller', team: Team.TOWNSFOLK }
        },
        {
          id: 'p2',
          name: 'Player 2',
          isAlive: true,
          isEvil: false,
          role: { id: 'empath', name: 'Empath', team: Team.TOWNSFOLK }
        },
        {
          id: 'p3',
          name: 'Player 3',
          isAlive: true,
          isEvil: true,
          role: { id: 'imp', name: 'Imp', team: Team.DEMON }
        },
        {
          id: 'p4',
          name: 'Player 4',
          isAlive: true,
          isEvil: true,
          role: { id: 'poisoner', name: 'Poisoner', team: Team.MINION }
        },
        {
          id: 'p5',
          name: 'Player 5',
          isAlive: true,
          isEvil: false,
          role: { id: 'monk', name: 'Monk', team: Team.TOWNSFOLK }
        }
      ],
      nominations: [],
      votes: [],
      nightActions: []
    };
  });

  describe('构造函数', () => {
    test('应该正确初始化AI决策引擎', () => {
      expect(aiEngine).toBeDefined();
      expect(aiEngine.gameStateManager).toBe(gameStateManager);
      expect(aiEngine.aiDifficulty).toBe('medium');
      expect(aiEngine.gameHistory).toEqual([]);
    });

    test('应该支持不同的AI难度级别', () => {
      const easyEngine = new AIDecisionEngine(gameStateManager, 'easy');
      const hardEngine = new AIDecisionEngine(gameStateManager, 'hard');

      expect(easyEngine.getDifficulty()).toBe('easy');
      expect(hardEngine.getDifficulty()).toBe('hard');
    });
  });

  describe('analyzeGameState', () => {
    test('应该正确分析游戏状态', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis).toBeDefined();
      expect(analysis.phase).toBe(GamePhase.DAY);
      expect(analysis.day).toBe(1);
      expect(analysis.playerCounts.total).toBe(5);
      expect(analysis.playerCounts.alive).toBe(5);
      expect(analysis.playerCounts.aliveGood).toBe(3);
      expect(analysis.playerCounts.aliveEvil).toBe(2);
      expect(analysis.playerCounts.aliveDemons).toBe(1);
      expect(analysis.playerCounts.aliveMinions).toBe(1);
    });

    test('应该计算游戏进度', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.gameProgress).toBeGreaterThanOrEqual(0);
      expect(analysis.gameProgress).toBeLessThanOrEqual(1);
      expect(analysis.deathRate).toBe(0); // 没有人死亡
    });

    test('应该计算死亡率', () => {
      mockGameState.players[0].isAlive = false;
      mockGameState.players[1].isAlive = false;

      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.deathRate).toBe(0.4); // 2/5 = 0.4
      expect(analysis.playerCounts.alive).toBe(3);
    });

    test('应该评估恶人优势', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.evilAdvantage).toBeDefined();
      expect(typeof analysis.evilAdvantage).toBe('number');
    });

    test('应该评估威胁等级', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.threatLevel).toBeDefined();
      expect(analysis.threatLevel).toBeGreaterThanOrEqual(0);
      expect(analysis.threatLevel).toBeLessThanOrEqual(1);
    });

    test('应该识别关键玩家', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.keyPlayers).toBeDefined();
      expect(Array.isArray(analysis.keyPlayers)).toBe(true);
      expect(analysis.keyPlayers.length).toBeGreaterThan(0);
    });

    test('应该包含时间戳', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.timestamp).toBeDefined();
      expect(typeof analysis.timestamp).toBe('number');
    });
  });

  describe('generateDecisionSuggestions', () => {
    test('应该为恶魔生成夜间杀人建议', () => {
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(suggestions).toBeDefined();
      expect(Array.isArray(suggestions)).toBe(true);
      expect(suggestions.length).toBeGreaterThan(0);

      const killSuggestion = suggestions.find(s => s.action.type === 'kill');
      expect(killSuggestion).toBeDefined();
      expect(killSuggestion.action.target).toBeDefined();
      expect(killSuggestion.confidence).toBeGreaterThan(0);
      expect(killSuggestion.reasoning).toBeDefined();
    });

    test('应该为爪牙生成夜间建议', () => {
      mockGameState.phase = GamePhase.NIGHT;
      const minion = mockGameState.players.find(p => p.role.team === Team.MINION);

      const context = {
        gameState: mockGameState,
        playerPerspective: minion,
        availableActions: [],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(suggestions).toBeDefined();
      expect(Array.isArray(suggestions)).toBe(true);
    });

    test('应该为恶人生成白天提名建议', () => {
      mockGameState.phase = GamePhase.DAY;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['nominate'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(suggestions).toBeDefined();
      expect(Array.isArray(suggestions)).toBe(true);

      const nominateSuggestion = suggestions.find(s => s.action.type === 'nominate');
      expect(nominateSuggestion).toBeDefined();
      expect(nominateSuggestion.action.target).toBeDefined();
    });

    test('应该为恶人生成投票策略建议', () => {
      mockGameState.phase = GamePhase.DAY;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['vote'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      const votingSuggestion = suggestions.find(s => s.action.type === 'voting_strategy');
      expect(votingSuggestion).toBeDefined();
      expect(votingSuggestion.action.strategy).toBeDefined();
      expect(['aggressive', 'defensive', 'balanced']).toContain(votingSuggestion.action.strategy);
    });

    test('不应该为好人生成建议', () => {
      const goodPlayer = mockGameState.players.find(p => !p.isEvil);

      const context = {
        gameState: mockGameState,
        playerPerspective: goodPlayer,
        availableActions: ['nominate'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(suggestions).toEqual([]);
    });

    test('建议应该包含完整的结构', () => {
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);
      const suggestion = suggestions[0];

      expect(suggestion.action).toBeDefined();
      expect(suggestion.confidence).toBeDefined();
      expect(suggestion.reasoning).toBeDefined();
      expect(suggestion.expectedOutcome).toBeDefined();
      expect(suggestion.risks).toBeDefined();
      expect(suggestion.alternatives).toBeDefined();
      expect(suggestion.priority).toBeDefined();
    });
  });

  describe('evaluateDecision', () => {
    test('应该评估杀人决策', () => {
      const decision = {
        action: {
          type: 'kill',
          target: mockGameState.players[0]
        },
        risks: [{ type: 'protection', level: 'medium' }]
      };

      const evaluation = aiEngine.evaluateDecision(decision, mockGameState);

      expect(evaluation).toBeDefined();
      expect(evaluation.decision).toBe(decision);
      expect(evaluation.benefit).toBeGreaterThan(0);
      expect(evaluation.risk).toBeGreaterThan(0);
      expect(evaluation.expectedValue).toBeDefined();
      expect(evaluation.successProbability).toBeGreaterThan(0);
      expect(evaluation.successProbability).toBeLessThanOrEqual(1);
      expect(evaluation.recommendation).toBeDefined();
      expect(['recommended', 'not_recommended']).toContain(evaluation.recommendation);
    });

    test('应该评估提名决策', () => {
      const decision = {
        action: {
          type: 'nominate',
          target: mockGameState.players[0]
        },
        risks: [{ type: 'counter_nomination', level: 'medium' }]
      };

      const evaluation = aiEngine.evaluateDecision(decision, mockGameState);

      expect(evaluation).toBeDefined();
      expect(evaluation.benefit).toBeGreaterThan(0);
      expect(evaluation.risk).toBeGreaterThan(0);
    });

    test('应该计算期望值', () => {
      const decision = {
        action: {
          type: 'kill',
          target: mockGameState.players[0]
        },
        risks: []
      };

      const evaluation = aiEngine.evaluateDecision(decision, mockGameState);

      expect(evaluation.expectedValue).toBe(evaluation.benefit - evaluation.risk);
    });

    test('应该根据期望值给出推荐', () => {
      const goodDecision = {
        action: {
          type: 'kill',
          target: mockGameState.players[0] // 信息角色
        },
        risks: []
      };

      const evaluation = aiEngine.evaluateDecision(goodDecision, mockGameState);

      expect(evaluation.recommendation).toBe('recommended');
      expect(evaluation.expectedValue).toBeGreaterThan(0);
    });

    test('应该包含置信度', () => {
      const decision = {
        action: {
          type: 'kill',
          target: mockGameState.players[0]
        },
        risks: []
      };

      const evaluation = aiEngine.evaluateDecision(decision, mockGameState);

      expect(evaluation.confidence).toBeGreaterThanOrEqual(0);
      expect(evaluation.confidence).toBeLessThanOrEqual(1);
    });
  });

  describe('learnFromGameOutcome', () => {
    test('应该记录游戏历史', () => {
      const gameHistory = {
        decisions: [],
        outcomes: []
      };

      const outcome = {
        winner: Team.EVIL,
        reason: 'All demons alive, good players eliminated'
      };

      aiEngine.learnFromGameOutcome(gameHistory, outcome);

      expect(aiEngine.gameHistory.length).toBe(1);
      expect(aiEngine.gameHistory[0].history).toBe(gameHistory);
      expect(aiEngine.gameHistory[0].outcome).toBe(outcome);
    });

    test('应该限制历史记录大小', () => {
      // 添加101条记录
      for (let i = 0; i < 101; i++) {
        aiEngine.learnFromGameOutcome({ id: i }, { winner: Team.EVIL });
      }

      expect(aiEngine.gameHistory.length).toBe(100);
      expect(aiEngine.gameHistory[0].history.id).toBe(1); // 第一条被移除
    });

    test('应该包含时间戳', () => {
      aiEngine.learnFromGameOutcome({}, { winner: Team.GOOD });

      expect(aiEngine.gameHistory[0].timestamp).toBeDefined();
      expect(typeof aiEngine.gameHistory[0].timestamp).toBe('number');
    });
  });

  describe('难度设置', () => {
    test('应该允许设置AI难度', () => {
      aiEngine.setDifficulty('easy');
      expect(aiEngine.getDifficulty()).toBe('easy');

      aiEngine.setDifficulty('hard');
      expect(aiEngine.getDifficulty()).toBe('hard');
    });

    test('应该拒绝无效的难度设置', () => {
      aiEngine.setDifficulty('invalid');
      expect(aiEngine.getDifficulty()).toBe('medium'); // 保持原值
    });

    test('简单难度应该减少建议数量', () => {
      aiEngine.setDifficulty('easy');
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(suggestions.length).toBeLessThanOrEqual(2);
    });

    test('简单难度应该降低置信度', () => {
      aiEngine.setDifficulty('easy');
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      suggestions.forEach(s => {
        expect(s.confidence).toBeLessThan(1);
      });
    });

    test('困难难度应该提高置信度', () => {
      aiEngine.setDifficulty('hard');
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      // 困难难度的置信度应该更高
      expect(suggestions.length).toBeGreaterThan(0);
    });
  });

  describe('边缘情况', () => {
    test('应该处理没有存活玩家的情况', () => {
      mockGameState.players.forEach(p => p.isAlive = false);

      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.playerCounts.alive).toBe(0);
      expect(analysis.playerCounts.aliveGood).toBe(0);
      expect(analysis.playerCounts.aliveEvil).toBe(0);
    });

    test('应该处理只有恶人存活的情况', () => {
      mockGameState.players.forEach(p => {
        if (!p.isEvil) p.isAlive = false;
      });

      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.playerCounts.aliveGood).toBe(0);
      expect(analysis.playerCounts.aliveEvil).toBe(2);
      expect(analysis.evilAdvantage).toBe(1);
    });

    test('应该处理只有好人存活的情况', () => {
      mockGameState.players.forEach(p => {
        if (p.isEvil) p.isAlive = false;
      });

      const analysis = aiEngine.analyzeGameState(mockGameState);

      expect(analysis.playerCounts.aliveGood).toBe(3);
      expect(analysis.playerCounts.aliveEvil).toBe(0);
      expect(analysis.evilAdvantage).toBe(-1);
    });

    test('应该处理没有可用行动的情况', () => {
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: [],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(Array.isArray(suggestions)).toBe(true);
    });

    test('应该处理空的玩家视角', () => {
      const context = {
        gameState: mockGameState,
        playerPerspective: null,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);

      expect(suggestions).toEqual([]);
    });
  });

  describe('格式化输出', () => {
    test('应该格式化决策建议', () => {
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);
      const formatted = aiEngine.formatSuggestions(suggestions);

      expect(formatted).toBeDefined();
      expect(typeof formatted).toBe('string');
      expect(formatted.length).toBeGreaterThan(0);
    });

    test('应该格式化游戏分析', () => {
      const analysis = aiEngine.analyzeGameState(mockGameState);
      const formatted = aiEngine.formatAnalysis(analysis);

      expect(formatted).toBeDefined();
      expect(typeof formatted).toBe('string');
      expect(formatted).toContain('游戏状态分析');
    });

    test('应该格式化决策评估', () => {
      const decision = {
        action: {
          type: 'kill',
          target: mockGameState.players[0]
        },
        risks: []
      };

      const evaluation = aiEngine.evaluateDecision(decision, mockGameState);
      const formatted = aiEngine.formatEvaluation(evaluation);

      expect(formatted).toBeDefined();
      expect(typeof formatted).toBe('string');
      expect(formatted).toContain('决策评估');
    });

    test('应该生成决策摘要', () => {
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const suggestions = aiEngine.generateDecisionSuggestions(context);
      const summary = aiEngine.generateSummary(suggestions);

      expect(summary).toBeDefined();
      expect(summary.totalSuggestions).toBe(suggestions.length);
      expect(summary.bestSuggestion).toBeDefined();
      expect(summary.averageConfidence).toBeGreaterThan(0);
    });

    test('应该生成完整的决策报告', () => {
      mockGameState.phase = GamePhase.NIGHT;
      const demon = mockGameState.players.find(p => p.role.team === Team.DEMON);

      const context = {
        gameState: mockGameState,
        playerPerspective: demon,
        availableActions: ['kill'],
        riskTolerance: 0.5
      };

      const report = aiEngine.getDecisionReport(context);

      expect(report).toBeDefined();
      expect(report.analysis).toBeDefined();
      expect(report.suggestions).toBeDefined();
      expect(report.summary).toBeDefined();
      expect(report.formattedAnalysis).toBeDefined();
      expect(report.formattedSuggestions).toBeDefined();
      expect(report.timestamp).toBeDefined();
    });
  });
});

/**
 * 自动化说书人系统测试
 */

import AutomatedStorytellerSystem from '../../../src/automation/core/AutomatedStorytellerSystem';
import { SystemStatus, AutomationLevel, AIDifficulty } from '../../../src/automation/types/AutomationTypes';
import { GamePhase } from '../../../src/automation/types/GameTypes';

describe('AutomatedStorytellerSystem', () => {
  let system;
  let mockStore;

  beforeEach(() => {
    // 创建模拟的Vuex store
    mockStore = {
      state: {
        automation: {
          logs: []
        }
      },
      commit: jest.fn(),
      dispatch: jest.fn()
    };

    system = new AutomatedStorytellerSystem();
    // 为GameStateManager设置模拟store
    system.gameStateManager.store = mockStore;
  });

  describe('构造函数', () => {
    test('应该正确初始化系统', () => {
      expect(system).toBeDefined();
      expect(system.status).toBe(SystemStatus.IDLE);
      expect(system.isPaused).toBe(false);
      expect(system.isInitialized).toBe(false);
      expect(system.configManager).toBeDefined();
      expect(system.gameStateManager).toBeDefined();
      expect(system.roleAssigner).toBeDefined();
      expect(system.victoryConditionChecker).toBeDefined();
    });
  });

  describe('initialize', () => {
    test('应该初始化系统', async () => {
      const config = {
        scriptType: 'trouble-brewing',
        playerCount: 7,
        automationLevel: AutomationLevel.FULL_AUTO,
        aiDifficulty: AIDifficulty.MEDIUM
      };

      const result = await system.initialize(config);

      expect(result.success).toBe(true);
      expect(system.isInitialized).toBe(true);
      expect(system.status).toBe(SystemStatus.IDLE);
      expect(system.currentConfig).toBeDefined();
      expect(system.aiDecisionEngine).toBeDefined();
      expect(system.nightActionProcessor).toBeDefined();
      expect(system.votingManager).toBeDefined();
    });

    test('应该在初始化时设置正确的状态', async () => {
      const config = {
        scriptType: 'trouble-brewing',
        playerCount: 7
      };

      await system.initialize(config);

      expect(system.status).toBe(SystemStatus.IDLE);
      expect(system.isInitialized).toBe(true);
    });

    test('应该处理初始化错误', async () => {
      const invalidConfig = {
        playerCount: 3 // 无效：小于5
      };

      await expect(system.initialize(invalidConfig)).rejects.toThrow();
      expect(system.status).toBe(SystemStatus.ERROR);
    });

    test('应该支持启用状态同步', async () => {
      const config = {
        scriptType: 'trouble-brewing',
        playerCount: 7,
        enableSync: true
      };

      await system.initialize(config);

      expect(system.stateSynchronizer).toBeDefined();
    });
  });

  describe('startAutomatedGame', () => {
    beforeEach(async () => {
      await system.initialize({
        scriptType: 'trouble-brewing',
        playerCount: 7
      });
    });

    test('应该在未初始化时抛出错误', async () => {
      const uninitializedSystem = new AutomatedStorytellerSystem();
      const players = [{ name: 'Player 1' }];

      await expect(uninitializedSystem.startAutomatedGame(players)).rejects.toThrow('System not initialized');
    });

    test('应该在游戏已运行时抛出错误', async () => {
      const players = [
        { name: 'Player 1' },
        { name: 'Player 2' },
        { name: 'Player 3' },
        { name: 'Player 4' },
        { name: 'Player 5' },
        { name: 'Player 6' },
        { name: 'Player 7' }
      ];

      // 启动第一个游戏
      system.status = SystemStatus.RUNNING;

      await expect(system.startAutomatedGame(players)).rejects.toThrow('Game is already running');
    });
  });

  describe('暂停和恢复', () => {
    test('应该暂停自动化', () => {
      system.status = SystemStatus.RUNNING;
      system.pauseAutomation();

      expect(system.isPaused).toBe(true);
      expect(system.status).toBe(SystemStatus.PAUSED);
    });

    test('应该恢复自动化', () => {
      system.status = SystemStatus.PAUSED;
      system.isPaused = true;

      system.resumeAutomation();

      expect(system.isPaused).toBe(false);
      expect(system.status).toBe(SystemStatus.RUNNING);
    });

    test('应该只在运行时暂停', () => {
      system.status = SystemStatus.IDLE;
      system.pauseAutomation();

      expect(system.status).toBe(SystemStatus.IDLE);
    });

    test('应该只在暂停时恢复', () => {
      system.status = SystemStatus.IDLE;
      system.resumeAutomation();

      expect(system.status).toBe(SystemStatus.IDLE);
    });
  });

  describe('stopGame', () => {
    test('应该停止游戏', () => {
      system.status = SystemStatus.RUNNING;
      system.isPaused = true;

      system.stopGame();

      expect(system.status).toBe(SystemStatus.IDLE);
      expect(system.isPaused).toBe(false);
    });
  });

  describe('getSystemStatus', () => {
    test('应该返回系统状态', () => {
      const status = system.getSystemStatus();

      expect(status).toBeDefined();
      expect(status.status).toBe(SystemStatus.IDLE);
      expect(status.isPaused).toBe(false);
      expect(status.isInitialized).toBe(false);
      expect(status.timestamp).toBeDefined();
    });

    test('应该包含当前配置', async () => {
      await system.initialize({
        scriptType: 'trouble-brewing',
        playerCount: 7
      });

      const status = system.getSystemStatus();

      expect(status.currentConfig).toBeDefined();
      expect(status.currentConfig.scriptType).toBe('trouble-brewing');
    });
  });

  describe('handleEmergency', () => {
    test('应该处理暂停紧急情况', () => {
      system.status = SystemStatus.RUNNING;
      system.handleEmergency('pause');

      expect(system.status).toBe(SystemStatus.PAUSED);
    });

    test('应该处理停止紧急情况', () => {
      system.status = SystemStatus.RUNNING;
      system.handleEmergency('stop');

      expect(system.status).toBe(SystemStatus.IDLE);
    });

    test('应该调用注册的紧急处理器', () => {
      const handler = jest.fn();
      system.onEmergency(handler);

      system.handleEmergency('pause');

      expect(handler).toHaveBeenCalledWith('pause');
    });

    test('应该处理未知的紧急类型', () => {
      expect(() => {
        system.handleEmergency('unknown');
      }).not.toThrow();
    });
  });

  describe('错误处理', () => {
    test('应该注册错误处理器', () => {
      const handler = jest.fn();
      system.onError(handler);

      expect(system.errorHandlers).toContain(handler);
    });

    test('应该注册紧急情况处理器', () => {
      const handler = jest.fn();
      system.onEmergency(handler);

      expect(system.emergencyHandlers).toContain(handler);
    });
  });

  describe('AI决策', () => {
    beforeEach(async () => {
      await system.initialize({
        scriptType: 'trouble-brewing',
        playerCount: 7,
        aiDifficulty: AIDifficulty.MEDIUM
      });
    });

    test('应该获取AI决策建议', () => {
      const context = {
        gameState: {
          phase: GamePhase.NIGHT,
          day: 1,
          players: [
            { id: 'p1', name: 'Player 1', isAlive: true, isEvil: false },
            { id: 'p2', name: 'Player 2', isAlive: true, isEvil: true }
          ]
        },
        playerPerspective: { id: 'p2', isEvil: true },
        availableActions: ['kill']
      };

      const suggestions = system.getAIDecisionSuggestions(context);

      expect(Array.isArray(suggestions)).toBe(true);
    });

    test('应该获取完整的决策报告', () => {
      const context = {
        gameState: {
          phase: GamePhase.NIGHT,
          day: 1,
          players: [
            { id: 'p1', name: 'Player 1', isAlive: true, isEvil: false },
            { id: 'p2', name: 'Player 2', isAlive: true, isEvil: true }
          ]
        },
        playerPerspective: { id: 'p2', isEvil: true },
        availableActions: ['kill']
      };

      const report = system.getAIDecisionReport(context);

      expect(report).toBeDefined();
      expect(report.analysis).toBeDefined();
      expect(report.suggestions).toBeDefined();
      expect(report.summary).toBeDefined();
    });

    test('应该在未初始化时抛出错误', () => {
      const uninitializedSystem = new AutomatedStorytellerSystem();
      const context = {
        gameState: { players: [] },
        playerPerspective: { isEvil: true },
        availableActions: []
      };

      expect(() => {
        uninitializedSystem.getAIDecisionSuggestions(context);
      }).toThrow('AI Decision Engine not initialized');
    });
  });

  describe('获取器方法', () => {
    test('应该获取配置管理器', () => {
      const configManager = system.getConfigManager();

      expect(configManager).toBeDefined();
      expect(configManager).toBe(system.configManager);
    });

    test('应该获取游戏状态管理器', () => {
      const gameStateManager = system.getGameStateManager();

      expect(gameStateManager).toBeDefined();
      expect(gameStateManager).toBe(system.gameStateManager);
    });

    test('应该获取AI决策引擎', async () => {
      await system.initialize({
        scriptType: 'trouble-brewing',
        playerCount: 7
      });

      const aiEngine = system.getAIDecisionEngine();

      expect(aiEngine).toBeDefined();
      expect(aiEngine).toBe(system.aiDecisionEngine);
    });

    test('应该获取胜负判断器', () => {
      const victoryChecker = system.getVictoryConditionChecker();

      expect(victoryChecker).toBeDefined();
      expect(victoryChecker).toBe(system.victoryConditionChecker);
    });
  });
});

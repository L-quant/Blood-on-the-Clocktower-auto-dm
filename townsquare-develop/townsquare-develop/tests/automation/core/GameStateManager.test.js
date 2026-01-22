/**
 * GameStateManager 测试
 */

import GameStateManager from '@/automation/core/GameStateManager';
import { GamePhase, Player, GameConfiguration } from '@/automation/types/GameTypes';

// 模拟Vuex store
const mockStore = {
  commit: jest.fn(),
  state: {
    automation: {
      systemStatus: 'idle'
    }
  }
};

describe('GameStateManager', () => {
  let gameStateManager;

  beforeEach(() => {
    gameStateManager = new GameStateManager(mockStore);
    mockStore.commit.mockClear();
  });

  describe('初始化', () => {
    test('should initialize with default state', () => {
      const state = gameStateManager.getCurrentState();
      
      expect(state.phase).toBe(GamePhase.SETUP);
      expect(state.day).toBe(0);
      expect(state.players).toEqual([]);
      expect(state.gameId).toBe('');
    });

    test('should initialize game with config and players', () => {
      const config = new GameConfiguration();
      const players = [
        new Player('player1', 'Alice'),
        new Player('player2', 'Bob')
      ];

      gameStateManager.initializeGame(config, players);
      const state = gameStateManager.getCurrentState();

      expect(state.phase).toBe(GamePhase.SETUP);
      expect(state.players).toHaveLength(2);
      expect(state.gameConfiguration).toEqual(config);
      expect(state.gameId).toBeTruthy();
    });
  });

  describe('玩家管理', () => {
    beforeEach(() => {
      const config = new GameConfiguration();
      gameStateManager.initializeGame(config, []);
    });

    test('should add player during setup phase', () => {
      const player = new Player('player1', 'Alice');
      
      gameStateManager.addPlayer(player);
      const state = gameStateManager.getCurrentState();
      
      expect(state.players).toHaveLength(1);
      expect(state.players[0].name).toBe('Alice');
    });

    test('should not add duplicate player', () => {
      const player = new Player('player1', 'Alice');
      
      gameStateManager.addPlayer(player);
      
      expect(() => {
        gameStateManager.addPlayer(player);
      }).toThrow('Player already exists: player1');
    });

    test('should not add player after game started', async () => {
      const player = new Player('player1', 'Alice');
      
      // 转换到游戏开始阶段
      await gameStateManager.transitionToPhase(GamePhase.FIRST_NIGHT);
      
      expect(() => {
        gameStateManager.addPlayer(player);
      }).toThrow('Cannot add players after game has started');
    });

    test('should remove player during setup phase', () => {
      const player = new Player('player1', 'Alice');
      
      gameStateManager.addPlayer(player);
      gameStateManager.removePlayer('player1');
      
      const state = gameStateManager.getCurrentState();
      expect(state.players).toHaveLength(0);
    });

    test('should update player state', () => {
      const player = new Player('player1', 'Alice');
      gameStateManager.addPlayer(player);
      
      gameStateManager.updatePlayerState('player1', { isAlive: false });
      
      const state = gameStateManager.getCurrentState();
      expect(state.players[0].isAlive).toBe(false);
    });
  });

  describe('阶段转换', () => {
    beforeEach(() => {
      const config = new GameConfiguration();
      const players = [new Player('player1', 'Alice')];
      gameStateManager.initializeGame(config, players);
    });

    test('should validate valid transitions', () => {
      expect(gameStateManager.validateTransition(GamePhase.SETUP, GamePhase.FIRST_NIGHT)).toBe(true);
      expect(gameStateManager.validateTransition(GamePhase.FIRST_NIGHT, GamePhase.DAY)).toBe(true);
      expect(gameStateManager.validateTransition(GamePhase.DAY, GamePhase.NIGHT)).toBe(true);
      expect(gameStateManager.validateTransition(GamePhase.NIGHT, GamePhase.DAY)).toBe(true);
    });

    test('should reject invalid transitions', () => {
      expect(gameStateManager.validateTransition(GamePhase.SETUP, GamePhase.DAY)).toBe(false);
      expect(gameStateManager.validateTransition(GamePhase.FIRST_NIGHT, GamePhase.NIGHT)).toBe(false);
      expect(gameStateManager.validateTransition(GamePhase.ENDED, GamePhase.DAY)).toBe(false);
    });

    test('should transition to first night from setup', async () => {
      await gameStateManager.transitionToPhase(GamePhase.FIRST_NIGHT);
      
      const state = gameStateManager.getCurrentState();
      expect(state.phase).toBe(GamePhase.FIRST_NIGHT);
    });

    test('should increment day when transitioning to day phase', async () => {
      await gameStateManager.transitionToPhase(GamePhase.FIRST_NIGHT);
      await gameStateManager.transitionToPhase(GamePhase.DAY);
      
      const state = gameStateManager.getCurrentState();
      expect(state.phase).toBe(GamePhase.DAY);
      expect(state.day).toBe(1);
    });

    test('should throw error for invalid transition', async () => {
      await expect(
        gameStateManager.transitionToPhase(GamePhase.DAY)
      ).rejects.toThrow('Invalid transition from setup to day');
    });
  });

  describe('玩家筛选', () => {
    beforeEach(() => {
      const config = new GameConfiguration();
      const players = [
        new Player('player1', 'Alice'),
        new Player('player2', 'Bob'),
        new Player('player3', 'Charlie')
      ];
      
      // 设置一些玩家状态
      players[0].isAlive = true;
      players[0].isEvil = false;
      players[1].isAlive = false;
      players[1].isEvil = true;
      players[2].isAlive = true;
      players[2].isEvil = true;
      
      gameStateManager.initializeGame(config, players);
    });

    test('should get alive players', () => {
      const alivePlayers = gameStateManager.getAlivePlayers();
      expect(alivePlayers).toHaveLength(2);
      expect(alivePlayers.map(p => p.name)).toEqual(['Alice', 'Charlie']);
    });

    test('should get dead players', () => {
      const deadPlayers = gameStateManager.getDeadPlayers();
      expect(deadPlayers).toHaveLength(1);
      expect(deadPlayers[0].name).toBe('Bob');
    });

    test('should get evil players', () => {
      const evilPlayers = gameStateManager.getEvilPlayers();
      expect(evilPlayers).toHaveLength(2);
      expect(evilPlayers.map(p => p.name)).toEqual(['Bob', 'Charlie']);
    });

    test('should get good players', () => {
      const goodPlayers = gameStateManager.getGoodPlayers();
      expect(goodPlayers).toHaveLength(1);
      expect(goodPlayers[0].name).toBe('Alice');
    });
  });

  describe('状态历史和回滚', () => {
    beforeEach(() => {
      const config = new GameConfiguration();
      const players = [new Player('player1', 'Alice')];
      gameStateManager.initializeGame(config, players);
    });

    test('should save state to history before changes', () => {
      const initialState = gameStateManager.getCurrentState();
      
      gameStateManager.updatePlayerState('player1', { isAlive: false });
      
      // 应该能够回滚
      gameStateManager.rollbackToPreviousState();
      
      const rolledBackState = gameStateManager.getCurrentState();
      expect(rolledBackState.players[0].isAlive).toBe(true);
    });

    test('should throw error when no history to rollback', () => {
      expect(() => {
        gameStateManager.rollbackToPreviousState();
      }).toThrow('No previous state to rollback to');
    });

    test('should limit history size', () => {
      // 超过最大历史记录数量的更新
      for (let i = 0; i < 60; i++) {
        gameStateManager.updatePlayerState('player1', { votes: i });
      }
      
      // 历史记录应该被限制在最大大小
      expect(gameStateManager.stateHistory.length).toBeLessThanOrEqual(50);
    });
  });

  describe('状态同步', () => {
    test('should sync state to store', () => {
      const config = new GameConfiguration();
      const players = [new Player('player1', 'Alice')];
      
      gameStateManager.initializeGame(config, players);
      
      // 验证store被调用
      expect(mockStore.commit).toHaveBeenCalledWith('players/set', expect.any(Array));
      expect(mockStore.commit).toHaveBeenCalledWith('session/setPlayerCount', 1);
    });
  });
});
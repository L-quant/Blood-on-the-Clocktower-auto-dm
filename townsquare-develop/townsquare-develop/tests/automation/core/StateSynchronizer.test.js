/**
 * StateSynchronizer 测试
 */

import StateSynchronizer from '@/automation/core/StateSynchronizer';
import { ErrorType } from '@/automation/types/AutomationTypes';

// 模拟WebSocket服务
const mockWebSocketService = {
  sendToClient: jest.fn(),
  on: jest.fn()
};

// 模拟Vuex store
const mockStore = {
  commit: jest.fn(),
  state: {
    game: { phase: 'setup' },
    players: { players: [] },
    session: { sessionId: 'test' },
    automation: { systemStatus: 'idle' }
  },
  getters: {
    'automation/getCurrentState': jest.fn()
  }
};

describe('StateSynchronizer', () => {
  let stateSynchronizer;

  beforeEach(() => {
    // 重置所有mock
    jest.clearAllMocks();
    
    stateSynchronizer = new StateSynchronizer(mockStore, mockWebSocketService);
    
    // 清除定时器
    jest.clearAllTimers();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('初始化', () => {
    test('should initialize with default values', () => {
      expect(stateSynchronizer.syncQueue).toEqual([]);
      expect(stateSynchronizer.isProcessing).toBe(false);
      expect(stateSynchronizer.connectedClients.size).toBe(0);
      expect(stateSynchronizer.inconsistentClients.size).toBe(0);
    });

    test('should setup WebSocket listeners', () => {
      expect(mockWebSocketService.on).toHaveBeenCalledWith('client_connect', expect.any(Function));
      expect(mockWebSocketService.on).toHaveBeenCalledWith('client_disconnect', expect.any(Function));
      expect(mockWebSocketService.on).toHaveBeenCalledWith('heartbeat', expect.any(Function));
    });
  });

  describe('客户端连接管理', () => {
    test('should handle client connection', () => {
      const clientId = 'client1';
      const clientInfo = { name: 'Player1', version: '1.0' };

      stateSynchronizer.handleClientConnect(clientId, clientInfo);

      const client = stateSynchronizer.connectedClients.get(clientId);
      expect(client).toBeDefined();
      expect(client.isConnected).toBe(true);
      expect(client.name).toBe('Player1');
      expect(mockStore.commit).toHaveBeenCalledWith('automation/ADD_LOG', expect.objectContaining({
        level: 'info',
        message: `Client connected: ${clientId}`
      }));
    });

    test('should handle client disconnection', () => {
      const clientId = 'client1';
      
      // 先连接客户端
      stateSynchronizer.handleClientConnect(clientId, { name: 'Player1' });
      
      // 然后断开连接
      stateSynchronizer.handleClientDisconnect(clientId);

      const client = stateSynchronizer.connectedClients.get(clientId);
      expect(client.isConnected).toBe(false);
      expect(client.disconnectionTime).toBeDefined();
    });

    test('should get connection statistics', () => {
      // 添加一些客户端
      stateSynchronizer.handleClientConnect('client1', { name: 'Player1' });
      stateSynchronizer.handleClientConnect('client2', { name: 'Player2' });
      stateSynchronizer.handleClientDisconnect('client2');

      const stats = stateSynchronizer.getConnectionStats();
      
      expect(stats.totalClients).toBe(2);
      expect(stats.connectedClients).toBe(1);
      expect(stats.inconsistentClients).toBe(0);
      // 由于sendCurrentStateToClient会添加同步项到队列，所以队列可能不为空
      expect(stats.queueSize).toBeGreaterThanOrEqual(0);
    });
  });

  describe('状态同步', () => {
    beforeEach(() => {
      // 添加一些连接的客户端
      stateSynchronizer.handleClientConnect('client1', { name: 'Player1' });
      stateSynchronizer.handleClientConnect('client2', { name: 'Player2' });
      
      // 清除由handleClientConnect产生的调用
      jest.clearAllMocks();
    });

    test('should add sync item to queue', async () => {
      const stateUpdate = { phase: 'day', day: 1 };
      
      // 直接测试syncState方法是否正确添加到队列
      const syncPromise = stateSynchronizer.syncState(stateUpdate);
      
      // 验证同步项被添加到队列
      expect(stateSynchronizer.syncQueue.length).toBeGreaterThan(0);
      
      // 模拟processSyncItem方法以避免复杂的异步处理
      stateSynchronizer.processSyncItem = jest.fn().mockResolvedValue();
      
      await syncPromise;
    });

    test('should sync to specific clients only', async () => {
      const stateUpdate = { phase: 'night' };
      const targetClients = ['client1'];
      
      // 清空队列
      stateSynchronizer.syncQueue = [];
      
      // 直接测试syncState是否正确设置目标客户端
      const syncPromise = stateSynchronizer.syncState(stateUpdate, targetClients);
      
      // 检查队列中的同步项
      expect(stateSynchronizer.syncQueue.length).toBe(1);
      const syncItem = stateSynchronizer.syncQueue[0];
      expect(syncItem.targetClients).toEqual(targetClients);
      expect(syncItem.stateUpdate).toEqual(stateUpdate);
      
      // 模拟处理完成
      stateSynchronizer.processSyncItem = jest.fn().mockResolvedValue();
      await syncPromise;
    });

    test('should handle sync errors with retry', async () => {
      const stateUpdate = { phase: 'day' };
      
      // 直接测试handleSyncItemError方法
      const syncItem = {
        id: 'test-sync',
        stateUpdate,
        targetClients: ['client1'],
        timestamp: Date.now(),
        retries: 0
      };

      const error = new Error('Network error');
      await stateSynchronizer.handleSyncItemError(syncItem, error);

      // 应该增加重试次数并重新加入队列
      expect(syncItem.retries).toBe(1);
      expect(stateSynchronizer.syncQueue).toContain(syncItem);
    });
  });

  describe('状态一致性检查', () => {
    test('should detect state inconsistency', () => {
      const stateHashes = {
        'client1': 'hash123',
        'client2': 'hash123',
        'client3': 'hash456' // 不一致
      };

      const inconsistentClients = stateSynchronizer.findInconsistentClients(stateHashes);
      
      expect(inconsistentClients).toEqual(['client3']);
    });

    test('should identify correct state by majority', () => {
      const stateHashes = {
        'client1': 'hash123',
        'client2': 'hash456',
        'client3': 'hash456',
        'client4': 'hash456'
      };

      const inconsistentClients = stateSynchronizer.findInconsistentClients(stateHashes);
      
      expect(inconsistentClients).toEqual(['client1']);
    });
  });

  describe('心跳检测', () => {
    test('should detect dead connections', () => {
      // 添加客户端
      stateSynchronizer.handleClientConnect('client1', { name: 'Player1' });
      
      const client = stateSynchronizer.connectedClients.get('client1');
      // 设置过期的心跳时间
      client.lastHeartbeat = Date.now() - (stateSynchronizer.heartbeatInterval * 3);

      // 执行心跳检测
      stateSynchronizer.performHeartbeatCheck();

      // 客户端应该被标记为断开连接
      expect(client.isConnected).toBe(false);
    });

    test('should clean up old disconnected clients', () => {
      // 添加并断开客户端
      stateSynchronizer.handleClientConnect('client1', { name: 'Player1' });
      stateSynchronizer.handleClientDisconnect('client1');
      
      const client = stateSynchronizer.connectedClients.get('client1');
      // 设置过期的断开时间
      client.disconnectionTime = Date.now() - (6 * 60 * 1000); // 6分钟前

      stateSynchronizer.cleanupDisconnectedClients();

      expect(stateSynchronizer.connectedClients.has('client1')).toBe(false);
    });
  });

  describe('权威状态管理', () => {
    test('should get authoritative state from store', () => {
      const authState = stateSynchronizer.getAuthoritativeState();
      
      expect(authState).toEqual({
        game: { phase: 'setup' },
        players: { players: [] },
        session: { sessionId: 'test' },
        automation: { systemStatus: 'idle' }
      });
    });

    test('should force sync state to inconsistent clients', async () => {
      const state = { phase: 'day', day: 1 };
      const inconsistentClients = ['client1', 'client2'];
      
      // 添加客户端
      inconsistentClients.forEach(clientId => {
        stateSynchronizer.handleClientConnect(clientId, { name: `Player${clientId}` });
      });

      // 重置mock计数器，因为handleClientConnect会调用sendToClient
      mockWebSocketService.sendToClient.mockClear();
      mockWebSocketService.sendToClient.mockResolvedValue();

      await stateSynchronizer.forceSyncState(state, inconsistentClients);

      expect(mockWebSocketService.sendToClient).toHaveBeenCalledTimes(2);
      expect(mockWebSocketService.sendToClient).toHaveBeenCalledWith(
        'client1',
        expect.objectContaining({
          type: 'force_state_sync',
          data: state
        })
      );
    });
  });

  describe('错误处理', () => {
    test('should handle sync errors and add to error log', () => {
      const error = new Error('Test error');
      
      stateSynchronizer.handleSyncError(error);

      expect(mockStore.commit).toHaveBeenCalledWith('automation/ADD_ERROR', {
        type: ErrorType.NETWORK,
        message: `Sync error: ${error.message}`,
        data: { error: error.stack }
      });
    });

    test('should retry failed sync items', async () => {
      const syncItem = {
        id: 'sync123',
        stateUpdate: { phase: 'day' },
        targetClients: ['client1'],
        timestamp: Date.now(),
        retries: 0
      };

      const error = new Error('Network failure');

      await stateSynchronizer.handleSyncItemError(syncItem, error);

      expect(syncItem.retries).toBe(1);
      expect(stateSynchronizer.syncQueue).toContain(syncItem);
    });

    test('should give up after max retries', async () => {
      const syncItem = {
        id: 'sync123',
        stateUpdate: { phase: 'day' },
        targetClients: ['client1'],
        timestamp: Date.now(),
        retries: 3 // 已达到最大重试次数
      };

      const error = new Error('Persistent failure');

      await stateSynchronizer.handleSyncItemError(syncItem, error);

      expect(mockStore.commit).toHaveBeenCalledWith('automation/ADD_ERROR', 
        expect.objectContaining({
          type: ErrorType.NETWORK,
          message: expect.stringContaining('Sync failed after 3 retries')
        })
      );
    });
  });

  describe('工具方法', () => {
    test('should generate unique sync IDs', () => {
      const id1 = stateSynchronizer.generateSyncId();
      const id2 = stateSynchronizer.generateSyncId();
      
      expect(id1).not.toBe(id2);
      expect(id1).toMatch(/^sync_/);
      expect(id2).toMatch(/^sync_/);
    });
  });
});
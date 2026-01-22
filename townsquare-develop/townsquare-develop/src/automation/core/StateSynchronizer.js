/**
 * 状态同步器
 * 负责确保所有客户端状态同步
 */

import { deepClone } from '../utils/GameUtils';
import { ErrorType, RecoveryStrategy } from '../types/AutomationTypes';

export default class StateSynchronizer {
  constructor(store, websocketService) {
    this.store = store;
    this.websocketService = websocketService;
    this.syncQueue = [];
    this.isProcessing = false;
    this.retryAttempts = new Map();
    this.maxRetries = 3;
    this.syncTimeout = 5000; // 5秒超时
    this.heartbeatInterval = 30000; // 30秒心跳
    this.lastSyncTimestamp = Date.now();
    
    // 客户端连接状态
    this.connectedClients = new Map();
    this.inconsistentClients = new Set();
    
    // 设置WebSocket监听器
    this.setupWebSocketListeners();
  }

  /**
   * 同步状态到所有客户端
   * @param {object} stateUpdate 状态更新
   * @param {string[]} targetClients 目标客户端列表，如果为空则同步到所有客户端
   * @returns {Promise<void>}
   */
  async syncState(stateUpdate, targetClients = null) {
    const syncItem = {
      id: this.generateSyncId(),
      stateUpdate: deepClone(stateUpdate),
      targetClients: targetClients || Array.from(this.connectedClients.keys()),
      timestamp: Date.now(),
      retries: 0
    };

    this.syncQueue.push(syncItem);
    
    if (!this.isProcessing) {
      await this.processSyncQueue();
    }
  }

  /**
   * 处理同步队列
   * @returns {Promise<void>}
   */
  async processSyncQueue() {
    if (this.isProcessing || this.syncQueue.length === 0) {
      return;
    }

    this.isProcessing = true;

    try {
      while (this.syncQueue.length > 0) {
        const syncItem = this.syncQueue.shift();
        await this.processSyncItem(syncItem);
      }
    } catch (error) {
      this.handleSyncError(error);
    } finally {
      this.isProcessing = false;
    }
  }

  /**
   * 处理单个同步项
   * @param {object} syncItem 同步项
   * @returns {Promise<void>}
   */
  async processSyncItem(syncItem) {
    try {
      const message = {
        type: 'state_sync',
        id: syncItem.id,
        data: syncItem.stateUpdate,
        timestamp: syncItem.timestamp
      };

      // 发送到目标客户端
      const promises = syncItem.targetClients.map(clientId => 
        this.sendToClient(clientId, message).catch(error => {
          // 捕获单个客户端的错误，但不阻止其他客户端
          this.store.commit('automation/ADD_LOG', {
            level: 'warn',
            message: `Failed to sync to client ${clientId}: ${error.message}`
          });
          return { error, clientId };
        })
      );

      // 等待所有客户端响应或超时
      const results = await Promise.allSettled(promises);
      
      // 检查是否有失败的客户端
      const failedClients = results
        .filter(result => result.status === 'fulfilled' && result.value && result.value.error)
        .map(result => result.value.clientId);

      if (failedClients.length > 0) {
        throw new Error(`Failed to sync to clients: ${failedClients.join(', ')}`);
      }

      // 验证同步结果
      await this.verifySyncResult(syncItem);

      this.lastSyncTimestamp = Date.now();
      
      this.store.commit('automation/ADD_LOG', {
        level: 'debug',
        message: `State synced to ${syncItem.targetClients.length} clients`,
        data: { syncId: syncItem.id }
      });

    } catch (error) {
      await this.handleSyncItemError(syncItem, error);
    }
  }

  /**
   * 发送消息到指定客户端
   * @param {string} clientId 客户端ID
   * @param {object} message 消息
   * @returns {Promise<void>}
   */
  async sendToClient(clientId, message) {
    return new Promise((resolve, reject) => {
      const client = this.connectedClients.get(clientId);
      if (!client || !client.isConnected) {
        reject(new Error(`Client ${clientId} is not connected`));
        return;
      }

      const timeout = setTimeout(() => {
        reject(new Error(`Sync timeout for client ${clientId}`));
      }, this.syncTimeout);

      try {
        // 通过WebSocket发送消息
        this.websocketService.sendToClient(clientId, message);
        
        // 等待客户端确认
        const confirmationHandler = (response) => {
          if (response.syncId === message.id) {
            clearTimeout(timeout);
            resolve();
          }
        };

        client.once('sync_confirmation', confirmationHandler);
        
      } catch (error) {
        clearTimeout(timeout);
        reject(error);
      }
    });
  }

  /**
   * 验证同步结果
   * @param {object} syncItem 同步项
   * @returns {Promise<void>}
   */
  async verifySyncResult(syncItem) {
    try {
      // 请求所有客户端的状态哈希进行验证
      const stateHashes = await this.collectStateHashes(syncItem.targetClients);
      
      // 检查状态一致性
      const uniqueHashes = new Set(Object.values(stateHashes));
      
      if (uniqueHashes.size > 1) {
        // 发现状态不一致
        const inconsistentClients = this.findInconsistentClients(stateHashes);
        this.inconsistentClients = new Set([...this.inconsistentClients, ...inconsistentClients]);
        
        this.store.commit('automation/ADD_ERROR', {
          type: ErrorType.NETWORK,
          message: `State inconsistency detected in clients: ${inconsistentClients.join(', ')}`,
          data: { syncId: syncItem.id, stateHashes }
        });

        // 触发状态修复
        await this.repairStateInconsistency(inconsistentClients);
      }
    } catch (error) {
      // 验证失败不应该阻止同步流程
      this.store.commit('automation/ADD_LOG', {
        level: 'warn',
        message: `Sync verification failed: ${error.message}`,
        data: { syncId: syncItem.id }
      });
    }
  }

  /**
   * 收集客户端状态哈希
   * @param {string[]} clientIds 客户端ID列表
   * @returns {Promise<object>}
   */
  async collectStateHashes(clientIds) {
    const promises = clientIds.map(clientId => 
      this.requestStateHash(clientId)
    );

    const results = await Promise.allSettled(promises);
    const stateHashes = {};

    results.forEach((result, index) => {
      if (result.status === 'fulfilled') {
        stateHashes[clientIds[index]] = result.value;
      }
    });

    return stateHashes;
  }

  /**
   * 请求客户端状态哈希
   * @param {string} clientId 客户端ID
   * @returns {Promise<string>}
   */
  async requestStateHash(clientId) {
    return new Promise((resolve, reject) => {
      const client = this.connectedClients.get(clientId);
      if (!client) {
        reject(new Error(`Client ${clientId} not found`));
        return;
      }

      const timeout = setTimeout(() => {
        reject(new Error(`State hash request timeout for client ${clientId}`));
      }, this.syncTimeout);

      const responseHandler = (hash) => {
        clearTimeout(timeout);
        resolve(hash);
      };

      client.once('state_hash_response', responseHandler);
      
      this.websocketService.sendToClient(clientId, {
        type: 'request_state_hash',
        timestamp: Date.now()
      });
    });
  }

  /**
   * 查找状态不一致的客户端
   * @param {object} stateHashes 状态哈希映射
   * @returns {string[]}
   */
  findInconsistentClients(stateHashes) {
    const hashCounts = {};
    
    // 统计每个哈希值的出现次数
    Object.entries(stateHashes).forEach(([clientId, hash]) => {
      if (!hashCounts[hash]) {
        hashCounts[hash] = [];
      }
      hashCounts[hash].push(clientId);
    });

    // 找到出现次数最多的哈希值（认为是正确的状态）
    const correctHash = Object.keys(hashCounts).reduce((a, b) => 
      hashCounts[a].length > hashCounts[b].length ? a : b
    );

    // 返回状态不一致的客户端
    return Object.entries(stateHashes)
      .filter(([clientId, hash]) => hash !== correctHash)
      .map(([clientId]) => clientId);
  }

  /**
   * 修复状态不一致
   * @param {string[]} inconsistentClients 不一致的客户端列表
   * @returns {Promise<void>}
   */
  async repairStateInconsistency(inconsistentClients) {
    try {
      // 获取权威状态（从游戏状态管理器）
      const authoritativeState = this.store.getters['automation/getCurrentState'] || 
                                this.getAuthoritativeState();

      // 强制同步权威状态到不一致的客户端
      await this.forceSyncState(authoritativeState, inconsistentClients);

      // 从不一致列表中移除已修复的客户端
      inconsistentClients.forEach(clientId => {
        this.inconsistentClients.delete(clientId);
      });

      this.store.commit('automation/ADD_LOG', {
        level: 'info',
        message: `State inconsistency repaired for ${inconsistentClients.length} clients`,
        data: { repairedClients: inconsistentClients }
      });

    } catch (error) {
      this.store.commit('automation/ADD_ERROR', {
        type: ErrorType.NETWORK,
        message: `Failed to repair state inconsistency: ${error.message}`,
        data: { inconsistentClients }
      });
    }
  }

  /**
   * 强制同步状态
   * @param {object} state 状态
   * @param {string[]} targetClients 目标客户端
   * @returns {Promise<void>}
   */
  async forceSyncState(state, targetClients) {
    const message = {
      type: 'force_state_sync',
      id: this.generateSyncId(),
      data: state,
      timestamp: Date.now()
    };

    const promises = targetClients.map(clientId => {
      const client = this.connectedClients.get(clientId);
      if (client && client.isConnected) {
        return this.websocketService.sendToClient(clientId, message);
      }
    });

    await Promise.allSettled(promises);
  }

  /**
   * 处理客户端连接
   * @param {string} clientId 客户端ID
   * @param {object} clientInfo 客户端信息
   */
  handleClientConnect(clientId, clientInfo) {
    this.connectedClients.set(clientId, {
      ...clientInfo,
      isConnected: true,
      lastHeartbeat: Date.now(),
      connectionTime: Date.now()
    });

    this.store.commit('automation/ADD_LOG', {
      level: 'info',
      message: `Client connected: ${clientId}`,
      data: { clientInfo }
    });

    // 异步发送当前状态给新连接的客户端，不阻塞连接流程
    this.sendCurrentStateToClient(clientId).catch(error => {
      this.store.commit('automation/ADD_ERROR', {
        type: ErrorType.NETWORK,
        message: `Failed to send initial state to client ${clientId}: ${error.message}`
      });
    });
  }

  /**
   * 处理客户端断开连接
   * @param {string} clientId 客户端ID
   */
  handleClientDisconnect(clientId) {
    const client = this.connectedClients.get(clientId);
    if (client) {
      client.isConnected = false;
      client.disconnectionTime = Date.now();
    }

    this.store.commit('automation/ADD_LOG', {
      level: 'info',
      message: `Client disconnected: ${clientId}`
    });
  }

  /**
   * 发送当前状态给客户端
   * @param {string} clientId 客户端ID
   */
  async sendCurrentStateToClient(clientId) {
    try {
      const currentState = this.getAuthoritativeState();
      await this.syncState(currentState, [clientId]);
    } catch (error) {
      this.store.commit('automation/ADD_ERROR', {
        type: ErrorType.NETWORK,
        message: `Failed to send current state to client ${clientId}: ${error.message}`
      });
    }
  }

  /**
   * 获取权威状态
   * @returns {object}
   */
  getAuthoritativeState() {
    // 从各个store模块获取当前状态
    return {
      game: this.store.state.game || {},
      players: this.store.state.players || {},
      session: this.store.state.session || {},
      automation: this.store.state.automation || {}
    };
  }

  /**
   * 启动心跳检测
   */
  startHeartbeat() {
    setInterval(() => {
      this.performHeartbeatCheck();
    }, this.heartbeatInterval);
  }

  /**
   * 执行心跳检测
   */
  performHeartbeatCheck() {
    const now = Date.now();
    const deadClients = [];

    this.connectedClients.forEach((client, clientId) => {
      if (client.isConnected && (now - client.lastHeartbeat) > this.heartbeatInterval * 2) {
        deadClients.push(clientId);
      }
    });

    // 处理死连接
    deadClients.forEach(clientId => {
      this.handleClientDisconnect(clientId);
    });

    if (deadClients.length > 0) {
      this.store.commit('automation/ADD_LOG', {
        level: 'warn',
        message: `Detected ${deadClients.length} dead connections`,
        data: { deadClients }
      });
    }
  }

  /**
   * 设置WebSocket监听器
   */
  setupWebSocketListeners() {
    if (this.websocketService) {
      this.websocketService.on('client_connect', (clientId, clientInfo) => {
        this.handleClientConnect(clientId, clientInfo);
      });

      this.websocketService.on('client_disconnect', (clientId) => {
        this.handleClientDisconnect(clientId);
      });

      this.websocketService.on('heartbeat', (clientId) => {
        const client = this.connectedClients.get(clientId);
        if (client) {
          client.lastHeartbeat = Date.now();
        }
      });
    }
  }

  /**
   * 处理同步错误
   * @param {Error} error 错误
   */
  handleSyncError(error) {
    this.store.commit('automation/ADD_ERROR', {
      type: ErrorType.NETWORK,
      message: `Sync error: ${error.message}`,
      data: { error: error.stack }
    });
  }

  /**
   * 处理同步项错误
   * @param {object} syncItem 同步项
   * @param {Error} error 错误
   * @returns {Promise<void>}
   */
  async handleSyncItemError(syncItem, error) {
    syncItem.retries += 1;

    if (syncItem.retries < this.maxRetries) {
      // 重试
      this.syncQueue.unshift(syncItem);
      
      this.store.commit('automation/ADD_LOG', {
        level: 'warn',
        message: `Sync retry ${syncItem.retries}/${this.maxRetries} for sync ${syncItem.id}`,
        data: { error: error.message }
      });
    } else {
      // 达到最大重试次数，记录错误
      this.store.commit('automation/ADD_ERROR', {
        type: ErrorType.NETWORK,
        message: `Sync failed after ${this.maxRetries} retries: ${error.message}`,
        data: { syncId: syncItem.id, targetClients: syncItem.targetClients }
      });
    }
  }

  /**
   * 生成同步ID
   * @returns {string}
   */
  generateSyncId() {
    return 'sync_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
  }

  /**
   * 获取连接统计信息
   * @returns {object}
   */
  getConnectionStats() {
    const connectedCount = Array.from(this.connectedClients.values())
      .filter(client => client.isConnected).length;
    
    return {
      totalClients: this.connectedClients.size,
      connectedClients: connectedCount,
      inconsistentClients: this.inconsistentClients.size,
      queueSize: this.syncQueue.length,
      lastSyncTimestamp: this.lastSyncTimestamp
    };
  }

  /**
   * 清理断开的客户端
   */
  cleanupDisconnectedClients() {
    const now = Date.now();
    const cleanupThreshold = 5 * 60 * 1000; // 5分钟

    this.connectedClients.forEach((client, clientId) => {
      if (!client.isConnected && 
          client.disconnectionTime && 
          (now - client.disconnectionTime) > cleanupThreshold) {
        this.connectedClients.delete(clientId);
        this.inconsistentClients.delete(clientId);
      }
    });
  }
}
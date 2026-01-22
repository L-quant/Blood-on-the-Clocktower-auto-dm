/**
 * 配置管理器
 * 负责游戏配置的保存、加载和验证
 */

import { AutomationLevel, AIDifficulty } from '../types/AutomationTypes';

/**
 * 默认配置
 */
const DEFAULT_CONFIG = {
  scriptType: 'trouble-brewing',
  playerCount: 7,
  gameMode: 'storyteller', // 'storyteller' 或 'player-only'
  automationLevel: AutomationLevel.FULL_AUTO,
  aiDifficulty: AIDifficulty.MEDIUM,
  timeSettings: {
    discussionTime: 300, // 5分钟
    nominationTime: 60,  // 1分钟
    votingTime: 30,      // 30秒
    nightActionTimeout: 60 // 1分钟
  },
  ruleVariants: [],
  debugMode: false,
  logLevel: 'info' // 'debug', 'info', 'warn', 'error'
};

/**
 * 配置管理器类
 */
export default class ConfigurationManager {
  constructor() {
    this.currentConfig = null;
    this.configHistory = [];
    this.maxHistorySize = 10;
    this.debugMode = false;
    this.logBuffer = [];
    this.maxLogSize = 1000;
  }

  /**
   * 获取默认配置
   * @returns {object} 默认配置对象
   */
  getDefaultConfig() {
    return JSON.parse(JSON.stringify(DEFAULT_CONFIG));
  }

  /**
   * 创建新配置
   * @param {object} options 配置选项
   * @returns {object} 创建的配置对象
   */
  createConfig(options = {}) {
    const config = {
      ...this.getDefaultConfig(),
      ...options,
      timeSettings: {
        ...DEFAULT_CONFIG.timeSettings,
        ...(options.timeSettings || {})
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
      version: '1.0.0'
    };

    // 验证配置
    const validation = this.validateConfig(config);
    if (!validation.valid) {
      throw new Error(`Invalid configuration: ${validation.errors.join(', ')}`);
    }

    this.currentConfig = config;
    this._addToHistory(config);

    this.log('info', 'Configuration created', config);

    return config;
  }

  /**
   * 加载配置
   * @param {object} config 配置对象
   * @returns {object} 加载的配置对象
   */
  loadConfig(config) {
    if (!config) {
      throw new Error('Configuration is required');
    }

    // 验证配置
    const validation = this.validateConfig(config);
    if (!validation.valid) {
      throw new Error(`Invalid configuration: ${validation.errors.join(', ')}`);
    }

    // 更新时间戳
    config.updatedAt = Date.now();

    this.currentConfig = config;
    this._addToHistory(config);

    this.log('info', 'Configuration loaded', config);

    return config;
  }

  /**
   * 保存配置
   * @param {object} config 要保存的配置（可选，默认保存当前配置）
   * @returns {string} 配置的JSON字符串
   */
  saveConfig(config = null) {
    const configToSave = config || this.currentConfig;

    if (!configToSave) {
      throw new Error('No configuration to save');
    }

    // 验证配置
    const validation = this.validateConfig(configToSave);
    if (!validation.valid) {
      throw new Error(`Cannot save invalid configuration: ${validation.errors.join(', ')}`);
    }

    // 更新时间戳
    configToSave.updatedAt = Date.now();

    const configJson = JSON.stringify(configToSave, null, 2);

    this.log('info', 'Configuration saved', configToSave);

    return configJson;
  }

  /**
   * 从JSON字符串加载配置
   * @param {string} configJson 配置的JSON字符串
   * @returns {object} 加载的配置对象
   */
  loadFromJson(configJson) {
    try {
      const config = JSON.parse(configJson);
      return this.loadConfig(config);
    } catch (error) {
      throw new Error(`Failed to parse configuration JSON: ${error.message}`);
    }
  }

  /**
   * 验证配置
   * @param {object} config 要验证的配置
   * @returns {object} 验证结果 { valid: boolean, errors: string[] }
   */
  validateConfig(config) {
    const errors = [];

    // 验证必需字段
    if (!config.scriptType) {
      errors.push('scriptType is required');
    }

    if (typeof config.playerCount !== 'number') {
      errors.push('playerCount must be a number');
    } else if (config.playerCount < 5 || config.playerCount > 20) {
      errors.push('playerCount must be between 5 and 20');
    }

    // 验证游戏模式
    const validGameModes = ['storyteller', 'player-only'];
    if (config.gameMode && !validGameModes.includes(config.gameMode)) {
      errors.push(`gameMode must be one of: ${validGameModes.join(', ')}`);
    }

    // 验证自动化级别
    const validAutomationLevels = Object.values(AutomationLevel);
    if (!validAutomationLevels.includes(config.automationLevel)) {
      errors.push(`automationLevel must be one of: ${validAutomationLevels.join(', ')}`);
    }

    // 验证AI难度
    const validAIDifficulties = Object.values(AIDifficulty);
    if (!validAIDifficulties.includes(config.aiDifficulty)) {
      errors.push(`aiDifficulty must be one of: ${validAIDifficulties.join(', ')}`);
    }

    // 验证时间设置
    if (config.timeSettings) {
      if (typeof config.timeSettings.discussionTime !== 'number' || config.timeSettings.discussionTime < 0) {
        errors.push('timeSettings.discussionTime must be a non-negative number');
      }
      if (typeof config.timeSettings.nominationTime !== 'number' || config.timeSettings.nominationTime < 0) {
        errors.push('timeSettings.nominationTime must be a non-negative number');
      }
      if (typeof config.timeSettings.votingTime !== 'number' || config.timeSettings.votingTime < 0) {
        errors.push('timeSettings.votingTime must be a non-negative number');
      }
      if (typeof config.timeSettings.nightActionTimeout !== 'number' || config.timeSettings.nightActionTimeout < 0) {
        errors.push('timeSettings.nightActionTimeout must be a non-negative number');
      }
    } else {
      errors.push('timeSettings is required');
    }

    // 验证规则变体
    if (!Array.isArray(config.ruleVariants)) {
      errors.push('ruleVariants must be an array');
    }

    // 验证调试模式
    if (typeof config.debugMode !== 'boolean') {
      errors.push('debugMode must be a boolean');
    }

    return {
      valid: errors.length === 0,
      errors
    };
  }

  /**
   * 获取当前配置
   * @returns {object} 当前配置对象
   */
  getCurrentConfig() {
    return this.currentConfig;
  }

  /**
   * 更新配置
   * @param {object} updates 要更新的配置项
   * @returns {object} 更新后的配置对象
   */
  updateConfig(updates) {
    if (!this.currentConfig) {
      throw new Error('No current configuration to update');
    }

    const updatedConfig = {
      ...this.currentConfig,
      ...updates,
      timeSettings: {
        ...this.currentConfig.timeSettings,
        ...(updates.timeSettings || {})
      },
      updatedAt: Date.now()
    };

    // 验证更新后的配置
    const validation = this.validateConfig(updatedConfig);
    if (!validation.valid) {
      throw new Error(`Invalid configuration update: ${validation.errors.join(', ')}`);
    }

    this.currentConfig = updatedConfig;
    this._addToHistory(updatedConfig);

    this.log('info', 'Configuration updated', updates);

    return updatedConfig;
  }

  /**
   * 重置为默认配置
   * @returns {object} 默认配置对象
   */
  resetToDefault() {
    this.currentConfig = this.getDefaultConfig();
    this._addToHistory(this.currentConfig);

    this.log('info', 'Configuration reset to default');

    return this.currentConfig;
  }

  /**
   * 启用调试模式
   */
  enableDebugMode() {
    this.debugMode = true;
    if (this.currentConfig) {
      this.currentConfig.debugMode = true;
      this.currentConfig.logLevel = 'debug';
    }
    this.log('info', 'Debug mode enabled');
  }

  /**
   * 禁用调试模式
   */
  disableDebugMode() {
    this.debugMode = false;
    if (this.currentConfig) {
      this.currentConfig.debugMode = false;
      this.currentConfig.logLevel = 'info';
    }
    this.log('info', 'Debug mode disabled');
  }

  /**
   * 记录日志
   * @param {string} level 日志级别 ('debug', 'info', 'warn', 'error')
   * @param {string} message 日志消息
   * @param {*} data 附加数据
   */
  log(level, message, data = null) {
    const logEntry = {
      timestamp: Date.now(),
      level,
      message,
      data
    };

    // 添加到日志缓冲区
    this.logBuffer.push(logEntry);

    // 限制日志缓冲区大小
    if (this.logBuffer.length > this.maxLogSize) {
      this.logBuffer.shift();
    }

    // 如果启用调试模式，输出到控制台
    if (this.debugMode || (this.currentConfig && this.currentConfig.debugMode)) {
      const logLevels = ['debug', 'info', 'warn', 'error'];
      const currentLogLevel = this.currentConfig?.logLevel || 'info';
      const currentLevelIndex = logLevels.indexOf(currentLogLevel);
      const messageLevelIndex = logLevels.indexOf(level);

      if (messageLevelIndex >= currentLevelIndex) {
        const timestamp = new Date(logEntry.timestamp).toISOString();
        console.log(`[${timestamp}] [${level.toUpperCase()}] ${message}`, data || '');
      }
    }
  }

  /**
   * 获取日志
   * @param {object} options 过滤选项
   * @returns {Array} 日志条目数组
   */
  getLogs(options = {}) {
    let logs = [...this.logBuffer];

    // 按级别过滤
    if (options.level) {
      logs = logs.filter(log => log.level === options.level);
    }

    // 按时间范围过滤
    if (options.startTime) {
      logs = logs.filter(log => log.timestamp >= options.startTime);
    }
    if (options.endTime) {
      logs = logs.filter(log => log.timestamp <= options.endTime);
    }

    // 限制数量
    if (options.limit) {
      logs = logs.slice(-options.limit);
    }

    return logs;
  }

  /**
   * 清除日志
   */
  clearLogs() {
    this.logBuffer = [];
    this.log('info', 'Logs cleared');
  }

  /**
   * 获取配置历史
   * @returns {Array} 配置历史数组
   */
  getConfigHistory() {
    return [...this.configHistory];
  }

  /**
   * 回滚到上一个配置
   * @returns {object} 回滚后的配置对象
   */
  rollbackConfig() {
    if (this.configHistory.length < 2) {
      throw new Error('No previous configuration to rollback to');
    }

    // 移除当前配置
    this.configHistory.pop();

    // 获取上一个配置
    const previousConfig = this.configHistory[this.configHistory.length - 1];
    this.currentConfig = JSON.parse(JSON.stringify(previousConfig));

    this.log('info', 'Configuration rolled back');

    return this.currentConfig;
  }

  /**
   * 导出配置为可读格式
   * @returns {string} 格式化的配置字符串
   */
  exportConfig() {
    if (!this.currentConfig) {
      throw new Error('No configuration to export');
    }

    const lines = [];
    lines.push('=== 游戏配置 ===');
    lines.push('');
    lines.push(`脚本类型: ${this.currentConfig.scriptType}`);
    lines.push(`玩家数量: ${this.currentConfig.playerCount}`);
    lines.push(`游戏模式: ${this.currentConfig.gameMode === 'player-only' ? '无说书人模式' : '说书人模式'}`);
    lines.push(`自动化级别: ${this.currentConfig.automationLevel}`);
    lines.push(`AI难度: ${this.currentConfig.aiDifficulty}`);
    lines.push('');
    lines.push('时间设置:');
    lines.push(`  讨论时间: ${this.currentConfig.timeSettings.discussionTime}秒`);
    lines.push(`  提名时间: ${this.currentConfig.timeSettings.nominationTime}秒`);
    lines.push(`  投票时间: ${this.currentConfig.timeSettings.votingTime}秒`);
    lines.push(`  夜间行动超时: ${this.currentConfig.timeSettings.nightActionTimeout}秒`);
    lines.push('');
    lines.push(`规则变体: ${this.currentConfig.ruleVariants.length > 0 ? this.currentConfig.ruleVariants.join(', ') : '无'}`);
    lines.push(`调试模式: ${this.currentConfig.debugMode ? '启用' : '禁用'}`);
    lines.push(`日志级别: ${this.currentConfig.logLevel}`);
    lines.push('');
    lines.push(`创建时间: ${new Date(this.currentConfig.createdAt).toLocaleString()}`);
    lines.push(`更新时间: ${new Date(this.currentConfig.updatedAt).toLocaleString()}`);
    lines.push(`版本: ${this.currentConfig.version}`);

    return lines.join('\n');
  }

  // 私有方法

  /**
   * 添加配置到历史记录
   * @private
   */
  _addToHistory(config) {
    // 深拷贝配置
    const configCopy = JSON.parse(JSON.stringify(config));
    this.configHistory.push(configCopy);

    // 限制历史记录大小
    if (this.configHistory.length > this.maxHistorySize) {
      this.configHistory.shift();
    }
  }
}

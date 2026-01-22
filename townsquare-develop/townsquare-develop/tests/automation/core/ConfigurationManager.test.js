/**
 * 配置管理器测试
 */

import ConfigurationManager from '../../../src/automation/core/ConfigurationManager';
import { AutomationLevel, AIDifficulty } from '../../../src/automation/types/AutomationTypes';

describe('ConfigurationManager', () => {
  let configManager;

  beforeEach(() => {
    configManager = new ConfigurationManager();
  });

  describe('构造函数', () => {
    test('应该正确初始化配置管理器', () => {
      expect(configManager).toBeDefined();
      expect(configManager.currentConfig).toBeNull();
      expect(configManager.configHistory).toEqual([]);
      expect(configManager.debugMode).toBe(false);
      expect(configManager.logBuffer).toEqual([]);
    });
  });

  describe('getDefaultConfig', () => {
    test('应该返回默认配置', () => {
      const defaultConfig = configManager.getDefaultConfig();

      expect(defaultConfig).toBeDefined();
      expect(defaultConfig.scriptType).toBe('trouble-brewing');
      expect(defaultConfig.playerCount).toBe(7);
      expect(defaultConfig.automationLevel).toBe(AutomationLevel.FULL_AUTO);
      expect(defaultConfig.aiDifficulty).toBe(AIDifficulty.MEDIUM);
      expect(defaultConfig.timeSettings).toBeDefined();
      expect(defaultConfig.ruleVariants).toEqual([]);
      expect(defaultConfig.debugMode).toBe(false);
    });

    test('应该返回独立的配置副本', () => {
      const config1 = configManager.getDefaultConfig();
      const config2 = configManager.getDefaultConfig();

      config1.playerCount = 10;

      expect(config2.playerCount).toBe(7);
    });
  });

  describe('createConfig', () => {
    test('应该创建新配置', () => {
      const config = configManager.createConfig();

      expect(config).toBeDefined();
      expect(config.createdAt).toBeDefined();
      expect(config.updatedAt).toBeDefined();
      expect(config.version).toBe('1.0.0');
      expect(configManager.currentConfig).toBe(config);
    });

    test('应该使用提供的选项创建配置', () => {
      const config = configManager.createConfig({
        scriptType: 'sects-violets',
        playerCount: 10,
        aiDifficulty: AIDifficulty.HARD
      });

      expect(config.scriptType).toBe('sects-violets');
      expect(config.playerCount).toBe(10);
      expect(config.aiDifficulty).toBe(AIDifficulty.HARD);
    });

    test('应该合并时间设置', () => {
      const config = configManager.createConfig({
        timeSettings: {
          discussionTime: 600
        }
      });

      expect(config.timeSettings.discussionTime).toBe(600);
      expect(config.timeSettings.nominationTime).toBe(60); // 保持默认值
    });

    test('应该验证配置', () => {
      expect(() => {
        configManager.createConfig({
          playerCount: 3 // 无效：小于5
        });
      }).toThrow('Invalid configuration');
    });

    test('应该添加配置到历史记录', () => {
      configManager.createConfig();

      expect(configManager.configHistory.length).toBe(1);
    });
  });

  describe('loadConfig', () => {
    test('应该加载有效配置', () => {
      const config = {
        scriptType: 'bad-moon-rising',
        playerCount: 8,
        automationLevel: AutomationLevel.FULL_AUTO,
        aiDifficulty: AIDifficulty.EASY,
        timeSettings: {
          discussionTime: 300,
          nominationTime: 60,
          votingTime: 30,
          nightActionTimeout: 60
        },
        ruleVariants: [],
        debugMode: false
      };

      const loadedConfig = configManager.loadConfig(config);

      expect(loadedConfig).toBeDefined();
      expect(loadedConfig.scriptType).toBe('bad-moon-rising');
      expect(loadedConfig.updatedAt).toBeDefined();
      expect(configManager.currentConfig).toBe(loadedConfig);
    });

    test('应该拒绝null配置', () => {
      expect(() => {
        configManager.loadConfig(null);
      }).toThrow('Configuration is required');
    });

    test('应该验证加载的配置', () => {
      const invalidConfig = {
        scriptType: 'test',
        playerCount: 100, // 无效：大于20
        automationLevel: AutomationLevel.FULL,
        aiDifficulty: AIDifficulty.MEDIUM,
        timeSettings: {
          discussionTime: 300,
          nominationTime: 60,
          votingTime: 30,
          nightActionTimeout: 60
        },
        ruleVariants: [],
        debugMode: false
      };

      expect(() => {
        configManager.loadConfig(invalidConfig);
      }).toThrow('Invalid configuration');
    });
  });

  describe('saveConfig', () => {
    test('应该保存当前配置', () => {
      configManager.createConfig();
      const savedJson = configManager.saveConfig();

      expect(savedJson).toBeDefined();
      expect(typeof savedJson).toBe('string');

      const parsed = JSON.parse(savedJson);
      expect(parsed.scriptType).toBe('trouble-brewing');
    });

    test('应该保存指定的配置', () => {
      const config = configManager.createConfig({ playerCount: 10 });
      const savedJson = configManager.saveConfig(config);

      const parsed = JSON.parse(savedJson);
      expect(parsed.playerCount).toBe(10);
    });

    test('应该在没有配置时抛出错误', () => {
      expect(() => {
        configManager.saveConfig();
      }).toThrow('No configuration to save');
    });

    test('应该更新时间戳', () => {
      configManager.createConfig();
      const originalTime = configManager.currentConfig.updatedAt;

      // 等待一小段时间
      setTimeout(() => {
        configManager.saveConfig();
        expect(configManager.currentConfig.updatedAt).toBeGreaterThan(originalTime);
      }, 10);
    });
  });

  describe('loadFromJson', () => {
    test('应该从JSON字符串加载配置', () => {
      const config = configManager.createConfig({ playerCount: 12 });
      const json = configManager.saveConfig(config);

      const newManager = new ConfigurationManager();
      const loadedConfig = newManager.loadFromJson(json);

      expect(loadedConfig.playerCount).toBe(12);
    });

    test('应该处理无效的JSON', () => {
      expect(() => {
        configManager.loadFromJson('invalid json');
      }).toThrow('Failed to parse configuration JSON');
    });
  });

  describe('validateConfig', () => {
    test('应该验证有效配置', () => {
      const config = configManager.getDefaultConfig();
      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(true);
      expect(validation.errors).toEqual([]);
    });

    test('应该检测缺少scriptType', () => {
      const config = configManager.getDefaultConfig();
      delete config.scriptType;

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('scriptType is required');
    });

    test('应该检测无效的playerCount', () => {
      const config = configManager.getDefaultConfig();
      config.playerCount = 3;

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('playerCount'))).toBe(true);
    });

    test('应该检测无效的automationLevel', () => {
      const config = configManager.getDefaultConfig();
      config.automationLevel = 'invalid';

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('automationLevel'))).toBe(true);
    });

    test('应该检测无效的aiDifficulty', () => {
      const config = configManager.getDefaultConfig();
      config.aiDifficulty = 'invalid';

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('aiDifficulty'))).toBe(true);
    });

    test('应该检测无效的timeSettings', () => {
      const config = configManager.getDefaultConfig();
      config.timeSettings.discussionTime = -1;

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('discussionTime'))).toBe(true);
    });

    test('应该检测缺少timeSettings', () => {
      const config = configManager.getDefaultConfig();
      delete config.timeSettings;

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('timeSettings is required');
    });

    test('应该检测无效的ruleVariants', () => {
      const config = configManager.getDefaultConfig();
      config.ruleVariants = 'not an array';

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('ruleVariants must be an array');
    });

    test('应该检测无效的debugMode', () => {
      const config = configManager.getDefaultConfig();
      config.debugMode = 'not a boolean';

      const validation = configManager.validateConfig(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('debugMode must be a boolean');
    });
  });

  describe('getCurrentConfig', () => {
    test('应该返回当前配置', () => {
      const config = configManager.createConfig();
      const current = configManager.getCurrentConfig();

      expect(current).toBe(config);
    });

    test('应该在没有配置时返回null', () => {
      expect(configManager.getCurrentConfig()).toBeNull();
    });
  });

  describe('updateConfig', () => {
    test('应该更新配置', () => {
      configManager.createConfig();
      const updated = configManager.updateConfig({ playerCount: 15 });

      expect(updated.playerCount).toBe(15);
      expect(configManager.currentConfig.playerCount).toBe(15);
    });

    test('应该合并时间设置更新', () => {
      configManager.createConfig();
      const updated = configManager.updateConfig({
        timeSettings: { discussionTime: 600 }
      });

      expect(updated.timeSettings.discussionTime).toBe(600);
      expect(updated.timeSettings.nominationTime).toBe(60); // 保持原值
    });

    test('应该在没有当前配置时抛出错误', () => {
      expect(() => {
        configManager.updateConfig({ playerCount: 10 });
      }).toThrow('No current configuration to update');
    });

    test('应该验证更新后的配置', () => {
      configManager.createConfig();

      expect(() => {
        configManager.updateConfig({ playerCount: 100 });
      }).toThrow('Invalid configuration update');
    });

    test('应该添加更新到历史记录', () => {
      configManager.createConfig();
      const initialHistoryLength = configManager.configHistory.length;

      configManager.updateConfig({ playerCount: 10 });

      expect(configManager.configHistory.length).toBe(initialHistoryLength + 1);
    });
  });

  describe('resetToDefault', () => {
    test('应该重置为默认配置', () => {
      configManager.createConfig({ playerCount: 15 });
      const reset = configManager.resetToDefault();

      expect(reset.playerCount).toBe(7);
      expect(configManager.currentConfig.playerCount).toBe(7);
    });
  });

  describe('调试模式', () => {
    test('应该启用调试模式', () => {
      configManager.createConfig();
      configManager.enableDebugMode();

      expect(configManager.debugMode).toBe(true);
      expect(configManager.currentConfig.debugMode).toBe(true);
      expect(configManager.currentConfig.logLevel).toBe('debug');
    });

    test('应该禁用调试模式', () => {
      configManager.createConfig();
      configManager.enableDebugMode();
      configManager.disableDebugMode();

      expect(configManager.debugMode).toBe(false);
      expect(configManager.currentConfig.debugMode).toBe(false);
      expect(configManager.currentConfig.logLevel).toBe('info');
    });
  });

  describe('日志记录', () => {
    test('应该记录日志', () => {
      configManager.log('info', 'Test message', { test: true });

      expect(configManager.logBuffer.length).toBe(1);
      expect(configManager.logBuffer[0].level).toBe('info');
      expect(configManager.logBuffer[0].message).toBe('Test message');
      expect(configManager.logBuffer[0].data).toEqual({ test: true });
    });

    test('应该限制日志缓冲区大小', () => {
      configManager.maxLogSize = 5;

      for (let i = 0; i < 10; i++) {
        configManager.log('info', `Message ${i}`);
      }

      expect(configManager.logBuffer.length).toBe(5);
      expect(configManager.logBuffer[0].message).toBe('Message 5');
    });

    test('应该获取日志', () => {
      configManager.log('info', 'Info message');
      configManager.log('error', 'Error message');
      configManager.log('debug', 'Debug message');

      const logs = configManager.getLogs();

      expect(logs.length).toBe(3);
    });

    test('应该按级别过滤日志', () => {
      configManager.log('info', 'Info message');
      configManager.log('error', 'Error message');

      const errorLogs = configManager.getLogs({ level: 'error' });

      expect(errorLogs.length).toBe(1);
      expect(errorLogs[0].level).toBe('error');
    });

    test('应该按时间范围过滤日志', () => {
      const startTime = Date.now();
      configManager.log('info', 'Message 1');
      configManager.log('info', 'Message 2');
      const endTime = Date.now();

      const logs = configManager.getLogs({ startTime, endTime });

      expect(logs.length).toBeGreaterThan(0);
    });

    test('应该限制日志数量', () => {
      for (let i = 0; i < 10; i++) {
        configManager.log('info', `Message ${i}`);
      }

      const logs = configManager.getLogs({ limit: 5 });

      expect(logs.length).toBe(5);
    });

    test('应该清除日志', () => {
      configManager.log('info', 'Test message');
      configManager.clearLogs();

      expect(configManager.logBuffer.length).toBe(1); // 只有清除日志的记录
    });
  });

  describe('配置历史', () => {
    test('应该获取配置历史', () => {
      configManager.createConfig();
      configManager.updateConfig({ playerCount: 10 });

      const history = configManager.getConfigHistory();

      expect(history.length).toBe(2);
    });

    test('应该限制历史记录大小', () => {
      configManager.maxHistorySize = 3;

      for (let i = 0; i < 5; i++) {
        configManager.createConfig({ playerCount: 5 + i });
      }

      expect(configManager.configHistory.length).toBe(3);
    });

    test('应该回滚配置', () => {
      configManager.createConfig({ playerCount: 7 });
      configManager.updateConfig({ playerCount: 10 });

      const rolledBack = configManager.rollbackConfig();

      expect(rolledBack.playerCount).toBe(7);
    });

    test('应该在没有历史时拒绝回滚', () => {
      configManager.createConfig();

      expect(() => {
        configManager.rollbackConfig();
      }).toThrow('No previous configuration to rollback to');
    });
  });

  describe('exportConfig', () => {
    test('应该导出配置为可读格式', () => {
      configManager.createConfig();
      const exported = configManager.exportConfig();

      expect(exported).toBeDefined();
      expect(typeof exported).toBe('string');
      expect(exported).toContain('游戏配置');
      expect(exported).toContain('脚本类型');
      expect(exported).toContain('玩家数量');
    });

    test('应该在没有配置时抛出错误', () => {
      expect(() => {
        configManager.exportConfig();
      }).toThrow('No configuration to export');
    });
  });
});

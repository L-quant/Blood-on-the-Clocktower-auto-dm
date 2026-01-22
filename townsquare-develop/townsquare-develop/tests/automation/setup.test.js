/**
 * 自动化系统设置测试
 */

import { GamePhase, Team, Player, Role, GameConfiguration } from '@/automation/types/GameTypes';
import { AutomationLevel, AIDifficulty, SystemStatus } from '@/automation/types/AutomationTypes';

describe('Automation System Setup', () => {
  describe('GameTypes', () => {
    test('should create Player instance correctly', () => {
      const player = new Player('player1', 'Alice');
      
      expect(player.id).toBe('player1');
      expect(player.name).toBe('Alice');
      expect(player.isAlive).toBe(true);
      expect(player.isEvil).toBe(false);
      expect(player.role).toBeNull();
    });

    test('should create Role instance correctly', () => {
      const roleData = {
        id: 'washerwoman',
        name: 'Washerwoman',
        team: Team.TOWNSFOLK,
        ability: 'You start knowing that 1 of 2 players is a particular Townsfolk.',
        firstNight: 33
      };
      
      const role = new Role(roleData);
      
      expect(role.id).toBe('washerwoman');
      expect(role.name).toBe('Washerwoman');
      expect(role.team).toBe(Team.TOWNSFOLK);
      expect(role.isGood).toBe(true);
      expect(role.isEvil).toBe(false);
    });

    test('should identify evil roles correctly', () => {
      const minionRole = new Role({
        id: 'poisoner',
        name: 'Poisoner',
        team: Team.MINION,
        ability: 'Each night, choose a player: they are poisoned tonight and tomorrow day.'
      });
      
      const demonRole = new Role({
        id: 'imp',
        name: 'Imp',
        team: Team.DEMON,
        ability: 'Each night*, choose a player: they die.'
      });
      
      expect(minionRole.isEvil).toBe(true);
      expect(minionRole.isGood).toBe(false);
      expect(demonRole.isEvil).toBe(true);
      expect(demonRole.isGood).toBe(false);
    });

    test('should create GameConfiguration with defaults', () => {
      const config = new GameConfiguration();
      
      expect(config.scriptType).toBe('trouble-brewing');
      expect(config.automationLevel).toBe('full');
      expect(config.aiDifficulty).toBe('medium');
      expect(config.debugMode).toBe(false);
      expect(config.timeSettings.discussionTime).toBe(300000);
    });
  });

  describe('AutomationTypes', () => {
    test('should have correct enum values', () => {
      expect(AutomationLevel.MANUAL).toBe('manual');
      expect(AutomationLevel.SEMI_AUTO).toBe('semi_auto');
      expect(AutomationLevel.FULL_AUTO).toBe('full_auto');
      
      expect(AIDifficulty.EASY).toBe('easy');
      expect(AIDifficulty.MEDIUM).toBe('medium');
      expect(AIDifficulty.HARD).toBe('hard');
      expect(AIDifficulty.EXPERT).toBe('expert');
      
      expect(SystemStatus.IDLE).toBe('idle');
      expect(SystemStatus.RUNNING).toBe('running');
      expect(SystemStatus.PAUSED).toBe('paused');
      expect(SystemStatus.ERROR).toBe('error');
    });
  });

  describe('Module Structure', () => {
    test('should have correct directory structure', () => {
      // 这个测试验证我们的模块结构是否正确设置
      expect(() => {
        require('@/automation/types/GameTypes');
        require('@/automation/types/AutomationTypes');
        require('@/automation/interfaces/IAutomatedStorytellerSystem');
      }).not.toThrow();
    });
  });
});
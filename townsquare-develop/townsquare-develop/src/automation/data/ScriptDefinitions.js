/**
 * 官方脚本定义
 * 包含Trouble Brewing, Bad Moon Rising, Sects & Violets等官方脚本
 */

import rolesJSON from '../../roles.json';
import { Team } from '../types/GameTypes';

/**
 * 脚本类型枚举
 */
export const ScriptType = {
  TROUBLE_BREWING: 'trouble-brewing',
  BAD_MOON_RISING: 'bad-moon-rising',
  SECTS_AND_VIOLETS: 'sects-and-violets',
  CUSTOM: 'custom'
};

/**
 * 脚本定义类
 */
export class ScriptDefinition {
  constructor(id, name, edition, roles) {
    this.id = id;
    this.name = name;
    this.edition = edition;
    this.roles = roles; // 角色ID列表
  }

  /**
   * 获取脚本中的所有角色
   * @returns {Role[]}
   */
  getRoles() {
    return rolesJSON.filter(role => 
      this.roles.includes(role.id) || role.edition === this.edition
    );
  }

  /**
   * 按团队获取角色
   * @param {string} team 团队类型
   * @returns {Role[]}
   */
  getRolesByTeam(team) {
    return this.getRoles().filter(role => role.team === team);
  }

  /**
   * 获取村民角色
   * @returns {Role[]}
   */
  getTownsfolk() {
    return this.getRolesByTeam(Team.TOWNSFOLK);
  }

  /**
   * 获取外来者角色
   * @returns {Role[]}
   */
  getOutsiders() {
    return this.getRolesByTeam(Team.OUTSIDER);
  }

  /**
   * 获取爪牙角色
   * @returns {Role[]}
   */
  getMinions() {
    return this.getRolesByTeam(Team.MINION);
  }

  /**
   * 获取恶魔角色
   * @returns {Role[]}
   */
  getDemons() {
    return this.getRolesByTeam(Team.DEMON);
  }

  /**
   * 获取设置阶段角色（影响角色分配的角色）
   * @returns {Role[]}
   */
  getSetupRoles() {
    return this.getRoles().filter(role => role.setup === true);
  }
}

/**
 * Trouble Brewing 脚本
 */
export const TroubleBrewingScript = new ScriptDefinition(
  ScriptType.TROUBLE_BREWING,
  'Trouble Brewing',
  'tb',
  [
    // Townsfolk
    'washerwoman', 'librarian', 'investigator', 'chef', 'empath',
    'fortuneteller', 'undertaker', 'monk', 'ravenkeeper', 'virgin',
    'slayer', 'soldier', 'mayor',
    // Outsiders
    'butler', 'drunk', 'recluse', 'saint',
    // Minions
    'poisoner', 'spy', 'scarletwoman', 'baron',
    // Demons
    'imp'
  ]
);

/**
 * Bad Moon Rising 脚本
 */
export const BadMoonRisingScript = new ScriptDefinition(
  ScriptType.BAD_MOON_RISING,
  'Bad Moon Rising',
  'bmr',
  [
    // Townsfolk
    'grandmother', 'sailor', 'chambermaid', 'exorcist', 'innkeeper',
    'gambler', 'gossip', 'courtier', 'professor', 'minstrel',
    'tealady', 'pacifist', 'fool',
    // Outsiders
    'tinker', 'moonchild', 'goon', 'lunatic',
    // Minions
    'godfather', 'devilsadvocate', 'assassin', 'mastermind',
    // Demons
    'zombuul', 'pukka', 'shabaloth', 'po'
  ]
);

/**
 * Sects & Violets 脚本
 */
export const SectsAndVioletsScript = new ScriptDefinition(
  ScriptType.SECTS_AND_VIOLETS,
  'Sects & Violets',
  'snv',
  [
    // Townsfolk
    'clockmaker', 'dreamer', 'snakecharmer', 'mathematician', 'flowergirl',
    'towncrier', 'oracle', 'savant', 'seamstress', 'philosopher',
    'artist', 'juggler', 'sage',
    // Outsiders
    'mutant', 'sweetheart', 'barber', 'klutz',
    // Minions
    'eviltwin', 'witch', 'cerenovus', 'pithag',
    // Demons
    'fanggu', 'vigormortis', 'nodashii', 'vortox'
  ]
);

/**
 * 获取脚本定义
 * @param {string} scriptType 脚本类型
 * @returns {ScriptDefinition}
 */
export function getScriptDefinition(scriptType) {
  switch (scriptType) {
    case ScriptType.TROUBLE_BREWING:
      return TroubleBrewingScript;
    case ScriptType.BAD_MOON_RISING:
      return BadMoonRisingScript;
    case ScriptType.SECTS_AND_VIOLETS:
      return SectsAndVioletsScript;
    default:
      return TroubleBrewingScript; // 默认使用Trouble Brewing
  }
}

/**
 * 获取所有可用脚本
 * @returns {ScriptDefinition[]}
 */
export function getAllScripts() {
  return [
    TroubleBrewingScript,
    BadMoonRisingScript,
    SectsAndVioletsScript
  ];
}

/**
 * 角色组合配置
 * 基于玩家数量的标准角色分配
 */
export const RoleCompositionTable = [
  null, // 0 players
  null, // 1 player
  null, // 2 players
  null, // 3 players
  null, // 4 players
  { townsfolk: 3, outsider: 0, minion: 1, demon: 1 }, // 5 players
  { townsfolk: 3, outsider: 1, minion: 1, demon: 1 }, // 6 players
  { townsfolk: 5, outsider: 0, minion: 1, demon: 1 }, // 7 players
  { townsfolk: 5, outsider: 1, minion: 1, demon: 1 }, // 8 players
  { townsfolk: 5, outsider: 2, minion: 1, demon: 1 }, // 9 players
  { townsfolk: 7, outsider: 0, minion: 2, demon: 1 }, // 10 players
  { townsfolk: 7, outsider: 1, minion: 2, demon: 1 }, // 11 players
  { townsfolk: 7, outsider: 2, minion: 2, demon: 1 }, // 12 players
  { townsfolk: 9, outsider: 0, minion: 3, demon: 1 }, // 13 players
  { townsfolk: 9, outsider: 1, minion: 3, demon: 1 }, // 14 players
  { townsfolk: 9, outsider: 2, minion: 3, demon: 1 }  // 15 players
];

/**
 * 获取角色组合配置
 * @param {number} playerCount 玩家数量
 * @returns {object} 角色组合 {townsfolk, outsider, minion, demon}
 */
export function getRoleComposition(playerCount) {
  if (playerCount < 5 || playerCount > 15) {
    throw new Error(`Invalid player count: ${playerCount}. Must be between 5 and 15.`);
  }
  return RoleCompositionTable[playerCount];
}

/**
 * 特殊角色处理规则
 */
export const SpecialRoleRules = {
  // 酒鬼：认为自己是村民，但实际是外来者
  drunk: {
    type: 'disguise',
    actualTeam: Team.OUTSIDER,
    perceivedTeam: Team.TOWNSFOLK,
    handler: (player, availableTownsfolk) => {
      // 酒鬼会被告知一个村民角色
      const randomTownsfolk = availableTownsfolk[
        Math.floor(Math.random() * availableTownsfolk.length)
      ];
      return {
        actualRole: 'drunk',
        perceivedRole: randomTownsfolk.id
      };
    }
  },

  // 男爵：增加2个外来者
  baron: {
    type: 'setup_modifier',
    modifyComposition: (composition) => {
      return {
        ...composition,
        townsfolk: composition.townsfolk - 2,
        outsider: composition.outsider + 2
      };
    }
  },

  // 教父：增加或减少1个外来者
  godfather: {
    type: 'setup_modifier',
    modifyComposition: (composition) => {
      const change = Math.random() < 0.5 ? -1 : 1;
      return {
        ...composition,
        townsfolk: composition.townsfolk - change,
        outsider: composition.outsider + change
      };
    }
  },

  // 疯子：认为自己是恶魔
  lunatic: {
    type: 'disguise',
    actualTeam: Team.OUTSIDER,
    perceivedTeam: Team.DEMON,
    handler: (player, availableDemons) => {
      const randomDemon = availableDemons[
        Math.floor(Math.random() * availableDemons.length)
      ];
      return {
        actualRole: 'lunatic',
        perceivedRole: randomDemon.id
      };
    }
  },

  // 间谍：可能被检测为好人
  spy: {
    type: 'detection_modifier',
    canRegisterAsGood: true
  },

  // 隐士：可能被检测为恶人
  recluse: {
    type: 'detection_modifier',
    canRegisterAsEvil: true
  }
};

/**
 * 检查角色是否有特殊规则
 * @param {string} roleId 角色ID
 * @returns {boolean}
 */
export function hasSpecialRule(roleId) {
  return roleId in SpecialRoleRules;
}

/**
 * 获取角色的特殊规则
 * @param {string} roleId 角色ID
 * @returns {object|null}
 */
export function getSpecialRule(roleId) {
  return SpecialRoleRules[roleId] || null;
}
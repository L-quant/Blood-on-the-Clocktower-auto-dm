/**
 * 游戏核心数据类型定义
 */

// 游戏阶段枚举
export const GamePhase = {
  SETUP: 'setup',
  FIRST_NIGHT: 'first_night',
  DAY: 'day',
  NIGHT: 'night',
  ENDED: 'ended'
};

// 角色团队枚举
export const Team = {
  TOWNSFOLK: 'townsfolk',
  OUTSIDER: 'outsider',
  MINION: 'minion',
  DEMON: 'demon',
  TRAVELER: 'traveler',
  FABLED: 'fabled'
};

// 能力时机枚举
export const AbilityTiming = {
  SETUP: 'setup',
  FIRST_NIGHT: 'first_night',
  OTHER_NIGHT: 'other_night',
  DAY: 'day',
  PASSIVE: 'passive'
};

// 能力类型枚举
export const AbilityType = {
  INFORMATION: 'information',
  PROTECTION: 'protection',
  KILL: 'kill',
  MANIPULATION: 'manipulation',
  DETECTION: 'detection',
  VOTING: 'voting'
};

// 游戏状态接口
export class GameState {
  constructor() {
    this.gameId = '';
    this.phase = GamePhase.SETUP;
    this.day = 0;
    this.players = [];
    this.deadPlayers = [];
    this.nominations = [];
    this.votes = [];
    this.nightActions = [];
    this.gameConfiguration = null;
    this.timestamp = Date.now();
  }
}

// 玩家状态接口
export class Player {
  constructor(id, name) {
    this.id = id;
    this.name = name;
    this.role = null;
    this.isAlive = true;
    this.isEvil = false;
    this.abilities = [];
    this.status = {};
    this.votes = 0;
    this.ghostVoteUsed = false;
    this.position = -1;
  }
}

// 角色定义接口
export class Role {
  constructor(data) {
    this.id = data.id;
    this.name = data.name;
    this.team = data.team;
    this.ability = data.ability;
    this.firstNight = data.firstNight || 0;
    this.otherNight = data.otherNight || 0;
    this.firstNightReminder = data.firstNightReminder || '';
    this.otherNightReminder = data.otherNightReminder || '';
    this.reminders = data.reminders || [];
    this.setup = data.setup || false;
  }

  get isEvil() {
    return this.team === Team.MINION || this.team === Team.DEMON;
  }

  get isGood() {
    return this.team === Team.TOWNSFOLK || this.team === Team.OUTSIDER;
  }
}

// 游戏配置接口
export class GameConfiguration {
  constructor() {
    this.scriptType = 'trouble-brewing';
    this.playerCount = 0;
    this.automationLevel = 'full';
    this.aiDifficulty = 'medium';
    this.timeSettings = {
      discussionTime: 300000, // 5分钟
      nominationTime: 60000,  // 1分钟
      votingTime: 30000,      // 30秒
      nightActionTimeout: 60000 // 1分钟
    };
    this.ruleVariants = [];
    this.debugMode = false;
  }
}

// 夜间行动接口
export class NightAction {
  constructor(playerId, roleId, actionType, target = null) {
    this.playerId = playerId;
    this.roleId = roleId;
    this.actionType = actionType;
    this.target = target;
    this.timestamp = Date.now();
    this.processed = false;
    this.result = null;
  }
}

// 投票接口
export class Vote {
  constructor(playerId, nominee, vote) {
    this.playerId = playerId;
    this.nominee = nominee;
    this.vote = vote; // true/false
    this.timestamp = Date.now();
  }
}

// 提名接口
export class Nomination {
  constructor(nominator, nominee) {
    this.nominator = nominator;
    this.nominee = nominee;
    this.timestamp = Date.now();
    this.votes = [];
    this.result = null;
  }
}
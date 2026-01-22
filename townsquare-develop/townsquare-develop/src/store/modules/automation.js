/**
 * 自动化说书人系统的Vuex模块
 */

import { SystemStatus, AutomationLevel } from '@/automation/types/AutomationTypes';
import { GamePhase } from '@/automation/types/GameTypes';

const state = () => ({
  // 系统状态
  systemStatus: SystemStatus.IDLE,
  automationLevel: AutomationLevel.FULL_AUTO,
  isAutomationEnabled: false,
  
  // 游戏自动化状态
  currentPhase: GamePhase.SETUP,
  isProcessingNightActions: false,
  isProcessingVoting: false,
  
  // AI决策状态
  aiDecisions: [],
  pendingDecisions: [],
  
  // 错误和日志
  errors: [],
  logs: [],
  
  // 配置
  configuration: {
    aiDifficulty: 'medium',
    debugMode: false,
    timeSettings: {
      discussionTime: 300000,
      nominationTime: 60000,
      votingTime: 30000,
      nightActionTimeout: 60000
    }
  }
});

const getters = {
  isSystemRunning: (state) => state.systemStatus === SystemStatus.RUNNING,
  isSystemIdle: (state) => state.systemStatus === SystemStatus.IDLE,
  hasErrors: (state) => state.errors.length > 0,
  latestError: (state) => state.errors[state.errors.length - 1] || null,
  pendingDecisionCount: (state) => state.pendingDecisions.length,
  isNightPhase: (state) => state.currentPhase === GamePhase.NIGHT || state.currentPhase === GamePhase.FIRST_NIGHT,
  isDayPhase: (state) => state.currentPhase === GamePhase.DAY
};

const mutations = {
  SET_SYSTEM_STATUS(state, status) {
    state.systemStatus = status;
  },
  
  SET_AUTOMATION_ENABLED(state, enabled) {
    state.isAutomationEnabled = enabled;
  },
  
  SET_AUTOMATION_LEVEL(state, level) {
    state.automationLevel = level;
  },
  
  SET_CURRENT_PHASE(state, phase) {
    state.currentPhase = phase;
  },
  
  SET_PROCESSING_NIGHT_ACTIONS(state, processing) {
    state.isProcessingNightActions = processing;
  },
  
  SET_PROCESSING_VOTING(state, processing) {
    state.isProcessingVoting = processing;
  },
  
  ADD_AI_DECISION(state, decision) {
    state.aiDecisions.push({
      ...decision,
      timestamp: Date.now()
    });
  },
  
  ADD_PENDING_DECISION(state, decision) {
    state.pendingDecisions.push(decision);
  },
  
  REMOVE_PENDING_DECISION(state, decisionId) {
    state.pendingDecisions = state.pendingDecisions.filter(d => d.id !== decisionId);
  },
  
  ADD_ERROR(state, error) {
    state.errors.push({
      ...error,
      timestamp: Date.now()
    });
  },
  
  CLEAR_ERRORS(state) {
    state.errors = [];
  },
  
  ADD_LOG(state, log) {
    state.logs.push({
      ...log,
      timestamp: Date.now()
    });
    
    // 保持日志数量在合理范围内
    if (state.logs.length > 1000) {
      state.logs = state.logs.slice(-500);
    }
  },
  
  UPDATE_CONFIGURATION(state, config) {
    state.configuration = {
      ...state.configuration,
      ...config
    };
  }
};

const actions = {
  async initializeAutomation({ commit, dispatch }, config) {
    try {
      commit('SET_SYSTEM_STATUS', SystemStatus.INITIALIZING);
      commit('UPDATE_CONFIGURATION', config);
      
      // 初始化各个子系统
      await dispatch('initializeSubsystems');
      
      commit('SET_SYSTEM_STATUS', SystemStatus.IDLE);
      commit('ADD_LOG', { level: 'info', message: 'Automation system initialized successfully' });
    } catch (error) {
      commit('SET_SYSTEM_STATUS', SystemStatus.ERROR);
      commit('ADD_ERROR', { type: 'initialization', message: error.message });
      throw error;
    }
  },
  
  async startAutomation({ commit, state }) {
    if (state.systemStatus !== SystemStatus.IDLE) {
      throw new Error('System must be idle to start automation');
    }
    
    commit('SET_SYSTEM_STATUS', SystemStatus.RUNNING);
    commit('SET_AUTOMATION_ENABLED', true);
    commit('ADD_LOG', { level: 'info', message: 'Automation started' });
  },
  
  async stopAutomation({ commit }) {
    commit('SET_SYSTEM_STATUS', SystemStatus.IDLE);
    commit('SET_AUTOMATION_ENABLED', false);
    commit('ADD_LOG', { level: 'info', message: 'Automation stopped' });
  },
  
  async pauseAutomation({ commit }) {
    commit('SET_SYSTEM_STATUS', SystemStatus.PAUSED);
    commit('ADD_LOG', { level: 'info', message: 'Automation paused' });
  },
  
  async resumeAutomation({ commit }) {
    commit('SET_SYSTEM_STATUS', SystemStatus.RUNNING);
    commit('ADD_LOG', { level: 'info', message: 'Automation resumed' });
  },
  
  async initializeSubsystems({ commit }) {
    // 这里将在后续任务中实现具体的子系统初始化
    commit('ADD_LOG', { level: 'debug', message: 'Initializing subsystems...' });
  },
  
  async handleError({ commit }, error) {
    commit('ADD_ERROR', error);
    commit('ADD_LOG', { level: 'error', message: `Error: ${error.message}` });
  }
};

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions
};
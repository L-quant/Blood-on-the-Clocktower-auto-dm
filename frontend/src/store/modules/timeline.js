// Vuex 模块：游戏事件时间线（阶段变化、死亡、投票）
//
// [OUT] store/index.js（模块注册）
// [POS] 事件记录，按天/类型展示游戏历史

const state = () => ({
  events: [],
  filters: ['all'] // 'all' | 'phase_change' | 'death' | 'nomination' | 'vote_result' | 'ability' | 'system'
});

const mutations = {
  addEvent(state, event) {
    state.events.push({
      id: event.id || Math.random().toString(36).substr(2, 10),
      type: event.type || 'system',
      timestamp: event.timestamp || Date.now(),
      dayCount: event.dayCount || 0,
      data: event.data || {},
      isPrivate: event.isPrivate || false
    });
  },
  setFilters(state, filters) {
    state.filters = filters;
  },
  clear(state) {
    state.events = [];
  },
  reset(state) {
    state.events = [];
    state.filters = ['all'];
  }
};

const getters = {
  filtered: state => {
    if (state.filters.includes('all')) {
      return state.events;
    }
    return state.events.filter(e => state.filters.includes(e.type));
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};

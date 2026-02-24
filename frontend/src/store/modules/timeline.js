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
  },
  byDay: state => dayCount => state.events.filter(e => e.dayCount === dayCount),
  latest: state => (count = 10) => state.events.slice(-count)
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};

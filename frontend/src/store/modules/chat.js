export default {
    namespaced: true,
    state: {
      messages: [],
      // Used to trigger socket events
      lastSentMessage: null 
    },
    mutations: {
      addMessage(state, message) {
        state.messages.push(message);
      },
      // Trigger mutation for socket.js to pick up
      triggerSend(state, payload) {
        state.lastSentMessage = payload;
      }
    },
    actions: {
      send({ commit }, { type, message }) {
        commit('triggerSend', { type, message });
      }
    },
    getters: {
      messages: state => state.messages
    }
  };
  
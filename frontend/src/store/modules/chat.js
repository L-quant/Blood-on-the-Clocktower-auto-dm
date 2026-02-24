const state = () => ({
  channels: {
    public: { messages: [], unread: 0 },
    evil: { messages: [], unread: 0 },
    assistant: { messages: [], unread: 0 },
    whispers: {}
    // whispers: { seatIndex -> { messages: [], unread: 0 } }
  },
  activeChannel: 'public',
  activeWhisperTarget: null, // seatIndex
  assistantLoading: false
});

const createMessage = (data) => ({
  id: data.id || Math.random().toString(36).substr(2, 10),
  seatIndex: data.seatIndex != null ? data.seatIndex : -1, // -1 for system/AI
  text: data.text || '',
  timestamp: data.timestamp || Date.now(),
  isSystem: data.isSystem || false,
  isMe: data.isMe || false
});

const mutations = {
  addPublicMessage(state, data) {
    state.channels.public.messages.push(createMessage(data));
    if (state.activeChannel !== 'public') {
      state.channels.public.unread++;
    }
  },
  addEvilMessage(state, data) {
    state.channels.evil.messages.push(createMessage(data));
    if (state.activeChannel !== 'evil') {
      state.channels.evil.unread++;
    }
  },
  addAssistantMessage(state, data) {
    state.channels.assistant.messages.push(createMessage(data));
    if (state.activeChannel !== 'assistant') {
      state.channels.assistant.unread++;
    }
  },
  addWhisperMessage(state, { targetSeat, data }) {
    if (!state.channels.whispers[targetSeat]) {
      state.channels.whispers = {
        ...state.channels.whispers,
        [targetSeat]: { messages: [], unread: 0 }
      };
    }
    state.channels.whispers[targetSeat].messages.push(createMessage(data));
    if (state.activeChannel !== 'whisper' || state.activeWhisperTarget !== targetSeat) {
      state.channels.whispers[targetSeat].unread++;
    }
  },
  setActiveChannel(state, channel) {
    state.activeChannel = channel;
    // Clear unread for the newly active channel
    if (channel === 'whisper' && state.activeWhisperTarget != null) {
      const whisper = state.channels.whispers[state.activeWhisperTarget];
      if (whisper) whisper.unread = 0;
    } else if (state.channels[channel]) {
      state.channels[channel].unread = 0;
    }
  },
  setActiveWhisperTarget(state, seatIndex) {
    state.activeWhisperTarget = seatIndex;
    if (state.channels.whispers[seatIndex]) {
      state.channels.whispers[seatIndex].unread = 0;
    }
  },
  setAssistantLoading(state, loading) {
    state.assistantLoading = loading;
  },
  clearChannel(state, channel) {
    if (state.channels[channel]) {
      state.channels[channel].messages = [];
      state.channels[channel].unread = 0;
    }
  },
  reset(state) {
    state.channels = {
      public: { messages: [], unread: 0 },
      evil: { messages: [], unread: 0 },
      assistant: { messages: [], unread: 0 },
      whispers: {}
    };
    state.activeChannel = 'public';
    state.activeWhisperTarget = null;
    state.assistantLoading = false;
  }
};

const getters = {
  activeMessages: state => {
    if (state.activeChannel === 'whisper' && state.activeWhisperTarget != null) {
      const whisper = state.channels.whispers[state.activeWhisperTarget];
      return whisper ? whisper.messages : [];
    }
    const channel = state.channels[state.activeChannel];
    return channel ? channel.messages : [];
  },
  totalUnread: state => {
    let count = state.channels.public.unread + state.channels.evil.unread + state.channels.assistant.unread;
    Object.values(state.channels.whispers).forEach(w => {
      count += w.unread;
    });
    return count;
  },
  hasEvilChannel: (state, getters, rootState) => {
    const myRole = rootState.players.myRole;
    return myRole && (myRole.team === 'minion' || myRole.team === 'demon');
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};

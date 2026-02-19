<template>
  <div class="chat-panel" :class="{ minimized: isMinimized }">
    <div class="chat-header" @click="toggleMinimize">
      <span class="chat-title">{{ $t('chat.title') }}</span>
      <span v-if="unreadCount > 0" class="unread-badge">{{ unreadCount }}</span>
      <font-awesome-icon :icon="isMinimized ? 'chevron-up' : 'chevron-down'" />
    </div>
    <div v-show="!isMinimized" class="chat-body">
      <div class="chat-messages" ref="messages">
        <div
          v-for="(msg, i) in messages"
          :key="i"
          class="chat-message"
          :class="{
            'system': msg.from === 'system' || msg.from === 'auto-dm',
            'self': msg.from === 'self',
            'autodm': msg.from === 'auto-dm'
          }"
        >
          <span class="msg-sender">{{ msg.displayName }}</span>
          <span class="msg-text">{{ msg.text }}</span>
        </div>
        <div v-if="messages.length === 0" class="chat-empty">
          {{ $t('chat.empty') }}
        </div>
      </div>
      <div class="chat-input">
        <input
          v-model="inputText"
          @keyup.enter="sendMessage"
          :placeholder="$t('chat.placeholder')"
          :disabled="!connected"
        />
        <button @click="sendMessage" :disabled="!inputText.trim() || !connected">
          <font-awesome-icon icon="paper-plane" />
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "ChatPanel",
  data() {
    return {
      isMinimized: true,
      inputText: "",
      messages: [],
      unreadCount: 0
    };
  },
  computed: {
    ...mapState(["session"]),
    connected() {
      return this.session && this.session.sessionId;
    }
  },
  methods: {
    toggleMinimize() {
      this.isMinimized = !this.isMinimized;
      if (!this.isMinimized) {
        this.unreadCount = 0;
        this.$nextTick(() => this.scrollToBottom());
      }
    },
    sendMessage() {
      const text = this.inputText.trim();
      if (!text) return;

      this.messages.push({
        from: "self",
        displayName: this.$t("chat.you"),
        text: text,
        timestamp: Date.now()
      });

      // Dispatch chat command via store
      this.$store.commit("session/sendChat", text);
      this.inputText = "";
      this.$nextTick(() => this.scrollToBottom());
    },
    addMessage(msg) {
      this.messages.push(msg);
      if (this.isMinimized) {
        this.unreadCount++;
      }
      this.$nextTick(() => this.scrollToBottom());
    },
    scrollToBottom() {
      const el = this.$refs.messages;
      if (el) {
        el.scrollTop = el.scrollHeight;
      }
    }
  },
  mounted() {
    // Listen for incoming chat events from the store
    this.$store.subscribe((mutation) => {
      if (mutation.type === "session/addChatMessage") {
        this.addMessage(mutation.payload);
      }
    });
  }
};
</script>

<style scoped lang="scss">
.chat-panel {
  position: fixed;
  bottom: 0;
  right: 20px;
  width: 320px;
  max-height: 400px;
  background: rgba(0, 0, 0, 0.85);
  border-radius: 8px 8px 0 0;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  font-family: inherit;
  color: #eee;
  box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.5);

  &.minimized {
    max-height: 36px;
  }
}

.chat-header {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  background: rgba(20, 20, 20, 0.95);
  border-radius: 8px 8px 0 0;
  user-select: none;

  .chat-title {
    flex: 1;
    font-weight: bold;
    font-size: 14px;
  }

  .unread-badge {
    background: #e74c3c;
    color: white;
    border-radius: 50%;
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    margin-right: 8px;
  }
}

.chat-body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  max-height: 300px;
  min-height: 100px;
}

.chat-message {
  margin-bottom: 6px;
  font-size: 13px;
  line-height: 1.4;

  .msg-sender {
    font-weight: bold;
    margin-right: 4px;
  }

  &.system .msg-sender {
    color: #f39c12;
  }

  &.autodm .msg-sender {
    color: #9b59b6;
  }

  &.self .msg-sender {
    color: #3498db;
  }
}

.chat-empty {
  text-align: center;
  color: #888;
  padding: 20px;
  font-size: 13px;
}

.chat-input {
  display: flex;
  padding: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);

  input {
    flex: 1;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 4px;
    color: #eee;
    padding: 6px 10px;
    font-size: 13px;
    outline: none;

    &:focus {
      border-color: rgba(255, 255, 255, 0.4);
    }

    &::placeholder {
      color: #777;
    }
  }

  button {
    margin-left: 6px;
    background: #3498db;
    border: none;
    border-radius: 4px;
    color: white;
    padding: 6px 10px;
    cursor: pointer;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    &:hover:not(:disabled) {
      background: #2980b9;
    }
  }
}

@media (max-width: 600px) {
  .chat-panel {
    width: calc(100% - 20px);
    right: 10px;
  }
}
</style>

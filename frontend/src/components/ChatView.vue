<template>
  <div class="chat-view">
    <!-- Channel tabs -->
    <div class="chat-view__tabs">
      <button
        class="chat-view__tab"
        :class="{ active: activeChannel === 'public' }"
        @click="switchChannel('public')"
      >
        {{ $t('chat.public') }}
        <span class="chat-view__badge" v-if="channels.public.unread">{{ channels.public.unread }}</span>
      </button>
      <button
        class="chat-view__tab"
        :class="{ active: activeChannel === 'whisper' }"
        @click="switchChannel('whisper')"
      >
        {{ $t('chat.whisper') }}
      </button>
      <button
        v-if="hasEvilChannel"
        class="chat-view__tab evil"
        :class="{ active: activeChannel === 'evil' }"
        @click="switchChannel('evil')"
      >
        {{ $t('chat.evil') }}
        <span class="chat-view__badge" v-if="channels.evil.unread">{{ channels.evil.unread }}</span>
      </button>
      <button
        class="chat-view__tab assistant"
        :class="{ active: activeChannel === 'assistant' }"
        @click="switchChannel('assistant')"
      >
        {{ $t('chat.assistant') }}
        <span class="chat-view__badge" v-if="channels.assistant.unread">{{ channels.assistant.unread }}</span>
      </button>
    </div>

    <!-- Whisper target selector -->
    <div class="chat-view__whisper-select" v-if="activeChannel === 'whisper'">
      <select v-model="whisperTarget" @change="onWhisperTargetChange">
        <option :value="null" disabled>{{ $t('chat.selectTarget') }}</option>
        <option
          v-for="p in otherPlayers"
          :key="p.seatIndex"
          :value="p.seatIndex"
        >{{ $t('chat.whisperTo', { n: p.seatIndex }) }}</option>
      </select>
    </div>

    <!-- AI Assistant quick actions -->
    <div class="chat-view__quick-actions" v-if="activeChannel === 'assistant' && messages.length === 0">
      <p class="chat-view__assistant-hint">{{ $t('chat.assistantHint') }}</p>
      <button
        v-for="(label, key) in quickActions"
        :key="key"
        class="chat-view__quick-btn"
        @click="sendAssistantMessage(label)"
      >{{ label }}</button>
    </div>

    <!-- Message list -->
    <div class="chat-view__messages" ref="messageList">
      <div v-if="messages.length === 0" class="chat-view__empty">
        {{ $t('chat.empty') }}
      </div>
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="chat-view__message"
        :class="{
          'is-me': msg.isMe,
          'is-system': msg.isSystem
        }"
      >
        <span class="chat-view__msg-seat" v-if="!msg.isMe && !msg.isSystem">
          {{ msg.seatIndex > 0 ? $t('square.seat', { n: msg.seatIndex }) : 'AI' }}
        </span>
        <div class="chat-view__msg-bubble" :class="{ me: msg.isMe, system: msg.isSystem }">
          {{ msg.text }}
        </div>
        <span class="chat-view__msg-time">
          {{ formatTime(msg.timestamp) }}
        </span>
      </div>
    </div>

    <!-- Loading indicator for AI assistant -->
    <div class="chat-view__loading" v-if="assistantLoading">
      <div class="chat-view__typing-dots">
        <span></span><span></span><span></span>
      </div>
    </div>

    <!-- Input -->
    <div class="chat-view__input-area">
      <input
        class="chat-view__input"
        v-model="inputText"
        :placeholder="activeChannel === 'assistant' ? $t('chat.assistantPlaceholder') : $t('chat.placeholder')"
        @keydown.enter="sendMessage"
      />
      <button
        class="chat-view__send-btn"
        :class="{ disabled: !inputText.trim() }"
        @click="sendMessage"
      >
        <font-awesome-icon icon="vote-yea" />
      </button>
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";
import apiService from "../services/ApiService";

export default {
  name: "ChatView",
  data() {
    return {
      inputText: "",
      whisperTarget: null
    };
  },
  computed: {
    ...mapState("chat", ["channels", "activeChannel", "activeWhisperTarget", "assistantLoading"]),
    ...mapState(["roomId", "seatIndex"]),
    ...mapState("players", ["players", "myRole"]),
    ...mapGetters("chat", ["activeMessages", "hasEvilChannel"]),
    messages() {
      return this.activeMessages;
    },
    otherPlayers() {
      return this.players.filter(p => !p.isMe);
    },
    quickActions() {
      return {
        howToPlay: this.$t('chat.quickActions.howToPlay'),
        whoToTrust: this.$t('chat.quickActions.whoToTrust'),
        strategy: this.$t('chat.quickActions.strategy')
      };
    }
  },
  watch: {
    messages() {
      this.$nextTick(() => {
        this.scrollToBottom();
      });
    },
    activeWhisperTarget(val) {
      this.whisperTarget = val;
    }
  },
  methods: {
    switchChannel(channel) {
      this.$store.commit("chat/setActiveChannel", channel);
    },
    onWhisperTargetChange() {
      this.$store.commit("chat/setActiveWhisperTarget", this.whisperTarget);
    },
    sendMessage() {
      const text = this.inputText.trim();
      if (!text) return;
      this.inputText = "";

      if (this.activeChannel === 'assistant') {
        this.sendAssistantMessage(text);
      } else {
        this.$store.dispatch("sendChat", {
          text,
          channel: this.activeChannel
        });
      }
    },
    async sendAssistantMessage(text) {
      // Add user message locally
      this.$store.commit("chat/addAssistantMessage", {
        text,
        isMe: true,
        seatIndex: this.seatIndex
      });

      // Call AI assistant API
      this.$store.commit("chat/setAssistantLoading", true);
      try {
        const response = await apiService.askAssistant(
          this.roomId,
          text,
          {
            roleId: this.myRole ? this.myRole.roleId : '',
            roleName: this.myRole ? this.myRole.roleName : '',
            team: this.myRole ? this.myRole.team : ''
          }
        );
        this.$store.commit("chat/addAssistantMessage", {
          text: response.answer || response.message || this.$t('errors.noResponse'),
          isSystem: true,
          seatIndex: -1
        });
      } catch (e) {
        this.$store.commit("chat/addAssistantMessage", {
          text: this.$t('errors.assistantUnavailable'),
          isSystem: true,
          seatIndex: -1
        });
      } finally {
        this.$store.commit("chat/setAssistantLoading", false);
      }
    },
    scrollToBottom() {
      const el = this.$refs.messageList;
      if (el) {
        el.scrollTop = el.scrollHeight;
      }
    },
    formatTime(ts) {
      const d = new Date(ts);
      return d.getHours().toString().padStart(2, '0') + ':' +
             d.getMinutes().toString().padStart(2, '0');
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.chat-view {
  height: 100%;
  display: flex;
  flex-direction: column;

  &__tabs {
    display: flex;
    gap: 2px;
    padding: 8px 12px;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    flex-shrink: 0;
  }

  &__tab {
    padding: 6px 12px;
    border: none;
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.08);
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.7rem;
    cursor: pointer;
    transition: all 200ms;
    white-space: nowrap;
    display: flex;
    align-items: center;
    gap: 4px;

    &.active {
      background: rgba(255, 255, 255, 0.2);
      color: white;
    }

    &.evil.active {
      background: rgba($demon, 0.3);
      color: $demon;
    }

    &.assistant.active {
      background: rgba($fabled, 0.2);
      color: $fabled;
    }
  }

  &__badge {
    min-width: 16px;
    height: 16px;
    border-radius: 8px;
    background: $demon;
    color: white;
    font-size: 0.55rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0 4px;
  }

  &__whisper-select {
    padding: 0 12px 8px;
    flex-shrink: 0;

    select {
      width: 100%;
      background: rgba(255, 255, 255, 0.08);
      color: white;
      border: 1px solid rgba(255, 255, 255, 0.15);
      border-radius: 8px;
      padding: 8px 12px;
      font-size: 0.8rem;
    }
  }

  &__quick-actions {
    padding: 16px;
    text-align: center;
  }

  &__assistant-hint {
    font-size: 0.75rem;
    opacity: 0.4;
    margin-bottom: 12px;
  }

  &__quick-btn {
    display: block;
    width: 100%;
    padding: 10px 16px;
    margin-bottom: 8px;
    border: 1px solid rgba($fabled, 0.3);
    border-radius: 8px;
    background: rgba($fabled, 0.05);
    color: $fabled;
    font-size: 0.8rem;
    cursor: pointer;
    text-align: left;
    transition: all 200ms;

    &:active {
      transform: scale(0.98);
      background: rgba($fabled, 0.1);
    }
  }

  &__messages {
    flex: 1;
    overflow-y: auto;
    padding: 8px 12px;
    -webkit-overflow-scrolling: touch;
  }

  &__empty {
    text-align: center;
    opacity: 0.3;
    padding: 40px 0;
    font-size: 0.85rem;
  }

  &__message {
    margin-bottom: 8px;
    display: flex;
    flex-direction: column;

    &.is-me {
      align-items: flex-end;
    }

    &.is-system {
      align-items: center;
    }
  }

  &__msg-seat {
    font-size: 0.6rem;
    opacity: 0.4;
    margin-bottom: 2px;
    padding-left: 8px;
  }

  &__msg-bubble {
    max-width: 80%;
    padding: 8px 14px;
    border-radius: 16px;
    font-size: 0.85rem;
    line-height: 1.4;
    word-break: break-word;
    background: rgba(255, 255, 255, 0.1);

    &.me {
      background: rgba($townsfolk, 0.25);
      border-bottom-right-radius: 4px;
    }

    &.system {
      background: rgba(255, 255, 255, 0.05);
      font-size: 0.75rem;
      opacity: 0.6;
      font-style: italic;
    }
  }

  &__msg-time {
    font-size: 0.5rem;
    opacity: 0.3;
    margin-top: 2px;
    padding: 0 8px;
  }

  &__loading {
    padding: 8px 12px;
    flex-shrink: 0;
  }

  &__typing-dots {
    display: flex;
    gap: 4px;
    padding: 8px 14px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 16px;
    width: fit-content;

    span {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: rgba(255, 255, 255, 0.3);
      animation: typing-bounce 1.4s ease-in-out infinite;

      &:nth-child(2) { animation-delay: 0.2s; }
      &:nth-child(3) { animation-delay: 0.4s; }
    }
  }

  &__input-area {
    display: flex;
    gap: 8px;
    padding: 8px 12px;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    flex-shrink: 0;
  }

  &__input {
    flex: 1;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 20px;
    padding: 8px 16px;
    color: white;
    font-size: 0.85rem;
    outline: none;
    font-family: inherit;

    &:focus {
      border-color: rgba(255, 255, 255, 0.3);
    }

    &::placeholder {
      color: rgba(255, 255, 255, 0.2);
    }
  }

  &__send-btn {
    width: 38px;
    height: 38px;
    border-radius: 50%;
    border: none;
    background: $townsfolk;
    color: white;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 200ms;
    flex-shrink: 0;

    &.disabled {
      opacity: 0.3;
      cursor: default;
    }

    &:active:not(.disabled) {
      transform: scale(0.9);
    }
  }
}

@keyframes typing-bounce {
  0%, 60%, 100% { transform: translateY(0); }
  30% { transform: translateY(-4px); }
}
</style>

<template>
  <div class="chat-container" :class="{ open: isOpen }">
    <div class="chat-header" @click="isOpen = !isOpen">
      Chat
      <span v-if="unread" class="unread-badge">{{ unread }}</span>
    </div>
    <div class="chat-body" v-show="isOpen">
      <div class="messages" ref="messages">
        <div v-for="(msg, i) in messages" :key="i" class="message" :class="msg.type">
          <span class="sender">{{ msg.sender }}:</span>
          <span class="text">{{ msg.message }}</span>
        </div>
      </div>
      <div class="input-area">
        <input 
          v-model="input" 
          @keydown.stop
          @keyup.stop="handleKeyup"
          placeholder="Type a message..." 
        />
      </div>
    </div>
  </div>
</template>

<script>
import { mapGetters } from 'vuex';

export default {
  data() {
    return {
      isOpen: true,
      input: '',
      unread: 0
    };
  },
  computed: {
    ...mapGetters('chat', ['messages'])
  },
  watch: {
    messages() {
      if (!this.isOpen) {
        this.unread++;
      }
      this.$nextTick(() => {
        const el = this.$refs.messages;
        if (el) el.scrollTop = el.scrollHeight;
      });
    },
    isOpen(val) {
      if (val) this.unread = 0;
    }
  },
  mounted() {
    // Auto scroll to bottom
    this.$nextTick(() => {
      const el = this.$refs.messages;
      if (el) el.scrollTop = el.scrollHeight;
    });
  },
  methods: {
    handleKeyup(e) {
      if (e.key === 'Enter') {
        this.send();
      }
    },
    send() {
      if (!this.input.trim()) return;
      const text = this.input.trim();
      
      // Basic support for whisper via /w
      // E.g. /w PlayerName message
      let type = 'public';
      let message = text;
      
      this.$store.dispatch('chat/send', { type, message });
      this.input = '';
    }
  }
};
</script>

<style scoped lang="scss">
@import "../vars.scss";

.chat-container {
  position: fixed;
  bottom: 0;
  right: 20px;
  width: 350px;
  background: rgba(20, 20, 30, 0.95);
  border: 1px solid #555;
  border-bottom: none;
  border-radius: 8px 8px 0 0;
  color: #eee;
  display: flex;
  flex-direction: column;
  z-index: 9999;
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  box-shadow: 0 0 10px rgba(0,0,0,0.5);
  transition: height 0.3s ease;
  height: 40px; /* Closed State */
  
  &.open {
    height: 400px;
  }
}

.chat-header {
  padding: 10px 15px;
  background: #2a2a3a;
  cursor: pointer;
  font-weight: bold;
  border-radius: 8px 8px 0 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #444;
  
  &:hover {
    background: #353545;
  }
}

.chat-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: calc(100% - 40px);
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  font-size: 14px;
  line-height: 1.4;
  
  /* Scrollbar styling */
  &::-webkit-scrollbar {
    width: 8px;
  }
  &::-webkit-scrollbar-track {
    background: #1a1a1a;
  }
  &::-webkit-scrollbar-thumb {
    background: #444;
    border-radius: 4px;
  }
}

.message {
  margin-bottom: 8px;
  word-wrap: break-word;
  
  &.whisper {
    color: #46d5ff; /* Outsider blue-ish */
    font-style: italic;
  }
  &.public {
    color: #ddd;
  }
}

.sender {
  font-weight: bold;
  margin-right: 5px;
  color: #c9c9c9;
}

.input-area {
  padding: 10px;
  border-top: 1px solid #444;
  background: #2a2a3a;
  
  input {
    width: 100%;
    padding: 8px 10px;
    border-radius: 20px;
    border: 1px solid #555;
    background: #1a1a25;
    color: #fff;
    outline: none;
    box-sizing: border-box; /* Important for width 100% */
    
    &:focus {
      border-color: #777;
    }
    
    &::placeholder {
      color: #777;
    }
  }
}

.unread-badge {
  background: #ce0100; /* Demon red */
  color: white;
  border-radius: 10px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: bold;
}
</style>

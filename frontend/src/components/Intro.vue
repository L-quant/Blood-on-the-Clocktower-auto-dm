<template>
  <div class="intro">
    <img src="static/apple-icon.png" alt="" class="logo" />
    
    <div class="main-content">
      <h2>Blood on the Clocktower</h2>
      <p class="subtitle">AI-Powered Storyteller</p>
      
      <div v-if="!isJoining" class="action-buttons">
        <button class="btn-primary" @click="createGame">
          <font-awesome-icon icon="plus-circle" /> Create New Room
        </button>
        
        <button class="btn-secondary" @click="isJoining = true">
          <font-awesome-icon icon="sign-in-alt" /> Join Room
        </button>
      </div>
      
      <div v-else class="join-form">
        <input 
          v-model="roomIdInput" 
          placeholder="Enter Room ID" 
          class="room-input"
          @keyup.enter="joinGame"
        />
        <div class="form-actions">
          <button class="btn-primary" @click="joinGame" :disabled="!roomIdInput">Join</button>
          <button class="btn-text" @click="isJoining = false">Cancel</button>
        </div>
      </div>

      <div class="error-msg" v-if="error">{{ error }}</div>
    </div>

    <div class="footer">
      Unofficial Virtual Grimoire.<br>
      Not affiliated with The Pandemonium Institute.
    </div>
  </div>
</template>

<script>
import { mapMutations } from "vuex";
import backendAPI from "@/api/backend";

export default {
  data() {
    return {
      isJoining: false,
      roomIdInput: "",
      error: ""
    };
  },
  methods: {
    ...mapMutations(["toggleMenu"]),
    
    async createGame() {
      try {
        this.error = "";
        const data = await backendAPI.createRoom();
        console.log("Room created:", data);
        
        // This sets the session ID handling in Vuex
        // We set spectator=false (Host)
        this.$store.commit("session/setSpectator", false);
        this.$store.commit("session/setSessionId", data.room_id);
        
        // Ensure we connect to the lobby socket
        this.$store.dispatch("session/connect", data.room_id);
        
      } catch (err) {
        console.error(err);
        this.error = "Failed to create room. Is backend running?";
      }
    },

    async joinGame() {
      if (!this.roomIdInput) return;
      try {
        this.error = "";
        // Verify room exists first
        await backendAPI.getRoomState(this.roomIdInput);
        
        // We set spectator=true (Player)
        this.$store.commit("session/setSpectator", true);
        this.$store.commit("session/setSessionId", this.roomIdInput);
        
        this.$store.dispatch("session/connect", this.roomIdInput);
        
      } catch (err) {
        console.error(err);
        this.error = "Room not found or cannot join.";
      }
    }
  }
};
</script>

<style scoped lang="scss">
.intro {
  text-align: center;
  width: 400px;
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 30px;
  background: rgba(20, 20, 30, 0.95);
  border: 2px solid #d4af37;
  border-radius: 15px;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #fff;
  box-shadow: 0 10px 30px rgba(0,0,0,0.7);

  .logo {
    width: 100px;
    height: 100px;
    margin-bottom: 20px;
    border-radius: 50%;
    border: 3px solid #d4af37;
    box-shadow: 0 0 15px rgba(212, 175, 55, 0.3);
  }
  
  .main-content {
    width: 100%;
    
    h2 {
      margin: 0;
      font-family: 'Cinzel', serif;
      color: #d4af37;
      font-size: 1.8em;
    }
    
    .subtitle {
      color: #aaa;
      margin-top: 5px;
      margin-bottom: 30px;
      font-style: italic;
    }
  }
  
  .action-buttons {
    display: flex;
    flex-direction: column;
    gap: 15px;
  }
  
  .join-form {
    display: flex;
    flex-direction: column;
    gap: 15px;
    
    .room-input {
      padding: 12px;
      border-radius: 5px;
      border: 1px solid #555;
      background: #222;
      color: white;
      text-align: center;
      font-size: 1.2em;
      
      &:focus {
        border-color: #d4af37;
        outline: none;
      }
    }
    
    .form-actions {
      display: flex;
      gap: 10px;
      justify-content: center;
    }
  }
  
  button {
    cursor: pointer;
    font-size: 1.1em;
    padding: 12px 20px;
    border-radius: 5px;
    transition: all 0.2s;
    font-weight: bold;
    
    &.btn-primary {
      background: #d4af37;
      color: #111;
      border: none;
      
      &:hover { background: #eac14d; transform: translateY(-2px); }
      &:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }
    }
    
    &.btn-secondary {
      background: transparent;
      color: #d4af37;
      border: 2px solid #d4af37;
      
      &:hover { background: rgba(212, 175, 55, 0.1); }
    }
    
    &.btn-text {
      background: transparent;
      color: #aaa;
      border: none;
      font-size: 0.9em;
      
      &:hover { color: white; }
    }
  }
  
  .error-msg {
    color: #ff5555;
    margin-top: 15px;
    font-size: 0.9em;
  }
  
  .footer {
    margin-top: 30px;
    font-size: 0.7em;
    color: #666;
    line-height: 1.5;
  }
}
</style>

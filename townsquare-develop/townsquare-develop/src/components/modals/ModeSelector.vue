<template>
  <Modal v-if="modals.modeSelector" @close="close">
    <div class="mode-selector">
      <h2>选择游戏模式</h2>
      <p class="description">
        请选择游戏模式。游戏开始后将无法更改。
      </p>
      
      <div class="mode-options">
        <!-- 说书人模式 -->
        <div 
          class="mode-option"
          :class="{ selected: selectedMode === 'storyteller' }"
          @click="selectMode('storyteller')"
        >
          <div class="mode-icon">
            <font-awesome-icon icon="book-open" />
          </div>
          <h3>说书人模式</h3>
          <p class="mode-description">
            传统模式，需要一名说书人主持游戏。说书人可以看到所有玩家的角色，并管理游戏流程。
          </p>
          <ul class="mode-features">
            <li><font-awesome-icon icon="check" /> 说书人可见所有角色</li>
            <li><font-awesome-icon icon="check" /> 手动管理游戏流程</li>
            <li><font-awesome-icon icon="check" /> 适合线下游戏</li>
          </ul>
        </div>
        
        <!-- 无说书人模式 -->
        <div 
          class="mode-option"
          :class="{ selected: selectedMode === 'player-only' }"
          @click="selectMode('player-only')"
        >
          <div class="mode-icon">
            <font-awesome-icon icon="users" />
          </div>
          <h3>无说书人模式</h3>
          <p class="mode-description">
            自动化模式，无需说书人。每个玩家只能看到自己的角色，系统自动管理游戏流程。
          </p>
          <ul class="mode-features">
            <li><font-awesome-icon icon="check" /> 角色隐私保护</li>
            <li><font-awesome-icon icon="check" /> 自动化游戏流程</li>
            <li><font-awesome-icon icon="check" /> 适合线上游戏</li>
            <li><font-awesome-icon icon="exclamation-triangle" class="warning" /> 需要自动化系统</li>
          </ul>
        </div>
      </div>
      
      <div class="mode-actions">
        <button 
          class="button confirm"
          :disabled="!selectedMode"
          @click="confirmMode"
        >
          <font-awesome-icon icon="check" />
          确认选择
        </button>
        <button 
          class="button cancel"
          @click="close"
        >
          <font-awesome-icon icon="times" />
          取消
        </button>
      </div>
      
      <div class="mode-warning" v-if="gameStarted">
        <font-awesome-icon icon="exclamation-triangle" />
        游戏已开始，无法更改模式
      </div>
    </div>
  </Modal>
</template>

<script>
import Modal from "./Modal";
import { mapState, mapGetters, mapMutations } from "vuex";

export default {
  name: "ModeSelector",
  components: {
    Modal
  },
  data() {
    return {
      selectedMode: null
    };
  },
  computed: {
    ...mapState(["modals"]),
    ...mapState("privacy", { currentMode: state => state.gameMode }),
    ...mapState("automation", { automationState: state => state }),
    gameStarted() {
      return this.automationState?.isAutomationEnabled || false;
    }
  },
  mounted() {
    // 初始化为当前模式
    this.selectedMode = this.currentMode || 'storyteller';
  },
  methods: {
    ...mapMutations(["toggleModal"]),
    selectMode(mode) {
      if (this.gameStarted) {
        return;
      }
      this.selectedMode = mode;
    },
    confirmMode() {
      if (!this.selectedMode || this.gameStarted) {
        return;
      }
      
      console.log('[ModeSelector] Confirming mode:', this.selectedMode);
      
      this.$store.dispatch('privacy/switchGameMode', this.selectedMode)
        .then(result => {
          console.log('[ModeSelector] Switch result:', result);
          
          if (result && result.success) {
            this.$emit('mode-selected', this.selectedMode);
            this.close();
          } else if (result && result.error) {
            alert(`切换模式失败: ${result.error}`);
          } else {
            // 如果没有明确的success标志，假设成功
            this.$emit('mode-selected', this.selectedMode);
            this.close();
          }
        })
        .catch(error => {
          console.error('[ModeSelector] Error switching mode:', error);
          alert('切换模式时发生错误');
        });
    },
    close() {
      this.toggleModal("modeSelector");
    }
  }
};
</script>

<style scoped lang="scss">
.mode-selector {
  color: white;
  max-width: 800px;
  padding: 20px;
  
  h2 {
    text-align: center;
    margin-bottom: 10px;
    font-size: 2em;
  }
  
  .description {
    text-align: center;
    margin-bottom: 30px;
    opacity: 0.8;
  }
  
  .mode-options {
    display: flex;
    gap: 20px;
    margin-bottom: 30px;
    
    .mode-option {
      flex: 1;
      background: rgba(255, 255, 255, 0.1);
      border: 3px solid rgba(255, 255, 255, 0.2);
      border-radius: 10px;
      padding: 20px;
      cursor: pointer;
      transition: all 0.3s ease;
      
      &:hover {
        background: rgba(255, 255, 255, 0.15);
        border-color: rgba(255, 255, 255, 0.4);
        transform: translateY(-5px);
      }
      
      &.selected {
        background: rgba(100, 150, 255, 0.3);
        border-color: rgba(100, 150, 255, 0.8);
        box-shadow: 0 0 20px rgba(100, 150, 255, 0.5);
      }
      
      .mode-icon {
        text-align: center;
        font-size: 3em;
        margin-bottom: 15px;
        opacity: 0.8;
      }
      
      h3 {
        text-align: center;
        margin-bottom: 15px;
        font-size: 1.5em;
      }
      
      .mode-description {
        text-align: center;
        margin-bottom: 20px;
        opacity: 0.9;
        font-size: 0.9em;
        line-height: 1.5;
      }
      
      .mode-features {
        list-style: none;
        padding: 0;
        margin: 0;
        
        li {
          padding: 8px 0;
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          
          &:first-child {
            border-top: none;
          }
          
          svg {
            margin-right: 10px;
            color: #4CAF50;
            
            &.warning {
              color: #FFC107;
            }
          }
        }
      }
    }
  }
  
  .mode-actions {
    display: flex;
    justify-content: center;
    gap: 20px;
    
    .button {
      padding: 12px 30px;
      font-size: 1.1em;
      border: none;
      border-radius: 5px;
      cursor: pointer;
      transition: all 0.3s ease;
      display: flex;
      align-items: center;
      gap: 10px;
      
      &.confirm {
        background: #4CAF50;
        color: white;
        
        &:hover:not(:disabled) {
          background: #45a049;
          transform: scale(1.05);
        }
        
        &:disabled {
          background: #666;
          cursor: not-allowed;
          opacity: 0.5;
        }
      }
      
      &.cancel {
        background: #f44336;
        color: white;
        
        &:hover {
          background: #da190b;
          transform: scale(1.05);
        }
      }
    }
  }
  
  .mode-warning {
    margin-top: 20px;
    padding: 15px;
    background: rgba(255, 193, 7, 0.2);
    border: 2px solid rgba(255, 193, 7, 0.5);
    border-radius: 5px;
    text-align: center;
    
    svg {
      margin-right: 10px;
      color: #FFC107;
    }
  }
}
</style>

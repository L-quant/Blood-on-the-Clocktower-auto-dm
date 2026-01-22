<template>
  <div class="automation-panel" v-if="isVisible">
    <div class="panel-header">
      <h3>自动化说书人</h3>
      <button class="close-btn" @click="close">×</button>
    </div>
    
    <div class="panel-content">
      <!-- 系统状态 -->
      <div class="status-section">
        <div class="status-indicator" :class="statusClass">
          <span class="status-dot"></span>
          <span class="status-text">{{ statusText }}</span>
        </div>
        <div class="phase-indicator" v-if="isAutomationEnabled">
          当前阶段: {{ phaseText }}
        </div>
      </div>

      <!-- 快速操作 -->
      <div class="quick-actions" v-if="!isAutomationEnabled">
        <h4>快速操作</h4>
        <button 
          class="button townsfolk" 
          @click="autoAssignRoles"
          :disabled="!canAssignRoles"
        >
          <font-awesome-icon icon="theater-masks" />
          自动分配角色
        </button>
        <small v-if="!canAssignRoles" class="hint">
          需要至少5名玩家并选择游戏版本
        </small>
      </div>

      <!-- 控制按钮 -->
      <div class="control-section">
        <template v-if="!isAutomationEnabled">
          <button 
            class="button townsfolk" 
            @click="initializeAutomation"
            :disabled="isInitializing"
          >
            <font-awesome-icon icon="play" />
            初始化自动化系统
          </button>
        </template>
        
        <template v-else>
          <div class="button-group">
            <button 
              class="button" 
              @click="startAutomation"
              :disabled="isSystemRunning || isProcessing"
              v-if="isSystemIdle"
            >
              <font-awesome-icon icon="play" />
              启动
            </button>
            
            <button 
              class="button" 
              @click="pauseAutomation"
              :disabled="!isSystemRunning"
              v-if="isSystemRunning"
            >
              <font-awesome-icon icon="pause" />
              暂停
            </button>
            
            <button 
              class="button" 
              @click="resumeAutomation"
              :disabled="systemStatus !== 'paused'"
              v-if="systemStatus === 'paused'"
            >
              <font-awesome-icon icon="play" />
              恢复
            </button>
            
            <button 
              class="button demon" 
              @click="stopAutomation"
              :disabled="!isAutomationEnabled"
            >
              <font-awesome-icon icon="stop" />
              停止
            </button>
          </div>
        </template>
      </div>

      <!-- AI决策建议 -->
      <div class="ai-section" v-if="isAutomationEnabled && pendingDecisions.length > 0">
        <h4>AI决策建议</h4>
        <div class="decision-list">
          <div 
            v-for="decision in pendingDecisions" 
            :key="decision.id"
            class="decision-item"
          >
            <div class="decision-header">
              <span class="decision-player">{{ decision.playerName }}</span>
              <span class="decision-type">{{ decision.type }}</span>
            </div>
            <div class="decision-suggestions">
              <button 
                v-for="(suggestion, index) in decision.suggestions.slice(0, 3)"
                :key="index"
                class="suggestion-btn"
                @click="applySuggestion(decision.id, suggestion)"
              >
                {{ suggestion.action }}
                <small>({{ suggestion.confidence }}%)</small>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 配置选项 -->
      <div class="config-section">
        <h4>配置</h4>
        <div class="config-item">
          <label>AI难度:</label>
          <select v-model="aiDifficulty" @change="updateConfiguration">
            <option value="easy">简单</option>
            <option value="medium">中等</option>
            <option value="hard">困难</option>
          </select>
        </div>
        <div class="config-item">
          <label>自动化级别:</label>
          <select v-model="automationLevel" @change="updateConfiguration">
            <option value="manual">手动</option>
            <option value="semi_auto">半自动</option>
            <option value="full_auto">全自动</option>
          </select>
        </div>
        <div class="config-item">
          <label>
            <input type="checkbox" v-model="debugMode" @change="updateConfiguration" />
            调试模式
          </label>
        </div>
      </div>

      <!-- 错误显示 -->
      <div class="error-section" v-if="hasErrors">
        <h4>错误</h4>
        <div class="error-message">
          {{ latestError.message }}
        </div>
        <button class="button" @click="clearErrors">清除错误</button>
      </div>

      <!-- 日志 -->
      <div class="log-section" v-if="debugMode">
        <h4>日志 (最近10条)</h4>
        <div class="log-list">
          <div 
            v-for="(log, index) in recentLogs" 
            :key="index"
            class="log-item"
            :class="log.level"
          >
            <span class="log-time">{{ formatTime(log.timestamp) }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters, mapActions, mapMutations } from 'vuex';
import AutomatedStorytellerSystem from '@/automation/core/AutomatedStorytellerSystem';

export default {
  name: 'AutomationPanel',
  
  data() {
    return {
      isVisible: false,
      isInitializing: false,
      automationSystem: null,
      aiDifficulty: 'medium',
      automationLevel: 'full_auto',
      debugMode: false
    };
  },
  
  computed: {
    ...mapState('automation', [
      'systemStatus',
      'isAutomationEnabled',
      'currentPhase',
      'isProcessingNightActions',
      'isProcessingVoting',
      'pendingDecisions',
      'errors',
      'logs',
      'configuration'
    ]),
    
    ...mapGetters('automation', [
      'isSystemRunning',
      'isSystemIdle',
      'hasErrors',
      'latestError',
      'pendingDecisionCount',
      'isNightPhase',
      'isDayPhase'
    ]),
    
    canAssignRoles() {
      const players = this.$store.state.players.players;
      const edition = this.$store.state.edition;
      return players.length >= 5 && edition && edition.id;
    },
    
    statusClass() {
      return {
        'status-idle': this.systemStatus === 'idle',
        'status-running': this.systemStatus === 'running',
        'status-paused': this.systemStatus === 'paused',
        'status-error': this.systemStatus === 'error'
      };
    },
    
    statusText() {
      const statusMap = {
        'idle': '空闲',
        'initializing': '初始化中',
        'running': '运行中',
        'paused': '已暂停',
        'error': '错误'
      };
      return statusMap[this.systemStatus] || '未知';
    },
    
    phaseText() {
      const phaseMap = {
        'setup': '准备阶段',
        'first_night': '第一夜',
        'day': '白天',
        'night': '夜晚',
        'ended': '游戏结束'
      };
      return phaseMap[this.currentPhase] || '未知';
    },
    
    isProcessing() {
      return this.isProcessingNightActions || this.isProcessingVoting;
    },
    
    recentLogs() {
      return this.logs.slice(-10).reverse();
    }
  },
  
  methods: {
    ...mapActions('automation', [
      'initializeAutomation',
      'startAutomation',
      'stopAutomation',
      'pauseAutomation',
      'resumeAutomation',
      'handleError'
    ]),
    
    ...mapMutations('automation', [
      'CLEAR_ERRORS',
      'UPDATE_CONFIGURATION',
      'REMOVE_PENDING_DECISION'
    ]),
    
    show() {
      this.isVisible = true;
    },
    
    close() {
      this.isVisible = false;
    },
    
    toggle() {
      this.isVisible = !this.isVisible;
    },
    
    async initializeAutomation() {
      try {
        this.isInitializing = true;
        
        // 创建自动化系统实例
        this.automationSystem = new AutomatedStorytellerSystem();
        
        // 初始化配置
        const config = {
          scriptType: this.$store.state.edition.id || 'trouble-brewing',
          playerCount: this.$store.state.players.players.length,
          aiDifficulty: this.aiDifficulty,
          automationLevel: this.automationLevel,
          debugMode: this.debugMode
        };
        
        // 初始化系统
        await this.automationSystem.initialize(config);
        
        // 更新Vuex状态
        await this.$store.dispatch('automation/initializeAutomation', config);
        
        this.$store.commit('automation/ADD_LOG', {
          level: 'info',
          message: '自动化系统初始化成功'
        });
      } catch (error) {
        console.error('Failed to initialize automation:', error);
        await this.handleError({
          type: 'initialization',
          message: error.message
        });
      } finally {
        this.isInitializing = false;
      }
    },
    
    async startAutomation() {
      try {
        if (!this.automationSystem) {
          throw new Error('Automation system not initialized');
        }
        
        // 获取玩家列表
        const players = this.$store.state.players.players.map(p => ({
          id: p.id,
          name: p.name
        }));
        
        if (players.length < 5) {
          throw new Error('至少需要5名玩家才能开始游戏');
        }
        
        // 启动自动化游戏
        await this.automationSystem.startAutomatedGame(players);
        await this.$store.dispatch('automation/startAutomation');
        
      } catch (error) {
        console.error('Failed to start automation:', error);
        await this.handleError({
          type: 'start',
          message: error.message
        });
      }
    },
    
    async stopAutomation() {
      try {
        if (this.automationSystem) {
          this.automationSystem.stopGame();
        }
        await this.$store.dispatch('automation/stopAutomation');
      } catch (error) {
        console.error('Failed to stop automation:', error);
      }
    },
    
    async pauseAutomation() {
      try {
        if (this.automationSystem) {
          this.automationSystem.pauseAutomation();
        }
        await this.$store.dispatch('automation/pauseAutomation');
      } catch (error) {
        console.error('Failed to pause automation:', error);
      }
    },
    
    async resumeAutomation() {
      try {
        if (this.automationSystem) {
          this.automationSystem.resumeAutomation();
        }
        await this.$store.dispatch('automation/resumeAutomation');
      } catch (error) {
        console.error('Failed to resume automation:', error);
      }
    },
    
    updateConfiguration() {
      this.UPDATE_CONFIGURATION({
        aiDifficulty: this.aiDifficulty,
        automationLevel: this.automationLevel,
        debugMode: this.debugMode
      });
    },
    
    applySuggestion(decisionId, suggestion) {
      // 应用AI建议
      console.log('Applying suggestion:', decisionId, suggestion);
      this.REMOVE_PENDING_DECISION(decisionId);
      
      // 这里可以添加实际应用建议的逻辑
    },
    
    clearErrors() {
      this.CLEAR_ERRORS();
    },
    
    formatTime(timestamp) {
      const date = new Date(timestamp);
      return date.toLocaleTimeString();
    },
    
    async autoAssignRoles() {
      try {
        // 导入角色分配器
        const RoleAssigner = (await import('@/automation/core/RoleAssigner')).default;
        const roleAssigner = new RoleAssigner();
        
        // 获取玩家列表
        const players = this.$store.state.players.players.map(p => ({
          id: p.id,
          name: p.name
        }));
        
        // 获取当前版本
        const scriptType = this.$store.state.edition.id || 'trouble-brewing';
        
        // 分配角色
        const assignments = await roleAssigner.assignRolesToPlayers(players, scriptType);
        
        // 应用到游戏状态
        assignments.forEach(assignment => {
          const player = this.$store.state.players.players.find(p => p.id === assignment.playerId);
          if (player) {
            // 设置角色
            this.$store.commit('players/update', {
              player: player,
              property: 'role',
              value: assignment.role
            });
          }
        });
        
        // 显示成功消息
        const summary = roleAssigner.getAssignmentSummary(assignments);
        alert(`角色分配成功！\n\n` +
          `村民: ${summary.townsfolk}\n` +
          `外来者: ${summary.outsider}\n` +
          `爪牙: ${summary.minion}\n` +
          `恶魔: ${summary.demon}\n\n` +
          `善良阵营: ${summary.goodPlayers}\n` +
          `邪恶阵营: ${summary.evilPlayers}`
        );
        
      } catch (error) {
        console.error('Failed to auto-assign roles:', error);
        alert(`角色分配失败: ${error.message}`);
      }
    }
  }
};
</script>

<style scoped lang="scss">
@import "../vars.scss";

.automation-panel {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 500px;
  max-height: 80vh;
  background: rgba(0, 0, 0, 0.95);
  border: 3px solid #000;
  border-radius: 10px;
  box-shadow: 0 0 30px rgba(0, 0, 0, 0.8);
  z-index: 100;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.panel-header {
  background: linear-gradient(to right, $townsfolk 0%, $demon 100%);
  padding: 10px 15px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  h3 {
    margin: 0;
    font-family: PiratesBay, sans-serif;
    font-size: 1.5em;
    color: white;
  }
  
  .close-btn {
    background: none;
    border: none;
    color: white;
    font-size: 2em;
    cursor: pointer;
    line-height: 1;
    padding: 0;
    width: 30px;
    height: 30px;
    
    &:hover {
      color: red;
    }
  }
}

.panel-content {
  padding: 15px;
  overflow-y: auto;
  flex: 1;
}

.status-section {
  margin-bottom: 20px;
  
  .status-indicator {
    display: flex;
    align-items: center;
    padding: 10px;
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.1);
    
    .status-dot {
      width: 12px;
      height: 12px;
      border-radius: 50%;
      margin-right: 10px;
      animation: pulse 2s infinite;
    }
    
    &.status-idle .status-dot {
      background: gray;
    }
    
    &.status-running .status-dot {
      background: green;
    }
    
    &.status-paused .status-dot {
      background: orange;
    }
    
    &.status-error .status-dot {
      background: red;
    }
  }
  
  .phase-indicator {
    margin-top: 10px;
    padding: 5px 10px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 3px;
    font-size: 0.9em;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.quick-actions {
  margin-bottom: 20px;
  
  h4 {
    margin-bottom: 10px;
    color: $townsfolk;
  }
  
  .button {
    width: 100%;
    padding: 10px;
    font-size: 1em;
    margin-bottom: 5px;
  }
  
  .hint {
    display: block;
    color: gray;
    font-size: 0.85em;
    margin-top: 5px;
  }
}

.control-section {
  margin-bottom: 20px;
  
  .button {
    width: 100%;
    padding: 10px;
    font-size: 1em;
  }
  
  .button-group {
    display: flex;
    gap: 10px;
    
    .button {
      flex: 1;
    }
  }
}

.ai-section {
  margin-bottom: 20px;
  
  h4 {
    margin-bottom: 10px;
    color: $townsfolk;
  }
  
  .decision-list {
    max-height: 200px;
    overflow-y: auto;
  }
  
  .decision-item {
    background: rgba(255, 255, 255, 0.05);
    padding: 10px;
    margin-bottom: 10px;
    border-radius: 5px;
    
    .decision-header {
      display: flex;
      justify-content: space-between;
      margin-bottom: 10px;
      
      .decision-player {
        font-weight: bold;
      }
      
      .decision-type {
        color: $demon;
        font-size: 0.9em;
      }
    }
    
    .decision-suggestions {
      display: flex;
      gap: 5px;
      flex-wrap: wrap;
    }
    
    .suggestion-btn {
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid rgba(255, 255, 255, 0.2);
      color: white;
      padding: 5px 10px;
      border-radius: 3px;
      cursor: pointer;
      font-size: 0.85em;
      
      &:hover {
        background: rgba(255, 255, 255, 0.2);
      }
      
      small {
        color: $townsfolk;
        margin-left: 5px;
      }
    }
  }
}

.config-section {
  margin-bottom: 20px;
  
  h4 {
    margin-bottom: 10px;
  }
  
  .config-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
    
    label {
      flex: 1;
    }
    
    select {
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid rgba(255, 255, 255, 0.2);
      color: white;
      padding: 5px 10px;
      border-radius: 3px;
      cursor: pointer;
      
      option {
        background: #000;
      }
    }
    
    input[type="checkbox"] {
      margin-right: 10px;
    }
  }
}

.error-section {
  margin-bottom: 20px;
  
  h4 {
    color: red;
    margin-bottom: 10px;
  }
  
  .error-message {
    background: rgba(255, 0, 0, 0.2);
    border: 1px solid red;
    padding: 10px;
    border-radius: 5px;
    margin-bottom: 10px;
    color: #ffcccc;
  }
}

.log-section {
  h4 {
    margin-bottom: 10px;
  }
  
  .log-list {
    max-height: 150px;
    overflow-y: auto;
    background: rgba(0, 0, 0, 0.5);
    border-radius: 5px;
    padding: 5px;
  }
  
  .log-item {
    font-size: 0.8em;
    padding: 3px 5px;
    margin-bottom: 2px;
    border-left: 3px solid gray;
    
    &.info {
      border-left-color: $townsfolk;
    }
    
    &.warn {
      border-left-color: orange;
    }
    
    &.error {
      border-left-color: red;
    }
    
    .log-time {
      color: gray;
      margin-right: 10px;
    }
  }
}
</style>

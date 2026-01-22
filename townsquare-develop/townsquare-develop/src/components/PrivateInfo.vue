<template>
  <div class="private-info" v-if="isPlayerOnlyMode && hasPrivateInfo">
    <div class="private-info-header" @click="isExpanded = !isExpanded">
      <font-awesome-icon :icon="isExpanded ? 'chevron-down' : 'chevron-right'" />
      <span>我的私密信息</span>
    </div>
    <transition name="slide">
      <div class="private-info-content" v-if="isExpanded">
        <!-- 我的角色 -->
        <div class="my-role" v-if="myRole && myRole.id">
          <h4>我的角色</h4>
          <div class="role-card">
            <span class="role-name">{{ myRole.name }}</span>
            <span class="role-ability">{{ myRole.ability }}</span>
          </div>
        </div>

        <!-- 夜间信息 -->
        <div class="night-info" v-if="nightInformation && nightInformation.length > 0">
          <h4>夜间信息</h4>
          <ul>
            <li v-for="(info, index) in nightInformation" :key="index">
              <span class="info-time">第{{ info.night }}夜</span>
              <span class="info-message">{{ info.message }}</span>
            </li>
          </ul>
        </div>

        <!-- 能力结果 -->
        <div class="ability-results" v-if="abilityResults && abilityResults.length > 0">
          <h4>能力结果</h4>
          <ul>
            <li v-for="(result, index) in abilityResults" :key="index">
              <span class="result-message">{{ result.message }}</span>
            </li>
          </ul>
        </div>

        <!-- 无私密信息提示 -->
        <div class="no-info" v-if="!hasAnyInfo">
          <p>暂无私密信息</p>
        </div>
      </div>
    </transition>
  </div>
</template>

<script>
import { mapState, mapGetters } from 'vuex';

export default {
  name: 'PrivateInfo',
  data() {
    return {
      isExpanded: false
    };
  },
  computed: {
    ...mapState('privacy', ['gameMode']),
    ...mapGetters('privacy', ['isPlayerOnlyMode']),
    
    // 从privacy store获取私密信息
    myRole() {
      return this.$store.state.privacy.myRole || {};
    },
    
    nightInformation() {
      return this.$store.state.privacy.nightInformation || [];
    },
    
    abilityResults() {
      return this.$store.state.privacy.abilityResults || [];
    },
    
    hasAnyInfo() {
      return (this.myRole && this.myRole.id) ||
             (this.nightInformation && this.nightInformation.length > 0) ||
             (this.abilityResults && this.abilityResults.length > 0);
    },
    
    hasPrivateInfo() {
      return this.isPlayerOnlyMode && this.hasAnyInfo;
    }
  }
};
</script>

<style scoped lang="scss">
.private-info {
  position: fixed;
  bottom: 20px;
  right: 20px;
  background: rgba(0, 0, 0, 0.8);
  border: 3px solid #000;
  border-radius: 10px;
  padding: 10px;
  max-width: 400px;
  z-index: 100;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.5);
  color: white;
}

.private-info-header {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: bold;
  font-size: 16px;
  padding: 5px;
  
  &:hover {
    color: #ff6b6b;
  }
}

.private-info-content {
  margin-top: 10px;
  max-height: 400px;
  overflow-y: auto;
}

.slide-enter-active, .slide-leave-active {
  transition: all 0.3s ease;
  max-height: 400px;
}

.slide-enter, .slide-leave-to {
  max-height: 0;
  opacity: 0;
}

.my-role, .night-info, .ability-results {
  margin-bottom: 15px;
  
  h4 {
    margin: 0 0 10px 0;
    color: #ffd700;
    font-size: 14px;
  }
}

.role-card {
  background: rgba(255, 255, 255, 0.1);
  padding: 10px;
  border-radius: 5px;
  
  .role-name {
    display: block;
    font-weight: bold;
    font-size: 16px;
    margin-bottom: 5px;
  }
  
  .role-ability {
    display: block;
    font-size: 12px;
    font-style: italic;
    color: #ccc;
  }
}

.night-info ul, .ability-results ul {
  list-style: none;
  padding: 0;
  margin: 0;
  
  li {
    background: rgba(255, 255, 255, 0.1);
    padding: 8px;
    margin-bottom: 5px;
    border-radius: 5px;
    font-size: 12px;
    
    .info-time {
      display: inline-block;
      background: rgba(255, 215, 0, 0.3);
      padding: 2px 6px;
      border-radius: 3px;
      margin-right: 8px;
      font-weight: bold;
    }
    
    .info-message, .result-message {
      color: #fff;
    }
  }
}

.no-info {
  text-align: center;
  color: #999;
  font-style: italic;
  padding: 20px;
}
</style>

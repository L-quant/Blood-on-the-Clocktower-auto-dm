<template>
  <div class="game-screen" :class="{ 'is-wide': !isMobile }">
    <!-- Mobile: tab-switched views -->
    <template v-if="isMobile">
      <SquareView v-show="activeTab === 'square'" />
      <ChatView v-show="activeTab === 'chat'" />
      <TimelineView v-show="activeTab === 'timeline'" />
      <MeView v-show="activeTab === 'me'" @open-settings="$emit('open-settings')" />
    </template>

    <!-- Tablet/Desktop: multi-column layout -->
    <template v-else>
      <!-- Left sidebar (desktop only, via CSS) -->
      <div class="game-screen__sidebar">
        <MeView @open-settings="$emit('open-settings')" />
      </div>

      <!-- Center: Town Square -->
      <div class="game-screen__center">
        <SquareView />
      </div>

      <!-- Right panel: Chat / Timeline tabs -->
      <div class="game-screen__right">
        <div class="game-screen__right-tabs">
          <button
            class="game-screen__right-tab"
            :class="{ active: rightTab === 'chat' }"
            @click="rightTab = 'chat'"
          >{{ $t('nav.chat') }}</button>
          <button
            class="game-screen__right-tab"
            :class="{ active: rightTab === 'timeline' }"
            @click="rightTab = 'timeline'"
          >{{ $t('nav.timeline') }}</button>
        </div>
        <div class="game-screen__right-content">
          <ChatView v-show="rightTab === 'chat'" />
          <TimelineView v-show="rightTab === 'timeline'" />
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import { mapState } from "vuex";
import SquareView from "./SquareView";
import ChatView from "./ChatView";
import TimelineView from "./TimelineView";
import MeView from "./MeView";

export default {
  name: "GameScreen",
  components: { SquareView, ChatView, TimelineView, MeView },
  data() {
    return {
      rightTab: 'chat'
    };
  },
  computed: {
    ...mapState("ui", ["activeTab", "isMobile"])
  }
};
</script>

<style lang="scss" scoped>
.game-screen {
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>

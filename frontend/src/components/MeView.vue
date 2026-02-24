<template>
  <div class="me-view">
    <!-- Role card -->
    <div class="me-view__role-card" v-if="myRole">
      <div class="me-view__role-header" :class="'team-' + myRole.team">
        <img
          v-if="roleIcon"
          class="me-view__role-icon"
          :src="roleIcon"
          :alt="myRole.roleName"
        />
        <div class="me-view__role-info">
          <h3 class="me-view__role-name">{{ roleDisplayName }}</h3>
          <span class="me-view__role-team">{{ $t('teams.' + myRole.team) }}</span>
        </div>
      </div>
      <div class="me-view__ability">
        <span class="me-view__label">{{ $t('me.ability') }}</span>
        <p class="me-view__ability-text">{{ myRole.ability }}</p>
      </div>
    </div>

    <!-- No role yet -->
    <div class="me-view__no-role" v-else>
      <p>{{ $t('me.noRole') }}</p>
    </div>

    <!-- Demon bluffs -->
    <div class="me-view__bluffs" v-if="bluffs.some(b => b)">
      <span class="me-view__section-title">{{ $t('me.bluffs') }}</span>
      <div class="me-view__bluff-list">
        <span
          v-for="(bluff, i) in bluffs"
          :key="i"
          class="me-view__bluff"
        >{{ bluff || '—' }}</span>
      </div>
    </div>

    <!-- Notes -->
    <div class="me-view__notes">
      <span class="me-view__section-title">{{ $t('me.notes') }}</span>
      <textarea
        class="me-view__notes-input"
        v-model="notesText"
        :placeholder="$t('me.notesPlaceholder')"
        rows="4"
        @input="saveNotes"
      ></textarea>
    </div>

    <!-- Quick links -->
    <div class="me-view__links">
      <button class="me-view__link" @click="openSettings">
        <font-awesome-icon icon="cog" />
        {{ $t('me.settings') }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "MeView",
  data() {
    return {
      notesText: '',
      saveDebounce: null
    };
  },
  computed: {
    ...mapState("players", ["myRole", "bluffs"]),
    ...mapState("ui", ["notes"]),
    roleDisplayName() {
      if (!this.myRole || !this.myRole.roleId) return '';
      const key = 'roles.' + this.myRole.roleId;
      return this.$te(key) ? this.$t(key) : this.myRole.roleName;
    },
    roleIcon() {
      if (!this.myRole || !this.myRole.roleId) return '';
      try {
        return require(`../assets/icons/${this.myRole.roleId}.png`);
      } catch (e) {
        return '';
      }
    }
  },
  watch: {
    notes: {
      immediate: true,
      handler(val) {
        this.notesText = val || '';
      }
    }
  },
  methods: {
    saveNotes() {
      clearTimeout(this.saveDebounce);
      this.saveDebounce = setTimeout(() => {
        this.$store.commit("ui/setNotes", this.notesText);
      }, 500);
    },
    openSettings() {
      this.$emit("open-settings");
    }
  },
  beforeDestroy() {
    clearTimeout(this.saveDebounce);
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.me-view {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  -webkit-overflow-scrolling: touch;

  &__role-card {
    background: rgba(0, 0, 0, 0.4);
    border: 2px solid rgba(255, 255, 255, 0.15);
    border-radius: 12px;
    overflow: hidden;
    margin-bottom: 16px;
  }

  &__role-header {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;

    &.team-townsfolk { background: rgba($townsfolk, 0.1); border-bottom: 2px solid rgba($townsfolk, 0.3); }
    &.team-outsider { background: rgba($outsider, 0.1); border-bottom: 2px solid rgba($outsider, 0.3); }
    &.team-minion { background: rgba($minion, 0.1); border-bottom: 2px solid rgba($minion, 0.3); }
    &.team-demon { background: rgba($demon, 0.1); border-bottom: 2px solid rgba($demon, 0.3); }
  }

  &__role-icon {
    width: 64px;
    height: 64px;
    object-fit: contain;
  }

  &__role-info {
    flex: 1;
  }

  &__role-name {
    font-size: 1.2rem;
    text-align: left;
    margin-bottom: 4px;
  }

  &__role-team {
    font-size: 0.75rem;
    opacity: 0.6;
  }

  &__ability {
    padding: 12px 16px;
  }

  &__label {
    font-size: 0.7rem;
    opacity: 0.4;
    text-transform: uppercase;
    letter-spacing: 1px;
    display: block;
    margin-bottom: 6px;
  }

  &__ability-text {
    font-size: 0.85rem;
    line-height: 1.5;
    font-family: Papyrus, serif;
    margin: 0;
  }

  &__no-role {
    text-align: center;
    padding: 40px 0;
    opacity: 0.4;
    font-size: 0.9rem;
  }

  &__bluffs {
    margin-bottom: 16px;
  }

  &__section-title {
    font-size: 0.7rem;
    opacity: 0.4;
    text-transform: uppercase;
    letter-spacing: 1px;
    display: block;
    margin-bottom: 8px;
  }

  &__bluff-list {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  &__bluff {
    padding: 4px 12px;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    font-size: 0.8rem;
  }

  &__notes {
    margin-bottom: 16px;
  }

  &__notes-input {
    width: 100%;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 8px;
    padding: 12px;
    color: white;
    font-size: 0.85rem;
    resize: vertical;
    outline: none;
    font-family: inherit;
    min-height: 80px;

    &:focus {
      border-color: rgba(255, 255, 255, 0.3);
    }

    &::placeholder {
      color: rgba(255, 255, 255, 0.15);
    }
  }

  &__links {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__link {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    border: none;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.7);
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 200ms;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
    }

    svg {
      opacity: 0.5;
    }
  }
}
</style>

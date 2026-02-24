<template>
  <div class="role-annotator">
    <h4 class="role-annotator__title">{{ $t('annotation.selectRole') }}</h4>

    <!-- Team filter tabs -->
    <div class="role-annotator__tabs">
      <button
        v-for="team in teams"
        :key="team.id"
        class="role-annotator__tab"
        :class="{ active: activeTeam === team.id, [team.id]: true }"
        @click="activeTeam = team.id"
      >
        {{ $t('teams.' + team.id) }}
      </button>
    </div>

    <!-- Role grid -->
    <div class="role-annotator__grid">
      <button
        v-for="role in filteredRoles"
        :key="role.id"
        class="role-annotator__role"
        :class="{ selected: selectedRoleId === role.id }"
        @click="selectRole(role)"
      >
        <img
          class="role-annotator__role-icon"
          :src="getRoleIcon(role.id)"
          :alt="role.name"
        />
        <span class="role-annotator__role-name">{{ localizedRoleName(role) }}</span>
      </button>
    </div>

    <!-- Quick team annotation -->
    <div class="role-annotator__quick-teams">
      <span class="role-annotator__label">{{ $t('annotation.selectTeam') }}</span>
      <div class="role-annotator__team-buttons">
        <button
          class="role-annotator__team-btn good"
          :class="{ selected: selectedTeam === 'townsfolk' }"
          @click="selectTeam('townsfolk')"
        >{{ $t('annotation.good') }}</button>
        <button
          class="role-annotator__team-btn evil"
          :class="{ selected: selectedTeam === 'minion' || selectedTeam === 'demon' }"
          @click="selectTeam('minion')"
        >{{ $t('annotation.evil') }}</button>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "RoleAnnotator",
  props: {
    currentRoleId: { type: String, default: '' },
    currentTeam: { type: String, default: '' }
  },
  data() {
    return {
      activeTeam: 'townsfolk',
      selectedRoleId: this.currentRoleId || '',
      selectedTeam: this.currentTeam || ''
    };
  },
  computed: {
    ...mapState(["roles"]),
    teams() {
      return [
        { id: 'townsfolk' },
        { id: 'outsider' },
        { id: 'minion' },
        { id: 'demon' }
      ];
    },
    filteredRoles() {
      const result = [];
      if (this.roles && this.roles.forEach) {
        this.roles.forEach(role => {
          if (role.team === this.activeTeam) {
            result.push(role);
          }
        });
      }
      return result;
    }
  },
  methods: {
    getRoleIcon(roleId) {
      try {
        return require(`../assets/icons/${roleId}.png`);
      } catch (e) {
        return '';
      }
    },
    selectRole(role) {
      this.selectedRoleId = role.id;
      this.selectedTeam = role.team;
      this.$emit("select", {
        guessedRoleId: role.id,
        guessedTeam: role.team
      });
    },
    localizedRoleName(role) {
      const key = 'roles.' + role.id;
      return this.$te(key) ? this.$t(key) : role.name;
    },
    selectTeam(team) {
      this.selectedTeam = team;
      this.selectedRoleId = '';
      this.$emit("select", {
        guessedRoleId: '',
        guessedTeam: team
      });
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.role-annotator {
  &__title {
    font-size: 0.85rem;
    text-align: left;
    margin-bottom: 8px;
    opacity: 0.7;
  }

  &__tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 12px;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  &__tab {
    flex: 1;
    padding: 6px 8px;
    border: none;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.08);
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.65rem;
    cursor: pointer;
    transition: all 200ms;
    white-space: nowrap;

    &.active {
      color: white;
    }
    &.active.townsfolk { background: rgba($townsfolk, 0.3); }
    &.active.outsider { background: rgba($outsider, 0.3); }
    &.active.minion { background: rgba($minion, 0.3); }
    &.active.demon { background: rgba($demon, 0.3); }
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 6px;
    max-height: 200px;
    overflow-y: auto;
    margin-bottom: 12px;
    -webkit-overflow-scrolling: touch;
  }

  &__role {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 8px 4px;
    border: 2px solid transparent;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.05);
    color: white;
    cursor: pointer;
    transition: all 200ms;

    &.selected {
      border-color: $townsfolk;
      background: rgba($townsfolk, 0.1);
    }

    &:active {
      transform: scale(0.95);
    }
  }

  &__role-icon {
    width: 36px;
    height: 36px;
    object-fit: contain;
  }

  &__role-name {
    font-size: 0.5rem;
    text-align: center;
    line-height: 1.1;
    opacity: 0.7;
  }

  &__quick-teams {
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    padding-top: 8px;
  }

  &__label {
    font-size: 0.7rem;
    opacity: 0.5;
    display: block;
    margin-bottom: 6px;
  }

  &__team-buttons {
    display: flex;
    gap: 8px;
  }

  &__team-btn {
    flex: 1;
    padding: 8px;
    border: 2px solid rgba(255, 255, 255, 0.15);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.05);
    color: white;
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 200ms;

    &.good.selected {
      border-color: $townsfolk;
      background: rgba($townsfolk, 0.15);
    }
    &.evil.selected {
      border-color: $demon;
      background: rgba($demon, 0.15);
    }

    &:active {
      transform: scale(0.95);
    }
  }
}
</style>

<template>
  <div class="edition-picker">
    <label class="edition-picker__label">{{ $t('lobby.edition') }}</label>
    <div class="edition-picker__options">
      <button
        v-for="ed in editions"
        :key="ed.id"
        class="edition-picker__option"
        :class="{ active: currentId === ed.id }"
        @click="select(ed)"
      >
        <img
          v-if="ed.logo"
          class="edition-picker__logo"
          :src="ed.logo"
          :alt="ed.name"
        />
        <span class="edition-picker__name">{{ editionName(ed.id) }}</span>
      </button>
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "EditionPicker",
  computed: {
    ...mapState(["edition"]),
    ...mapGetters(["editionList"]),
    currentId() {
      return this.edition ? this.edition.id : '';
    },
    editions() {
      return this.editionList.map(e => ({
        id: e.id,
        name: e.name,
        logo: e.logo || null
      }));
    }
  },
  methods: {
    editionName(id) {
      const key = `editions.${id}`;
      const translated = this.$t(key);
      // Fallback to raw name if no translation found
      return translated === key ? (this.editionList.find(e => e.id === id) || {}).name || id : translated;
    },
    select(ed) {
      this.$store.commit("setEdition", ed.id);
      this.$emit("change", ed);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.edition-picker {
  &__label {
    display: block;
    text-align: center;
    font-size: 0.8rem;
    opacity: 0.6;
    margin-bottom: 8px;
  }

  &__options {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: center;
  }

  &__option {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 8px 12px;
    border: 2px solid rgba(255, 255, 255, 0.15);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.05);
    color: white;
    cursor: pointer;
    transition: all 200ms;
    min-width: 80px;

    &:hover {
      border-color: rgba(255, 255, 255, 0.3);
    }

    &.active {
      border-color: $townsfolk;
      background: rgba($townsfolk, 0.1);
    }

    &:active {
      transform: scale(0.95);
    }
  }

  &__logo {
    width: 40px;
    height: 40px;
    object-fit: contain;
  }

  &__name {
    font-size: 0.65rem;
    text-align: center;
    white-space: nowrap;
  }
}
</style>

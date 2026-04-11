<template>
  <div class="neko-empty-state" :class="{ 'is-compact': compact }">
    <img class="neko-empty-state__image" src="/neko-badge.svg" alt="" aria-hidden="true" />
    <div class="neko-empty-state__title">{{ title }}</div>
    <div class="neko-empty-state__description">{{ description }}</div>
  </div>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    default: '喵，这里暂时还是空的',
  },
  description: {
    type: String,
    default: '等你继续添加、选择或触发动作后，猫咪会把内容带到这里。',
  },
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>

<style scoped>
.neko-empty-state {
  min-height: 220px;
  padding: 22px 18px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  text-align: center;
}

.neko-empty-state.is-compact {
  min-height: 160px;
  padding: 16px 14px;
}

.neko-empty-state__image {
  width: 72px;
  height: 72px;
  filter: drop-shadow(0 12px 24px rgba(127, 89, 54, 0.14));
}

.neko-empty-state.is-compact .neko-empty-state__image {
  width: 58px;
  height: 58px;
}

.neko-empty-state__title {
  font-size: 16px;
  font-weight: 700;
  color: #5b3d2b;
}

.neko-empty-state__description {
  max-width: 360px;
  font-size: 13px;
  line-height: 1.7;
  color: #7a6150;
}

:global(html.dark) .neko-empty-state__image {
  filter: drop-shadow(0 12px 24px rgba(0, 0, 0, 0.22));
}

:global(html.dark) .neko-empty-state__title {
  color: #f0dcc8;
}

:global(html.dark) .neko-empty-state__description {
  color: #c9b39d;
}
</style>
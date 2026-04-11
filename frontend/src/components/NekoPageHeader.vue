<template>
  <div class="neko-page-header" :class="[`tone-${tone}`, `is-${resolvedTheme}`]">
    <button type="button" class="neko-page-header__back" @click="$emit('back')">
      <el-icon class="neko-page-header__back-icon"><ArrowLeft /></el-icon>
      <span>{{ backLabel }}</span>
    </button>
    <div class="neko-page-header__divider"></div>
    <div class="neko-page-header__title-wrap">
      <h1 class="neko-page-header__title">{{ title }}</h1>
      <span class="neko-page-header__tag">猫爪值班中</span>
    </div>
    <div class="neko-page-header__badge-wrap">
      <img class="neko-page-header__badge" src="/neko-badge.svg" alt="" aria-hidden="true" />
    </div>
    <p class="neko-page-header__desc">{{ description }}</p>
  </div>
</template>

<script setup>
import { ArrowLeft } from '@element-plus/icons-vue'
import { resolvedTheme } from '@/utils/theme'

defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    required: true,
  },
  backLabel: {
    type: String,
    default: '返回主页',
  },
  tone: {
    type: String,
    default: 'amber',
  },
})

defineEmits(['back'])
</script>

<style scoped>
.neko-page-header {
  display: grid;
  grid-template-columns: auto 1px auto auto minmax(220px, 1fr);
  align-items: center;
  gap: 16px;
  padding: 14px 18px;
  border-radius: 26px;
  border: 1px solid rgba(181, 136, 89, 0.16);
  background: linear-gradient(135deg, rgba(255, 248, 237, 0.96), rgba(255, 240, 219, 0.9));
  box-shadow: 0 14px 28px rgba(104, 76, 48, 0.08);
}

.neko-page-header__back {
  border: 0;
  background: transparent;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 999px;
  color: #705647;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.18s ease, color 0.18s ease, transform 0.18s ease;
}

.neko-page-header__back:hover {
  background: rgba(255, 255, 255, 0.55);
  transform: translateY(-1px);
}

.neko-page-header__back-icon {
  font-size: 14px;
}

.neko-page-header__divider {
  width: 1px;
  height: 38px;
  background: rgba(170, 135, 103, 0.2);
}

.neko-page-header__title-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.neko-page-header__title {
  margin: 0;
  font-size: 20px;
  line-height: 1;
  color: #5b3d2b;
  font-weight: 700;
  white-space: nowrap;
}

.neko-page-header__tag {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.56);
  color: #9b6a41;
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.neko-page-header__badge-wrap {
  width: 52px;
  height: 52px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.48);
}

.neko-page-header__badge {
  width: 38px;
  height: 38px;
}

.neko-page-header__desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #765b4a;
}

.tone-blue {
  background: linear-gradient(135deg, rgba(241, 248, 255, 0.96), rgba(231, 243, 255, 0.9));
  border-color: rgba(104, 151, 208, 0.16);
}

.tone-blue .neko-page-header__title {
  color: #45698f;
}

.tone-blue .neko-page-header__tag {
  color: #507eb0;
}

.tone-blue .neko-page-header__desc,
.tone-blue .neko-page-header__back {
  color: #5d7188;
}

.tone-green {
  background: linear-gradient(135deg, rgba(241, 250, 244, 0.96), rgba(230, 246, 236, 0.9));
  border-color: rgba(93, 168, 120, 0.16);
}

.tone-green .neko-page-header__title {
  color: #427558;
}

.tone-green .neko-page-header__tag {
  color: #4d8f62;
}

.tone-green .neko-page-header__desc,
.tone-green .neko-page-header__back {
  color: #597061;
}

.tone-orange {
  background: linear-gradient(135deg, rgba(255, 245, 235, 0.96), rgba(255, 236, 214, 0.9));
  border-color: rgba(223, 154, 73, 0.16);
}

.tone-orange .neko-page-header__title {
  color: #9c6020;
}

.tone-orange .neko-page-header__tag {
  color: #bb7a32;
}

.tone-orange .neko-page-header__desc,
.tone-orange .neko-page-header__back {
  color: #84624a;
}

.neko-page-header.is-dark {
  background: linear-gradient(135deg, rgba(36, 34, 35, 0.96), rgba(28, 31, 36, 0.92));
  border-color: rgba(241, 211, 176, 0.12);
  box-shadow: 0 16px 30px rgba(0, 0, 0, 0.18);
}

.neko-page-header.is-dark .neko-page-header__divider {
  background: rgba(255, 255, 255, 0.08);
}

.neko-page-header.is-dark .neko-page-header__badge-wrap,
.neko-page-header.is-dark .neko-page-header__tag,
.neko-page-header.is-dark .neko-page-header__back:hover {
  background: rgba(255, 255, 255, 0.06);
}

.neko-page-header.is-dark .neko-page-header__title {
  color: #f0d7be;
}

.neko-page-header.is-dark .neko-page-header__tag {
  color: #efc28b;
}

.neko-page-header.is-dark .neko-page-header__desc,
.neko-page-header.is-dark .neko-page-header__back {
  color: #ceb8a4;
}

.neko-page-header.is-dark.tone-blue {
  background: linear-gradient(135deg, rgba(26, 38, 53, 0.94), rgba(31, 40, 49, 0.9));
  border-color: rgba(108, 155, 213, 0.14);
}

.neko-page-header.is-dark.tone-green {
  background: linear-gradient(135deg, rgba(23, 39, 29, 0.94), rgba(28, 40, 33, 0.9));
  border-color: rgba(94, 173, 122, 0.12);
}

.neko-page-header.is-dark.tone-orange {
  background: linear-gradient(135deg, rgba(55, 38, 24, 0.94), rgba(43, 34, 28, 0.9));
  border-color: rgba(223, 154, 73, 0.12);
}

@media (max-width: 1100px) {
  .neko-page-header {
    grid-template-columns: auto 1px minmax(0, 1fr);
    grid-template-areas:
      'back divider title'
      'badge badge desc';
    align-items: start;
  }

  .neko-page-header__back {
    grid-area: back;
  }

  .neko-page-header__divider {
    grid-area: divider;
    height: 100%;
    min-height: 38px;
  }

  .neko-page-header__title-wrap {
    grid-area: title;
  }

  .neko-page-header__badge-wrap {
    grid-area: badge;
    margin-left: 6px;
  }

  .neko-page-header__desc {
    grid-area: desc;
    align-self: center;
  }
}

@media (max-width: 720px) {
  .neko-page-header {
    grid-template-columns: 1fr;
    grid-template-areas:
      'back'
      'title'
      'badge'
      'desc';
    gap: 10px;
    padding: 12px 14px;
  }

  .neko-page-header__divider {
    display: none;
  }

  .neko-page-header__title-wrap {
    flex-wrap: wrap;
  }

  .neko-page-header__title {
    font-size: 18px;
  }
}
</style>
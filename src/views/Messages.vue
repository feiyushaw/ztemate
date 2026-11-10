<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { snapshot as s, copyCode } from "../state";
import { otp, date } from "../format";
import Icon from "../components/Icon.vue";
const search = ref(""),
  selected = ref("");
const conversations = computed(() => {
  const groups = new Map<string, typeof s.value.messages>();
  for (const m of s.value.messages) {
    if (search.value && !`${m.sender} ${m.content}`.includes(search.value))
      continue;
    groups.set(m.sender, [...(groups.get(m.sender) || []), m]);
  }
  return [...groups.entries()].map(([sender, messages]) => ({
    sender,
    messages,
    unread: messages.some((m) => m.unread),
  }));
});
watch(
  conversations,
  (c) => {
    if (!c.some((v) => v.sender === selected.value))
      selected.value = c[0]?.sender || "";
  },
  { immediate: true },
);
const current = computed(() =>
  conversations.value.find((c) => c.sender === selected.value),
);
</script>
<template>
  <div class="sms-workspace card page-enter">
    <aside class="conversation-panel">
      <div class="search-box">
        <Icon name="Search" :size="16" /><input
          v-model="search"
          aria-label="搜索号码和短信"
          placeholder="搜索号码或短信"
          type="search"
        />
      </div>
      <div class="conversation-label">
        全部会话 <span>{{ conversations.length }}</span>
      </div>
      <button
        v-for="c in conversations"
        :key="c.sender"
        class="conversation"
        :class="{ selected: selected === c.sender }"
        @click="selected = c.sender"
      >
        <span class="avatar"><Icon name="MessageSquare" :size="20" /></span>
        <div>
          <div class="conversation-title">
            <strong>{{ c.sender }}</strong
            ><i v-if="c.unread" class="unread-dot"></i>
          </div>
          <p>{{ c.messages[0]?.content }}</p>
          <time>{{ date(c.messages[0]?.date || "") }}</time>
        </div>
      </button>
      <p v-if="!conversations.length" class="empty-text">
        {{ search ? "没有匹配的短信" : "暂无会话" }}
      </p>
    </aside>
    <section class="thread">
      <header class="thread-header">
        <div>
          <h2>{{ current?.sender || "短信中心" }}</h2>
          <p>
            {{
              current
                ? `${current.messages.length} 条本地记录`
                : "连接路由器后同步短信"
            }}
          </p>
        </div>
        <span class="tag"><Icon name="ShieldCheck" :size="12" />本地归档</span>
      </header>
      <p v-if="s.smsWarning" class="inline-warning">{{ s.smsWarning }}</p>
      <div class="thread-body">
        <div v-if="!current" class="large-empty">
          <Icon name="MessageSquare" :size="40" />
          <h2>让消息，触手可及</h2>
          <p>验证码和通知短信会保存在这台 Mac 上。</p>
        </div>
        <article
          v-for="m in [...(current?.messages || [])].reverse()"
          :key="m.id"
          class="message"
        >
          <time>{{ date(m.date) }}</time>
          <div class="bubble">{{ m.content }}</div>
          <div v-if="otp(m.content)" class="otp-inline">
            <span
              >验证码 <strong class="mono">{{ otp(m.content) }}</strong></span
            ><button
              class="text-button"
              @click="copyCode(m.id, otp(m.content))"
            >
              <Icon name="Copy" :size="14" />复制
            </button>
          </div>
        </article>
      </div>
      <footer class="thread-footer">
        <Icon name="Info" :size="15" /><span
          >首版仅接收与归档。发送、已读和删除操作将在固件验证后开放。</span
        >
      </footer>
    </section>
  </div>
</template>

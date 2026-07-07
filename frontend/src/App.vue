<template>
  <router-view />
</template>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
:root {
  --red: #fe2c55; --red-light: #ff4d6a;
  --bg: #f5f5f5; --text1: #121212; --text2: #666; --text3: #999;
  --border: #f0f0f0; --header-h: 64px;
  --max-width: 1120px;
  --card-gap: 16px;
}
body { font-family: -apple-system,BlinkMacSystemFont,'PingFang SC','Hiragino Sans GB','Microsoft YaHei',sans-serif; background: var(--bg); color: var(--text1); -webkit-font-smoothing: antialiased; }
a { text-decoration: none; color: inherit; }
img { display: block; max-width: 100%; }
</style>

# i18n

## 职责
中英双语国际化配置，浏览器自动检测语言，英文兜底

## 成员文件
- `index.js` → Vue I18n 初始化，浏览器语言检测 (非英语默认中文)，fallback 为英文
- `en.json` → 英文翻译 (20 个顶级 key：app/home/lobby/game/chat/night/vote/roles 等，含 22 个角色名+能力翻译)
- `zh.json` → 中文翻译 (20 个顶级 key，roles 含 83+ 角色名翻译 + 22 个能力翻译)

## 对外接口
- `default` → VueI18n 实例 (locale/fallbackLocale/messages)

## 依赖
无项目内部依赖

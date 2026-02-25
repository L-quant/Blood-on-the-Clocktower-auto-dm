# services

## 职责
前端服务层：REST API 客户端、音效播放管理

## 成员文件
- `ApiService.js` → REST API 客户端，JWT 认证、房间管理、事件查询、AI 助手接口
- `SoundService.js` → HTML5 Audio 音效管理，预加载与并发播放

## 对外接口
- `apiService.quickLogin(name) → Promise` → 快速登录 (POST /v1/auth/quick)
- `apiService.ensureAuth() → Promise` → 确保认证有效，自动重新登录
- `apiService.createRoom() → Promise<{room_id}>` → 创建房间
- `apiService.joinRoom(roomId) → Promise` → 加入房间
- `apiService.getRoomState(roomId) → Promise` → 获取房间状态
- `apiService.getEvents(roomId, afterSeq) → Promise` → 增量拉取事件
- `apiService.askAssistant(roomId, question, context) → Promise` → 查询 AI 助手
- `apiService.clearAuth()` → 清除认证信息
- `soundService.preload()` → 预加载音效
- `soundService.play(name)` → 播放音效
- `soundService.setMuted(muted)` → 设置静音
- `soundService.setVolume(vol)` → 设置音量

## 依赖
无项目内部依赖 (仅引用 assets/sounds/)

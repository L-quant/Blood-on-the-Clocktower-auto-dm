# Blood on the Clocktower - Agent DM

[English](#english) | [中文](#中文)

---

<a name="english"></a>
# English

A production-grade backend for a multiplayer real-time "Agent DM" game platform inspired by Blood on the Clocktower. Built with modern backend engineering practices including event sourcing, per-room sequential consistency, idempotency, visibility projection (information isolation), WebSocket realtime, and observability.

## Features

- **Event Sourcing**: All state changes are stored as immutable events with monotonic sequence numbers per room
- **Per-Room Sequential Consistency**: Single goroutine per room (RoomActor) serializes all command processing
- **Idempotency**: Command deduplication using `idempotency_key` per actor and command type
- **Visibility Projection**: Information isolation - players only see events they're allowed to see (whispers, roles, night actions)
- **WebSocket Real-time**: Live event broadcasting with resync support via `last_seq`
- **Observability**: OpenTelemetry tracing + Prometheus metrics + structured logging with zap
- **Agent/LLM Integration**: Stub narrator agent that emits non-authoritative `system_hint` events

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP/WebSocket Layer                      │
│  ┌─────────┐  ┌──────────┐  ┌──────────────────────────────┐   │
│  │  REST   │  │    WS    │  │         Auth (JWT)           │   │
│  │  API    │  │  Server  │  │                              │   │
│  └────┬────┘  └────┬─────┘  └──────────────────────────────┘   │
│       │            │                                            │
│  ┌────▼────────────▼────┐                                       │
│  │     Room Manager     │ ← Manages room actor lifecycle        │
│  └──────────┬───────────┘                                       │
│             │                                                    │
│  ┌──────────▼───────────┐                                       │
│  │     Room Actor       │ ← Single goroutine per room           │
│  │  ┌───────────────┐   │                                       │
│  │  │    Engine     │   │ ← Pure, deterministic game rules      │
│  │  └───────────────┘   │                                       │
│  └──────────┬───────────┘                                       │
│             │                                                    │
│  ┌──────────▼───────────┐                                       │
│  │   Event Store (DB)   │ ← MySQL with per-room sequences       │
│  └──────────────────────┘                                       │
└─────────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Go 1.25.5
- **HTTP Framework**: Chi
- **WebSocket**: Gorilla WebSocket
- **Database**: MySQL 8.0
- **Cache**: Redis 7
- **Logging**: zap (structured JSON)
- **Tracing**: OpenTelemetry
- **Metrics**: Prometheus

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25.5+

### 1. Start Infrastructure

```bash
docker-compose up -d
```

This starts:
- MySQL on port 3316
- Redis on port 6389
- Prometheus on port 9190
- Grafana on port 3100 (optional, admin:admin)

### 2. Build & Run Server

```bash
make build
./bin/agentdm
```

Or directly:

```bash
go run ./cmd/server
```

The server starts on `:8080` by default.

### 3. Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `DB_DSN` | `root:password@tcp(localhost:3316)/agentdm?...` | MySQL DSN |
| `REDIS_ADDR` | `localhost:6389` | Redis address |
| `JWT_SECRET` | `dev-secret-change` | JWT signing secret |
| `SNAPSHOT_INTERVAL` | `50` | Events between snapshots |
| `TRACE_STDOUT` | `true` | Print traces to stdout |

## API Reference

### Authentication

#### Register
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'
```

Response:
```json
{"token":"eyJ...","user_id":"uuid"}
```

#### Login
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'
```

### Rooms

#### Create Room
```bash
curl -X POST http://localhost:8080/v1/rooms \
  -H "Authorization: Bearer <token>"
```

Response:
```json
{"room_id":"uuid"}
```

#### Join Room
```bash
curl -X POST http://localhost:8080/v1/rooms/{room_id}/join \
  -H "Authorization: Bearer <token>"
```

#### Fetch Events
```bash
curl http://localhost:8080/v1/rooms/{room_id}/events?after_seq=0 \
  -H "Authorization: Bearer <token>"
```

#### Fetch State (Projected)
```bash
curl http://localhost:8080/v1/rooms/{room_id}/state \
  -H "Authorization: Bearer <token>"
```

#### Replay to Sequence
```bash
curl "http://localhost:8080/v1/rooms/{room_id}/replay?to_seq=10&viewer=user_id" \
  -H "Authorization: Bearer <token>"
```

### Health & Metrics

```bash
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

## WebSocket Protocol

Connect to: `ws://localhost:8080/ws?token=<jwt_token>`

### Client → Server Messages

#### Subscribe to Room
```json
{
  "type": "subscribe",
  "request_id": "req-1",
  "payload": {
    "room_id": "uuid",
    "last_seq": 0
  }
}
```

#### Send Command
```json
{
  "type": "command",
  "request_id": "req-2",
  "payload": {
    "command_id": "uuid",
    "idempotency_key": "unique-key",
    "room_id": "uuid",
    "type": "public_chat",
    "last_seen_seq": 5,
    "data": {"message": "Hello!"}
  }
}
```

#### Ping
```json
{"type": "ping", "request_id": "req-3", "payload": {}}
```

### Server → Client Messages

#### Event
```json
{
  "type": "event",
  "payload": {
    "room_id": "uuid",
    "seq": 6,
    "event_type": "public.chat",
    "data": {"message": "Hello!"},
    "server_ts": 1234567890123
  }
}
```

#### Command Result
```json
{
  "type": "command_result",
  "request_id": "req-2",
  "payload": {
    "command_id": "uuid",
    "status": "accepted",
    "applied_seq_from": 6,
    "applied_seq_to": 6
  }
}
```

#### Error
```json
{
  "type": "error",
  "request_id": "req-1",
  "payload": {
    "code": "forbidden",
    "message": "not a member of room"
  }
}
```

## Command Types

| Type | Phase | Description |
|------|-------|-------------|
| `join` | lobby | Join the game |
| `leave` | lobby | Leave the game |
| `start_game` | lobby | Start the game (DM only, ≥3 players) |
| `public_chat` | any | Send public message |
| `whisper` | any | Send private message |
| `nominate` | day | Nominate a player for execution |
| `vote` | day | Vote yes/no on nomination |
| `ability.use` | night | Use night ability |

## Event Sourcing Explained

1. **Commands** are validated against current state
2. **Events** are generated (0..N per command)
3. **Transaction** atomically:
   - Assigns monotonic sequence numbers via `room_sequences` table
   - Persists events
   - Records dedup entry
   - Optionally creates snapshot
4. **State** is updated in-memory by reducing events
5. **Broadcast** sends projected events to subscribers

### Resync Flow

When a client reconnects:
1. Client sends `subscribe` with `last_seq` (last seen sequence)
2. Server loads events after `last_seq` from database
3. Server sends missed events to client
4. Client is now synchronized

## Visibility Projection Rules

| Event Type | Visibility |
|------------|------------|
| `public.chat` | All room members |
| `whisper.sent` | Sender, recipient, DM |
| `role.assigned` | Target player, DM |
| `ability.resolved` | Actor, target, DM |
| Others | All room members |

## Testing

```bash
go test ./...
```

## Makefile Commands

```bash
make build        # Build binary
make run          # Build and run
make test         # Run tests
make tidy         # go mod tidy
make docker-up    # Start docker-compose
make docker-down  # Stop docker-compose
make clean        # Remove build artifacts
```

---

<a name="中文"></a>
# 中文

一个为多人实时 "Agent DM" 游戏平台设计的生产级后端，灵感来自《血染钟楼》。采用现代后端工程实践，包括事件溯源、房间级顺序一致性、幂等性、可见性投影（信息隔离）、WebSocket 实时通信和可观测性。

## 特性

- **事件溯源**：所有状态变更作为不可变事件存储，每个房间有单调递增的序列号
- **房间级顺序一致性**：每个房间一个 goroutine（RoomActor）串行处理所有命令
- **幂等性**：使用 `idempotency_key` 按用户和命令类型进行命令去重
- **可见性投影**：信息隔离 - 玩家只能看到允许看到的事件（私聊、角色、夜间行动）
- **WebSocket 实时通信**：实时事件广播，支持通过 `last_seq` 重新同步
- **可观测性**：OpenTelemetry 链路追踪 + Prometheus 指标 + zap 结构化日志
- **Agent/LLM 集成**：桩叙述者 Agent，发出非权威的 `system_hint` 事件

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP/WebSocket 层                         │
│  ┌─────────┐  ┌──────────┐  ┌──────────────────────────────┐   │
│  │  REST   │  │    WS    │  │         认证 (JWT)           │   │
│  │  API    │  │  服务器   │  │                              │   │
│  └────┬────┘  └────┬─────┘  └──────────────────────────────┘   │
│       │            │                                            │
│  ┌────▼────────────▼────┐                                       │
│  │     房间管理器        │ ← 管理房间 Actor 生命周期            │
│  └──────────┬───────────┘                                       │
│             │                                                    │
│  ┌──────────▼───────────┐                                       │
│  │     房间 Actor       │ ← 每个房间单独的 goroutine            │
│  │  ┌───────────────┐   │                                       │
│  │  │    引擎       │   │ ← 纯净的、确定性的游戏规则           │
│  │  └───────────────┘   │                                       │
│  └──────────┬───────────┘                                       │
│             │                                                    │
│  ┌──────────▼───────────┐                                       │
│  │   事件存储 (数据库)   │ ← MySQL 带房间级序列号               │
│  └──────────────────────┘                                       │
└─────────────────────────────────────────────────────────────────┘
```

## 技术栈

- **语言**：Go 1.25.5
- **HTTP 框架**：Chi
- **WebSocket**：Gorilla WebSocket
- **数据库**：MySQL 8.0
- **缓存**：Redis 7
- **日志**：zap（结构化 JSON）
- **链路追踪**：OpenTelemetry
- **指标**：Prometheus

## 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.25.5+

### 1. 启动基础设施

```bash
docker-compose up -d
```

这将启动：
- MySQL 端口 3316
- Redis 端口 6389
- Prometheus 端口 9190
- Grafana 端口 3100（可选，用户名密码：admin:admin）

### 2. 构建并运行服务器

```bash
make build
./bin/agentdm
```

或直接运行：

```bash
go run ./cmd/server
```

服务器默认在 `:8080` 启动。

### 3. 环境变量

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `HTTP_ADDR` | `:8080` | HTTP 服务器地址 |
| `DB_DSN` | `root:password@tcp(localhost:3316)/agentdm?...` | MySQL DSN |
| `REDIS_ADDR` | `localhost:6389` | Redis 地址 |
| `JWT_SECRET` | `dev-secret-change` | JWT 签名密钥 |
| `SNAPSHOT_INTERVAL` | `50` | 快照间隔事件数 |
| `TRACE_STDOUT` | `true` | 打印追踪到标准输出 |

## API 参考

### 认证

#### 注册
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'
```

响应：
```json
{"token":"eyJ...","user_id":"uuid"}
```

#### 登录
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'
```

### 房间

#### 创建房间
```bash
curl -X POST http://localhost:8080/v1/rooms \
  -H "Authorization: Bearer <token>"
```

响应：
```json
{"room_id":"uuid"}
```

#### 加入房间
```bash
curl -X POST http://localhost:8080/v1/rooms/{room_id}/join \
  -H "Authorization: Bearer <token>"
```

#### 获取事件
```bash
curl http://localhost:8080/v1/rooms/{room_id}/events?after_seq=0 \
  -H "Authorization: Bearer <token>"
```

#### 获取状态（投影后）
```bash
curl http://localhost:8080/v1/rooms/{room_id}/state \
  -H "Authorization: Bearer <token>"
```

#### 重放到指定序列
```bash
curl "http://localhost:8080/v1/rooms/{room_id}/replay?to_seq=10&viewer=user_id" \
  -H "Authorization: Bearer <token>"
```

### 健康检查和指标

```bash
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

## WebSocket 协议

连接地址：`ws://localhost:8080/ws?token=<jwt_token>`

### 客户端 → 服务器消息

#### 订阅房间
```json
{
  "type": "subscribe",
  "request_id": "req-1",
  "payload": {
    "room_id": "uuid",
    "last_seq": 0
  }
}
```

#### 发送命令
```json
{
  "type": "command",
  "request_id": "req-2",
  "payload": {
    "command_id": "uuid",
    "idempotency_key": "unique-key",
    "room_id": "uuid",
    "type": "public_chat",
    "last_seen_seq": 5,
    "data": {"message": "你好！"}
  }
}
```

#### Ping
```json
{"type": "ping", "request_id": "req-3", "payload": {}}
```

### 服务器 → 客户端消息

#### 事件
```json
{
  "type": "event",
  "payload": {
    "room_id": "uuid",
    "seq": 6,
    "event_type": "public.chat",
    "data": {"message": "你好！"},
    "server_ts": 1234567890123
  }
}
```

#### 命令结果
```json
{
  "type": "command_result",
  "request_id": "req-2",
  "payload": {
    "command_id": "uuid",
    "status": "accepted",
    "applied_seq_from": 6,
    "applied_seq_to": 6
  }
}
```

#### 错误
```json
{
  "type": "error",
  "request_id": "req-1",
  "payload": {
    "code": "forbidden",
    "message": "不是房间成员"
  }
}
```

## 命令类型

| 类型 | 阶段 | 描述 |
|------|------|------|
| `join` | 大厅 | 加入游戏 |
| `leave` | 大厅 | 离开游戏 |
| `start_game` | 大厅 | 开始游戏（仅 DM，≥3 玩家） |
| `public_chat` | 任意 | 发送公开消息 |
| `whisper` | 任意 | 发送私密消息 |
| `nominate` | 白天 | 提名一名玩家处决 |
| `vote` | 白天 | 对提名投票是/否 |
| `ability.use` | 夜晚 | 使用夜间技能 |

## 事件溯源详解

1. **命令** 根据当前状态进行验证
2. 生成 **事件**（每个命令 0..N 个）
3. **事务** 原子性地：
   - 通过 `room_sequences` 表分配单调序列号
   - 持久化事件
   - 记录去重条目
   - 可选地创建快照
4. 通过归约事件更新内存中的 **状态**
5. **广播** 将投影后的事件发送给订阅者

### 重新同步流程

当客户端重新连接时：
1. 客户端发送 `subscribe` 带 `last_seq`（最后看到的序列号）
2. 服务器从数据库加载 `last_seq` 之后的事件
3. 服务器发送遗漏的事件给客户端
4. 客户端现在已同步

## 可见性投影规则

| 事件类型 | 可见性 |
|----------|--------|
| `public.chat` | 所有房间成员 |
| `whisper.sent` | 发送者、接收者、DM |
| `role.assigned` | 目标玩家、DM |
| `ability.resolved` | 行动者、目标、DM |
| 其他 | 所有房间成员 |

## 测试

```bash
go test ./...
```

## Makefile 命令

```bash
make build        # 构建二进制文件
make run          # 构建并运行
make test         # 运行测试
make tidy         # go mod tidy
make docker-up    # 启动 docker-compose
make docker-down  # 停止 docker-compose
make clean        # 删除构建产物
```

---

## License

MIT License - see [LICENSE](LICENSE)

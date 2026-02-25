# store

## 职责
MySQL 数据访问层：用户/房间 CRUD、事件溯源 (追加/加载/快照)、幂等去重、事务管理

## 成员文件
- `models.go` → 数据模型定义：User、Room、RoomMember、DedupRecord、Snapshot、AgentRun
- `store.go` → 数据库连接与事务管理 (ConnectMySQL、WithTx)
- `event_store.go` → 事件溯源操作：追加事件、加载事件、快照、幂等去重
- `room_repo.go` → 房间与成员的 CRUD
- `user_repo.go` → 用户认证与查询

## 对外接口
- `New(db *sql.DB) *Store` → 创建 Store 实例
- `ConnectMySQL(dsn string) (*sql.DB, error)` → 建立 MySQL 连接 (含连接池配置)
- `(*Store) WithTx(ctx context.Context, fn func(*sql.Tx) error) error` → 执行事务
- `(*Store) Close() error` → 关闭数据库连接
- `(*Store) CreateUser(ctx context.Context, u User) error` → 创建用户
- `(*Store) GetUserByEmail(ctx context.Context, email string) (*User, error)` → 按邮箱查询用户
- `(*Store) GetUserByID(ctx context.Context, id string) (*User, error)` → 按 ID 查询用户
- `(*Store) CreateRoom(ctx context.Context, r Room) error` → 创建房间并初始化序号计数器
- `(*Store) GetRoom(ctx context.Context, id string) (*Room, error)` → 查询房间
- `(*Store) AddRoomMember(ctx context.Context, m RoomMember) error` → 添加/更新房间成员
- `(*Store) GetRoomMembers(ctx context.Context, roomID string) ([]RoomMember, error)` → 获取房间成员列表
- `(*Store) IsMember(ctx context.Context, roomID, userID string) (bool, string, error)` → 检查成员资格
- `(*Store) GetDedupRecord(ctx context.Context, roomID, actorUserID, idempotencyKey, commandType string) (*DedupRecord, error)` → 查询幂等记录
- `(*Store) SaveDedupRecord(ctx context.Context, tx *sql.Tx, r DedupRecord) error` → 保存幂等记录
- `(*Store) GetLatestSnapshot(ctx context.Context, roomID string) (*Snapshot, error)` → 获取最新快照
- `(*Store) SaveSnapshot(ctx context.Context, tx *sql.Tx, snap Snapshot) error` → 保存快照
- `(*Store) LoadEventsAfter(ctx context.Context, roomID string, afterSeq int64, limit int) ([]StoredEvent, error)` → 加载指定序号后的事件
- `(*Store) LoadEventsUpTo(ctx context.Context, roomID string, toSeq int64) ([]StoredEvent, error)` → 加载到指定序号的所有事件
- `(*Store) AppendEvents(ctx context.Context, roomID string, events []StoredEvent, dedup *DedupRecord, snap *Snapshot) error` → 原子追加事件+去重+快照

## 依赖
无内部依赖

# types

## 职责
全局共享类型定义：错误码、命令/事件信封、投影事件、观察者上下文

## 成员文件
- `types.go` → AppError 错误类型、CommandEnvelope、Event、CommandResult、ProjectedEvent、Viewer

## 对外接口
- `NewError(code ErrorCode, msg string) *AppError` → 创建应用错误
- `WrapError(code ErrorCode, msg string, err error) *AppError` → 包装底层错误为应用错误
- `Is(err error, code ErrorCode) bool` → 检查错误是否匹配指定错误码

## 依赖
无内部依赖

# auth

## 职责
JWT 令牌管理与 bcrypt 密码哈希，提供用户认证基础设施

## 成员文件
- `auth.go` → JWT 生成/解析与密码哈希/校验

## 对外接口
- `NewJWTManager(secret string, ttl time.Duration) *JWTManager` → 创建 JWT 管理器
- `(*JWTManager) Generate(userID string) (string, error)` → 为用户生成签名 JWT
- `(*JWTManager) Parse(tokenStr string) (*Claims, error)` → 解析并验证 JWT
- `HashPassword(pw string) (string, error)` → bcrypt 哈希密码
- `CheckPassword(hash, pw string) error` → 验证密码与哈希是否匹配

## 依赖
无内部依赖

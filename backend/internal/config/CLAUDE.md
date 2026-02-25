# config

## 职责
从环境变量加载应用配置，提供所有组件的默认值 (HTTP、DB、Redis、JWT、RabbitMQ、Qdrant、LLM、游戏计时)

## 成员文件
- `config.go` → 读取环境变量并返回 Config 结构体

## 对外接口
- `Load() Config` → 加载并返回完整应用配置

## 依赖
无内部依赖

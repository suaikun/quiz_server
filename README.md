# Quiz Server

一个基于 Go 的在线答题后端服务，支持用户注册/登录、随机抽题、成绩提交与排行榜查询。项目采用分层架构，使用 MySQL 持久化业务数据、Redis 提供排行榜高性能查询，适合作为暑期实习简历中的后端项目展示。

## 项目亮点

- 基于 `Gin` 搭建 RESTful API，完成完整答题业务闭环。
- 使用 `JWT` + 中间件实现鉴权，受保护接口仅允许登录用户访问。
- 使用 `bcrypt` 存储密码哈希，避免明文密码风险。
- 采用 `MySQL + Redis` 组合：MySQL 保证数据可靠性，Redis 支撑排行榜高效读取。
- 按 `handler/service/repository` 分层，依赖注入清晰，便于扩展和维护。
- 支持优雅停机（Graceful Shutdown）与连接池配置，贴近生产实践。

## 技术栈

- 语言：Go `1.25+`
- Web 框架：Gin
- 数据库：MySQL
- 缓存：Redis
- 鉴权：JWT (`github.com/golang-jwt/jwt/v5`)
- 安全：bcrypt (`golang.org/x/crypto/bcrypt`)

## 核心功能

- 用户模块
  - `POST /api/register`：用户注册
  - `POST /api/login`：用户登录并返回 JWT
- 答题模块（需 JWT）
  - `GET /api/quiz`：获取随机 5 题
  - `POST /api/submit`：提交成绩并尝试刷新个人最佳
- 排行榜模块
  - `GET /api/leaderboard`：查询 Top10 排行榜

## 架构设计

- `handler`：处理 HTTP 请求、参数校验、响应封装、鉴权中间件
- `service`：处理业务逻辑（注册登录、成绩判定、Token 生成）
- `repository`：负责 MySQL/Redis 数据读写
- `pkg/jwt`：JWT 生成与解析工具
- `config`：环境变量加载与统一配置管理

数据流：
`Client -> Gin Handler -> Service -> Repository -> MySQL/Redis`

## 项目结构

```text
quiz_server/
├─ cmd/
│  └─ main.go
├─ internal/
│  ├─ config/
│  ├─ handler/
│  ├─ model/
│  ├─ pkg/jwt/
│  ├─ repository/
│  └─ service/
├─ docs/
│  └─ schema.sql
├─ .env.example
├─ go.mod
└─ README.md
```

## 快速开始

### 1. 准备依赖

- MySQL 8.x
- Redis 6.x+
- Go 1.25+

### 2. 初始化数据库

执行 [`docs/schema.sql`](docs/schema.sql) 创建表结构。

### 3. 配置环境变量

复制 `.env.example`，按本地环境修改：

```bash
cp .env.example .env
```

Windows PowerShell 可直接设置环境变量后运行程序。

### 4. 启动服务

```bash
go run ./cmd
```

默认监听地址：`http://localhost:8080`

## 环境变量说明

| 变量名 | 说明 | 默认值 |
| --- | --- | --- |
| `SERVER_ADDR` | 服务监听地址 | `:8080` |
| `GIN_MODE` | Gin 运行模式 | `release` |
| `SHUTDOWN_TIMEOUT_SEC` | 优雅停机超时时间（秒） | `5` |
| `MYSQL_DSN` | MySQL 连接串（必填） | - |
| `DB_MAX_OPEN_CONNS` | 最大打开连接数 | `100` |
| `DB_MAX_IDLE_CONNS` | 最大空闲连接数 | `100` |
| `DB_CONN_MAX_LIFETIME_MIN` | 连接最大生命周期（分钟） | `60` |
| `REDIS_ADDR` | Redis 地址 | `127.0.0.1:6379` |
| `REDIS_PASSWORD` | Redis 密码 | 空 |
| `REDIS_DB` | Redis DB 编号 | `0` |
| `JWT_SECRET` | JWT 密钥（建议生产环境必配） | 开发默认值 |
| `JWT_EXPIRE_HOURS` | JWT 过期时间（小时） | `24` |

## API 示例

### 注册

```http
POST /api/register
Content-Type: application/json

{
  "username": "alice",
  "password": "123456"
}
```

### 登录

```http
POST /api/login
Content-Type: application/json

{
  "username": "alice",
  "password": "123456"
}
```

### 获取题目（带 Token）

```http
GET /api/quiz
Authorization: Bearer <JWT_TOKEN>
```

### 提交成绩（带 Token）

```http
POST /api/submit
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "username": "alice",
  "score": 80,
  "time_taken": 120
}
```

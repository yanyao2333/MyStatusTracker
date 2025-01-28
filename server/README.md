# Server

服务端

## 快速开始

### 配置文件

创建 `.env`，并仿照 `.env.example` 填写环境变量

### 启动服务

```bash
go run server
```

## API 端点

### 1. 实时状态推送 (SSE)

```text
GET /events
```

**响应示例**：

```json
{ 
  "timestamp":1738030973,
  "status":"摸鱼中🤲🐟",
  "status_code":1,
  "software":"VSCode",
  "message":"正在使用 VSCode 写代码👨‍💻\\n当前状态：「摸鱼中🤲🐟」"
}
```

### 2. 更新用户状态

```text
POST /update-status
```

**鉴权要求**：需要有效密码

**请求头**：

```http
Content-Type: application/json
X-Password: your_secure_password_here
```

**请求体**：

```json
{
  "status": "忙碌",
  "status_code": "1" // 1 为在线，2 为离线
}
```

**成功响应**：`200 OK`

### 3. 更新使用软件

```text
POST /update-software
```

**鉴权要求**：需要有效密码

**请求头**：

```http
Content-Type: application/json
X-Password: your_secure_password_here
```

**请求体**：

```json
{
  "software": "VSCode",
  "message": "正在使用 VSCode 写代码" // 如果没有 message 字段，会自动生成为 `正在使用「${software}」` 格式显示
}
```

**成功响应**：`200 OK`

## 鉴权说明

- 使用HTTP头认证方式
- 需要添加以下请求头：

  ```http
  X-Password: [配置文件中的密码]
  ```

- 仅影响状态更新接口，SSE接口无需鉴权

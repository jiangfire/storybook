# REST API 设计

> 基于Gin框架的RESTful API设计，所有响应使用统一格式。

---

## 一、API通用规范

### 1.1 基础URL

```
开发环境: http://localhost:8080/api
生产环境: https://api.storybook.com/api
```

### 1.2 请求头

```http
Content-Type: application/json
Authorization: Bearer <JWT_TOKEN>
Accept-Language: zh-CN
```

### 1.3 统一响应格式

**成功响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应**：
```json
{
  "code": 40001,
  "message": "参数验证失败",
  "errors": [
    {
      "field": "email",
      "message": "邮箱格式不正确"
    }
  ]
}
```

### 1.4 状态码规范

| Code | HTTP Status | 描述 |
|------|-------------|------|
| 0 | 200 | 成功 |
| 40001 | 400 | 参数验证失败 |
| 40101 | 401 | 未登录 |
| 40102 | 401 | Token过期 |
| 40301 | 403 | 权限不足 |
| 40401 | 404 | 资源不存在 |
| 40901 | 409 | 资源冲突 |
| 50001 | 500 | 服务器内部错误 |

---

## 二、认证相关 API

### 2.1 用户注册

**请求**：
```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "Pass1234",
  "role": "developer"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "注册成功",
  "data": {
    "user": {
      "id": 1,
      "email": "user@example.com",
      "role": "developer"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-01-16T10:00:00Z"
  }
}
```

**验证AC映射**：
- AC-1.1.1: 邮箱密码注册
- AC-1.1.3: 密码强度验证
- AC-1.1.4: 角色选择
- AC-1.1.5: 返回JWT Token
- AC-1.1.7: 邮箱唯一性

---

### 2.2 用户登录

**请求**：
```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "Pass1234"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "user": {
      "id": 1,
      "email": "user@example.com",
      "role": "developer"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-01-16T10:00:00Z"
  }
}
```

**验证AC映射**：
- AC-1.2.1: 正确凭证登录
- AC-1.2.2: 返回JWT Token
- AC-1.2.5: 错误凭证提示

---

### 2.3 刷新Token

**请求**：
```http
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应**：
```json
{
  "code": 0,
  "message": "Token刷新成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-01-16T10:00:00Z"
  }
}
```

---

## 三、项目相关 API

### 3.1 创建项目

**请求**：
```http
POST /api/projects
Authorization: Bearer <TOKEN>
Content-Type: application/json

{
  "name": "电商平台",
  "description": "在线购物平台",
  "agile_mode": "kanban"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "项目创建成功",
  "data": {
    "id": 1,
    "name": "电商平台",
    "description": "在线购物平台",
    "agile_mode": "kanban",
    "owner": {
      "id": 1,
      "email": "user@example.com"
    },
    "created_at": "2025-01-15T10:00:00Z"
  }
}
```

**验证AC映射**：
- AC-2.1.1: 项目名称验证
- AC-2.1.2: 描述字段
- AC-2.1.3: 敏捷模式选择
- AC-2.1.4: 创建者为Owner

---

### 3.2 获取项目列表

**请求**：
```http
GET /api/projects?page=1&limit=20&search=电商
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "projects": [
      {
        "id": 1,
        "name": "电商平台",
        "description": "在线购物平台",
        "agile_mode": "kanban",
        "owner": {
          "id": 1,
          "email": "user@example.com"
        },
        "member_count": 5,
        "story_count": 23,
        "is_owner": true,
        "created_at": "2025-01-15T10:00:00Z"
      }
    ],
    "total": 45,
    "page": 1,
    "limit": 20
  }
}
```

**验证AC映射**：
- AC-2.2.1: 显示创建的项目
- AC-2.2.2: 显示参与的项目
- AC-2.2.3: 搜索功能
- AC-2.2.4: 敏捷模式
- AC-2.2.5: 成员数量

---

### 3.3 获取项目详情

**请求**：
```http
GET /api/projects/1
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "电商平台",
    "description": "在线购物平台",
    "agile_mode": "kanban",
    "owner": {
      "id": 1,
      "email": "user@example.com"
    },
    "members": [
      {
        "user": {
          "id": 2,
          "email": "dev@example.com"
        },
        "role_in_project": "developer"
      }
    ],
    "statistics": {
      "total_stories": 23,
      "status_breakdown": {
        "backlog": 5,
        "ready": 3,
        "in_progress": 8,
        "test": 4,
        "done": 3
      },
      "completion_rate": 13.0
    },
    "created_at": "2025-01-15T10:00:00Z"
  }
}
```

---

## 四、用户故事 API

### 4.1 创建用户故事

**请求**：
```http
POST /api/projects/1/stories
Authorization: Bearer <TOKEN>
Content-Type: application/json

{
  "title": "用户登录功能",
  "description": "作为已注册用户，我想要通过邮箱和密码登录...",
  "story_type": "feature",
  "priority": 3,
  "story_points": 5,
  "acceptance_criteria": [
    {
      "id": "ac-1",
      "description": "可以输入邮箱和密码登录",
      "order": 1
    },
    {
      "id": "ac-2",
      "description": "登录成功返回JWT Token",
      "order": 2
    }
  ]
}
```

**响应**：
```json
{
  "code": 0,
  "message": "用户故事创建成功",
  "data": {
    "id": 1,
    "project_id": 1,
    "title": "用户登录功能",
    "description": "作为已注册用户，我想要通过邮箱和密码登录...",
    "story_type": "feature",
    "status": "backlog",
    "priority": 3,
    "story_points": 5,
    "acceptance_criteria": [
      {
        "id": "ac-1",
        "ref": "AC-3.1.1",
        "description": "可以输入邮箱和密码登录",
        "status": "pending",
        "order": 1
      }
    ],
    "created_by": {
      "id": 1,
      "email": "user@example.com"
    },
    "created_at": "2025-01-15T10:00:00Z"
  }
}
```

**验证AC映射**：
- AC-3.1.1: 标题验证
- AC-3.1.3: 故事类型
- AC-3.1.4: 优先级
- AC-3.1.5: 故事点
- AC-3.1.6: 验收标准
- AC-3.1.7: 默认待办状态

---

### 4.2 获取项目故事列表（看板视图）

**请求**：
```http
GET /api/projects/1/stories?status=in_progress&assignee=1
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "stories": [
      {
        "id": 1,
        "title": "用户登录功能",
        "story_type": "feature",
        "status": "in_progress",
        "priority": 3,
        "story_points": 5,
        "position": 1.0,
        "assigned_to": {
          "id": 2,
          "email": "dev@example.com"
        },
        "acceptance_criteria_summary": {
          "total": 7,
          "passed": 5,
          "pending": 2,
          "completion_percentage": 71.4
        }
      }
    ],
    "total": 8
  }
}
```

---

### 4.3 更新故事状态

**请求**：
```http
PATCH /api/stories/1/status
Authorization: Bearer <TOKEN>
Content-Type: application/json

{
  "status": "test",
  "position": 1.0
}
```

**响应**：
```json
{
  "code": 0,
  "message": "状态更新成功",
  "data": {
    "id": 1,
    "status": "test",
    "position": 1.0,
    "updated_at": "2025-01-15T11:00:00Z"
  }
}
```

**验证AC映射**：
- AC-4.2.2: 状态自动更新
- AC-4.2.3: 活动日志记录

---

### 4.4 领取故事

**请求**：
```http
POST /api/stories/1/claim
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "故事领取成功",
  "data": {
    "id": 1,
    "assigned_to": {
      "id": 1,
      "email": "user@example.com"
    },
    "status": "in_progress",
    "updated_at": "2025-01-15T11:00:00Z"
  }
}
```

**验证AC映射**：
- AC-3.4.1: 分配给自己
- AC-3.4.2: 状态变为开发中
- AC-3.4.3: 活动日志

---

### 4.5 释放故事

**请求**：
```http
DELETE /api/stories/1/claim
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "故事释放成功",
  "data": {
    "id": 1,
    "assigned_to": null,
    "status": "ready",
    "updated_at": "2025-01-15T11:00:00Z"
  }
}
```

---

### 4.6 获取故事详情

**请求**：
```http
GET /api/stories/1
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "project": {
      "id": 1,
      "name": "电商平台"
    },
    "title": "用户登录功能",
    "description": "作为已注册用户，我想要通过邮箱和密码登录...",
    "story_type": "feature",
    "status": "in_progress",
    "priority": 3,
    "story_points": 5,
    "assigned_to": {
      "id": 2,
      "email": "dev@example.com"
    },
    "created_by": {
      "id": 1,
      "email": "pm@example.com"
    },
    "acceptance_criteria": [
      {
        "id": "ac-1",
        "ref": "AC-3.1.1",
        "description": "可以输入邮箱和密码登录",
        "status": "passed",
        "evidence": "见 auth/handler_test.go:45",
        "order": 1
      }
    ],
    "tags": ["authentication", "security"],
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T11:30:00Z"
  }
}
```

**验证AC映射**：
- AC-5.1.1: 完整信息展示
- AC-5.1.2: 验收标准列表
- AC-5.1.3: AC可勾选
- AC-5.1.4: 活动历史

---

### 4.7 更新AC状态

**请求**：
```http
PATCH /api/stories/1/acceptance-criteria/ac-1
Authorization: Bearer <TOKEN>
Content-Type: application/json

{
  "status": "passed",
  "evidence": "见 auth/handler.go:89"
}
```

**响应**：
```json
{
  "code": 0,
  "message": "AC状态更新成功",
  "data": {
    "id": "ac-1",
    "status": "passed",
    "evidence": "见 auth/handler.go:89",
    "updated_at": "2025-01-15T12:00:00Z"
  }
}
```

---

### 4.8 获取活动历史

**请求**：
```http
GET /api/stories/1/activities?page=1&limit=20
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "activities": [
      {
        "id": 1,
        "entity_type": "story",
        "entity_id": 1,
        "action": "status_changed",
        "old_value": {
          "status": "backlog"
        },
        "new_value": {
          "status": "in_progress"
        },
        "user": {
          "id": 2,
          "email": "dev@example.com",
          "avatar_url": "/avatars/2.jpg"
        },
        "created_at": "2025-01-15T11:00:00Z"
      },
      {
        "id": 2,
        "entity_type": "story",
        "entity_id": 1,
        "action": "updated",
        "old_value": {
          "priority": 2
        },
        "new_value": {
          "priority": 3
        },
        "user": {
          "id": 1,
          "email": "pm@example.com"
        },
        "created_at": "2025-01-15T10:30:00Z"
      }
    ],
    "total": 15,
    "page": 1,
    "limit": 20
  }
}
```

**验证AC映射**：
- AC-5.2.1: 状态变更记录
- AC-5.2.2: 字段修改记录
- AC-5.2.3: 操作人信息
- AC-5.2.4: 新旧值对比

---

## 五、统计相关 API

### 5.1 项目概览

**请求**：
```http
GET /api/projects/1/overview
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "project": {
      "id": 1,
      "name": "电商平台"
    },
    "statistics": {
      "total_stories": 23,
      "status_breakdown": {
        "backlog": 5,
        "ready": 3,
        "in_progress": 8,
        "test": 4,
        "done": 3
      },
      "completion_rate": 13.0,
      "active_members": 5,
      "avg_story_points": 4.5
    },
    "recent_activities": [
      {
        "action": "story_created",
        "user": "张三",
        "description": "创建了故事「购物车功能」",
        "created_at": "2025-01-15T11:00:00Z"
      }
    ]
  }
}
```

**验证AC映射**：
- AC-7.1.1: 总故事数
- AC-7.1.2: 状态分布
- AC-7.1.3: 完成率
- AC-7.1.4: 活跃成员

---

### 5.2 个人工作台

**请求**：
```http
GET /api/me/dashboard
Authorization: Bearer <TOKEN>
```

**响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": 1,
      "email": "user@example.com",
      "role": "developer"
    },
    "my_stories": {
      "assigned": [
        {
          "id": 1,
          "title": "用户登录功能",
          "project": "电商平台",
          "status": "in_progress",
          "priority": 3
        }
      ],
      "created": [
        {
          "id": 2,
          "title": "注册功能",
          "project": "电商平台",
          "status": "backlog"
        }
      ]
    },
    "statistics": {
      "total_assigned": 8,
      "in_progress": 3,
      "completed": 5
    }
  }
}
```

---

## 六、错误响应示例

### 6.1 参数验证失败

```http
HTTP/1.1 400 Bad Request

{
  "code": 40001,
  "message": "参数验证失败",
  "errors": [
    {
      "field": "email",
      "message": "邮箱格式不正确"
    },
    {
      "field": "password",
      "message": "密码至少8位，包含字母和数字"
    }
  ]
}
```

### 6.2 认证失败

```http
HTTP/1.1 401 Unauthorized

{
  "code": 40101,
  "message": "未登录或Token无效",
  "data": null
}
```

### 6.3 权限不足

```http
HTTP/1.1 403 Forbidden

{
  "code": 40301,
  "message": "权限不足，只有产品经理可以创建用户故事",
  "data": null
}
```

### 6.4 资源不存在

```http
HTTP/1.1 404 Not Found

{
  "code": 40401,
  "message": "用户故事不存在",
  "data": null
}
```

---

## 七、WebSocket 实时通知

### 7.1 连接

```
ws://localhost:8080/ws?token=<JWT_TOKEN>
```

### 7.2 消息格式

**故事状态变更**：
```json
{
  "type": "story.status_changed",
  "data": {
    "story_id": 1,
    "project_id": 1,
    "old_status": "backlog",
    "new_status": "in_progress",
    "actor": {
      "id": 2,
      "email": "dev@example.com"
    }
  },
  "timestamp": "2025-01-15T11:00:00Z"
}
```

**AC状态更新**：
```json
{
  "type": "story.ac_updated",
  "data": {
    "story_id": 1,
    "ac_id": "ac-1",
    "ac_status": "passed",
    "actor": {
      "id": 2,
      "email": "dev@example.com"
    }
  },
  "timestamp": "2025-01-15T11:00:00Z"
}
```

---

## 八、API限流

### 8.1 限流规则

| 接口类型 | 限制 |
|---------|------|
| 登录/注册 | 同IP每分钟3次 |
| API请求 | 同用户每分钟100次 |
| WebSocket连接 | 同用户最多5个并发连接 |

### 8.2 响应头

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642252800
```

---

## 九、MCP专用API

MCP服务将通过内部HTTP端口暴露，不对外公开：

### 9.1 获取故事的AC列表

```
GET /mcp/stories/:id/acceptance-criteria
```

### 9.2 检查AC覆盖度

```
GET /mcp/stories/:id/ac-coverage
```

### 9.3 更新AC状态

```
POST /mcp/stories/:id/acceptance-criteria/:ac_id/status
```

详见 [05-MCP服务设计.md](./05-MCP服务设计.md)

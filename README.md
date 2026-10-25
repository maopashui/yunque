# 云雀 YunQue

在线 Markdown 文档管理与实时协作平台。

## 功能概览

- 账号登录与按用户名分享（所有者 / 可编辑 / 只读）
- 首位注册用户为管理员；可在管理页开关公开注册
- 文档管理：创建、导入导出、收藏、回收站
- 阅读视图：虚拟滚动、目录、搜索定位
- WYSIWYG 编辑：标题、列表、表格、代码高亮、图片、文字颜色等
- 实时协作：在线用户、远程光标、Yjs 增量同步
- 历史版本：自动/手动快照、预览对比、恢复

## Docker 部署

单容器：前端静态资源由后端同源托管（`/api` + SPA），数据落在卷 `/data`。

生产建议：在反向代理上终止 TLS，并保持 `YUNQUE_COOKIE_SECURE=true`。首次启动后注册的第一个账号为管理员，建议立即在「管理」页按需关闭公开注册。

```bash
# 构建并启动（生产默认 Cookie Secure=true，需 HTTPS）
docker compose up -d --build

# 本地明文调试时可覆盖：
# YUNQUE_COOKIE_SECURE=false docker compose up -d --build

# 打开
open http://localhost:8080
```

等价命令：

```bash
docker build -t yunque .
docker run --rm -p 8080:8080 -v yunque-data:/data \
  -e YUNQUE_ALLOWED_ORIGIN= \
  -e YUNQUE_COOKIE_SECURE=true \
  yunque
```

### 环境变量

| 变量 | 默认（镜像内） | 说明 |
|------|----------------|------|
| `YUNQUE_ADDR` | `:8080` | 监听地址 |
| `YUNQUE_DB_PATH` | `/data/yunque.db` | SQLite 路径 |
| `YUNQUE_ASSETS_DIR` | `/data/assets` | 文档图片等资源 |
| `YUNQUE_STATIC_DIR` | `/app/static` | 前端 `dist`（空则不托管 SPA） |
| `YUNQUE_ALLOWED_ORIGIN` | 空 | CORS；同源部署保持空 |
| `YUNQUE_COOKIE_SECURE` | `false`（compose 生产示例为 `true`） | HTTPS 时设为 `true` |
| `YUNQUE_MIGRATIONS_DIR` | `/app/migrations` | 迁移目录 |

生产若挂在域名后，将 `YUNQUE_ALLOWED_ORIGIN` 设为对外 Origin（如 `https://docs.example.com`），并开启 `YUNQUE_COOKIE_SECURE=true`。公开注册由管理员在应用内开关，无需环境变量。

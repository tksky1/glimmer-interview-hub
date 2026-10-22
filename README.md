# 微光工作室 · 招新面试预约平台

全新的前后端独立部署项目：管理员在日历上批量发布面试时段，成员使用**旧微光账号**登录后选择 3 个志愿时段，系统按偏好自动分配并支持人工调整确认。

## 功能特性

- **免登录接入**：复用旧系统（glimmer-backend / vue-glimemr-main）的 JWT 登录态，不实现登录注册页；登录后自动返回本平台
- **方向隔离**：场次按方向（C/计系、Java、Web、ML）创建，成员只能看到并预约本方向的日程，不同方向互不冲突
- **邀请码绑定**：旧系统不提供方向信息，成员通过管理员生成的邀请码/专属链接（`/interview/?invite=码`）登录后绑定方向
- **三日历**：日历同时展示三天，时间轴 5 分钟精度；拖拽框选贴近边缘自动滚动，已框选范围持久高亮
- **心仪时间**：成员每场次选择 3 个时段并按偏好排序（即时高亮、可排序增删），时段上实时显示「首选人数」
- **智能分配**：最小费用流算法按偏好自动分配（优先满足第一心仪），管理员可预览统计、改派、对调
- **确认生效**：管理员确认后面试者即可看到确定的面试时间与地点，心仪时间锁定；管理员仍可随时修改
- **一键部署**：Docker Compose 编排 backend（Go）+ web（Nginx + 前端静态资源）两个服务

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.23 + Gin + GORM + SQLite（纯 Go 驱动，免 CGO） |
| 前端 | Vue 3 + Vite + Element Plus + Pinia + Vue Router + Axios |
| 部署 | Docker Compose（backend + web 两服务，SQLite 数据卷持久化） |

## 快速开始（本地预览）

```bash
# 1. 配置环境变量（按需修改）
cp .env.example .env
#    编辑 .env：WEB_PORT 为访问端口；ADMIN_USERNAMES 填入你的旧系统用户名（管理员）

# 2. 一键构建并启动
docker compose up -d --build

# 3. 访问
#    面试平台   http://localhost:8081/interview/
#    旧登录页   http://localhost:8081/submit/login
```

首次构建需要在容器内下载 Go 模块与 npm 依赖（已配置 goproxy.cn / npmmirror 国内镜像），约 3~5 分钟；之后构建有缓存，很快。

### 登录闭环说明

本地预览时，Nginx 同源托管两部分：

- `/interview/` — 新面试平台
- `/submit/` — 旧前端预构建产物（`deploy/legacy/`，仅本地预览用）

旧登录页直接请求线上 `api.glimmer.org.cn` 完成登录，JWT 写入同一 origin 的 `localStorage`，新平台立即可读——因此无需在新平台实现任何登录注册页面。

**登录返回**：由于同标签页内 `storage` 事件不触发，`deploy/legacy/index.html` 中注入了一个桥接脚本（仅本地预览拷贝有）：检测到登录成功（token 出现）后显示倒计时横幅，5 秒后自动跳回 `/interview/`，也可点「留在本页」取消。若你在多个标签页分别打开面试平台和旧登录页，则登录后面试平台标签页会通过 `storage` 事件自动进入，无需跳转。

> 生产部署时：将新前端放到与旧系统相同的域名下（如 `glimmer.org.cn/interview/`），并删除 `deploy/legacy/` 与 nginx 中 `/submit/` 配置，改为直接指向线上旧系统登录地址。

## 配置项（.env）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `WEB_PORT` | `8081` | Web 服务对外端口 |
| `JWT_SECRET` | `RecruitmentSystem` | 本地验签快路径密钥；旧系统线上密钥不同时自动走委派校验，无需修改 |
| `LEGACY_API_BASE` | `https://api.glimmer.org.cn` | 旧系统 API 地址（委派校验 token，需容器可访问） |
| `ADMIN_USERNAMES` | 空 | 初始管理员（旧系统登录名，逗号分隔），仅首次启动时种子化 |

## 登录态校验机制

旧系统线上部署的 JWT 密钥可能与仓库代码不同（且将来可能轮换），因此后端采用两级校验：

1. **本地快路径**：用 `JWT_SECRET` 验签，命中则零网络开销；
2. **委派校验**：验签失败时免验签提取身份，带 token 请求旧系统 API（`GET /problem`），以其 200/401 为权威结论，结果按 token 缓存 10 分钟；旧 API 网络不可达时，对结构合法且未过期的 token 降级放行，避免误锁。

管理员判定：内置 `admins` 名单 **或** 旧系统令牌中的 `role=admin`。

## 使用流程

1. **管理员**：登录后进入「管理后台」→「方向邀请码」生成邀请码（或复制专属链接）发给同学
2. **成员**：点击专属链接（或进入平台后输入邀请码）→ 用微光账号登录 → 自动绑定方向
3. **管理员**：「新建面试场次」→ 选择方向 → 在日历某一天内拖拽框选时间 → 设置每场时长、地点、联系人 → 创建 → 发布
4. **成员**：在首页日历点击时段选 3 个心仪时间（可排序、增删）→ 保存
5. **管理员**：预约截止后进入「分配管理」→「生成分配」→ 查看心仪达成统计 → 按需改派/对调 →「确认分配」
6. **成员**：在首页与「我的心仪时间」查看已确认的面试时间与地点

## 目录结构

```
├── backend/            # Go 后端（Gin + GORM + SQLite）
│   ├── cmd/server/     # 程序入口
│   └── internal/       # config/model/auth/middleware/handler/service/repository/algorithm
├── frontend/           # Vue3 前端
│   └── src/
│       ├── components/ # AppHeader / LoginGate / WeekCalendar
│       └── views/      # member（Home/MySchedule）+ admin（Dashboard/SessionEdit/Assignment）
├── deploy/
│   ├── nginx/          # 同源托管与反代配置
│   └── legacy/         # 旧前端预构建产物（本地预览登录用）
├── web/Dockerfile      # 前端多阶段构建 + nginx
├── docker-compose.yml
└── docs/screenshots/   # 界面截图
```

## 数据与运维

- 数据存储于 Docker 卷 `interview-data`（SQLite 单文件 `/data/interview.db`），备份只需复制该卷
- 重置全部数据：`docker compose down -v && docker compose up -d`
- 查看后端日志：`docker compose logs -f backend`

## API 概览

统一响应 `{code, msg, data}`，除 `/api/health` 外均需请求头携带旧系统 JWT（头名 `token`）。

- `GET /api/me` 当前用户与是否管理员
- `GET /api/sessions`、`GET /api/sessions/{id}` 场次列表/详情（含时段首选人数、我的志愿/分配）
- `PUT /api/preferences` 保存三志愿
- `GET /api/my/schedule` 我的预约总览
- `POST /api/admin/sessions` 等一整套管理员接口（发布/分配/改派/对调/确认/管理员名单）

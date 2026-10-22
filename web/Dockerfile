# 多阶段构建：node 构建前端 -> nginx 托管静态资源并反代 API
FROM node:20-alpine AS build

WORKDIR /app

# 使用国内 npm 镜像加速依赖安装
RUN npm config set registry https://registry.npmmirror.com

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund

COPY frontend/ ./
RUN npm run build

FROM nginx:alpine

# 新前端（base=/interview/）
COPY --from=build /app/dist /usr/share/nginx/html/interview
# 旧前端预构建产物（仅用于本地预览时同源登录闭环）
COPY deploy/legacy /usr/share/nginx/html/submit
# nginx 配置
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

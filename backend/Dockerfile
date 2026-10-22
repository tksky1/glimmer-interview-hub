# 多阶段构建：纯 Go 驱动（modernc sqlite），无需 CGO
FROM golang:1.23-alpine AS build

# 国内模块代理加速
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20

WORKDIR /
COPY --from=build /server /server

# 时区由 Go 内嵌 tzdata 提供（main.go 中 import _ "time/tzdata"）
ENV TZ=Asia/Shanghai

# SQLite 数据文件持久化目录
VOLUME /data
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/server"]

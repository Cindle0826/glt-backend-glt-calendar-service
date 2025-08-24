# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build
WORKDIR /src

# 可選：加快 go mod cache
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# 複製其餘程式碼
COPY . .

# 建置 Linux/amd64 的 bootstrap（LocalStack on Windows/amd64 常見）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o bootstrap .

# 打包 Lambda ZIP（zip 內需有 bootstrap 於根目錄）
RUN apk add --no-cache zip && zip -9 function.zip bootstrap

FROM scratch AS export
COPY --from=build /src/function.zip /out/function.zip
CMD ["/out/function.zip"]
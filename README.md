# send-ipv6-cloudflare_DDNS

一个用 Go 语言编写的工具，用于获取本机 IPv6 地址，提供 HTTP 接口供其他设备查询，并可选择性地更新 Cloudflare DNS 记录。

## 功能特点

- 提供 HTTP 接口 `/getip` 用于查询本机 IPv6 地址
- 定时推送 IPv6 地址变更通知
- 支持自动更新 Cloudflare DNS 记录
- 支持配置文件和命令行参数配置
- 自动过滤本地链路和私有 IPv6 地址

## 配置说明

### 配置文件 (config.json)

```json
{
    "clientName": "设备名称",
    "interval": 5,
    "httpPort": "8080",
    "baseURL": "http://your-notification-service.com/api/",
    "cloudflareToken": "your-cloudflare-api-token",
    "cloudflareZone": "your-zone-id",
    "cloudflareID": "your-dns-record-id",
    "cloudflareDomain": "your-domain.com",
    "cloudflareBaseURL": "https://api.cloudflare.com",
    "cloudflareEmail": "your-cloudflare-email"
}
```

### 配置项说明

| 配置项 | 说明 |
|--------|------|
| clientName | 设备标识名称 |
| interval | IPv6 地址检查和通知间隔（分钟） |
| httpPort | HTTP 服务监听端口 |
| baseURL | 通知服务的基础 URL |
| cloudflareToken | Cloudflare API 令牌 |
| cloudflareZone | Cloudflare Zone ID |
| cloudflareID | DNS 记录 ID |
| cloudflareDomain | 需要更新的域名 |
| cloudflareBaseURL | Cloudflare API 基础 URL（可选） |
| cloudflareEmail | Cloudflare 账户邮箱 |

## 使用方法

### 命令行参数

```bash
send-ipv6-cloudflare_DDNS.exe -n 设备名称 -i 检查间隔 -p HTTP端口
```

| 参数 | 说明 |
|------|------|
| -n | 设置客户端名称（可选，默认使用配置文件值） |
| -i | 设置检查间隔，单位为分钟（可选，默认使用配置文件值） |
| -p | 设置 HTTP 服务端口（可选，默认使用配置文件值） |

### HTTP API

获取 IPv6 地址：`http://your-server:port/getip`

响应示例：
```json
{
    "ipv6": "2001:db8::1"
}
```

## 注意事项

1. 确保系统已启用 IPv6 网络
2. 如果要使用 Cloudflare DNS 更新功能，需要正确配置所有 Cloudflare 相关参数
3. 程序会自动过滤本地链路地址和私有 IPv6 地址
4. 建议将程序设置为系统服务，确保开机自动运行

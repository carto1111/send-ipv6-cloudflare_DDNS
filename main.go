package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Config struct {
	ClientName string `json:"clientName"`
	Interval   int    `json:"interval"`
	HttpPort   string `json:"httpPort"`
	BaseURL    string `json:"baseURL"`
	// Cloudflare配置
	CloudflareToken   string `json:"cloudflareToken"`
	CloudflareZone    string `json:"cloudflareZone"`
	CloudflareID      string `json:"cloudflareID"`
	CloudflareDomain  string `json:"cloudflareDomain"`
	CloudflareBaseURL string `json:"cloudflareBaseURL"`
	CloudflareEmail   string `json:"cloudflareEmail"`
}

var (
	// 复用 HTTP 客户端以减少资源消耗
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	config Config
)

// 添加新的处理函数
func getIPv6Handler(w http.ResponseWriter, r *http.Request) {
	interfaces, err := net.Interfaces()
	if err != nil {
		http.Error(w, fmt.Sprintf("获取网络接口失败: %v", err), http.StatusInternalServerError)
		return
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		if ipv6 := findIPv6(addrs); ipv6 != "" {
			response := map[string]string{"ipv6": ipv6}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	http.Error(w, "未找到可用的IPv6地址", http.StatusNotFound)
}

func loadConfig() error {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	return nil
}

func main() {
	// 加载配置文件
	if err := loadConfig(); err != nil {
		fmt.Printf("警告：%v，将使用默认配置和命令行参数\n", err)
	}

	// 定义命令行参数，使用配置文件中的值作为默认值
	clientName := flag.String("n", config.ClientName, "客户端名称")
	interval := flag.Int("i", config.Interval, "发送间隔（分钟）")
	httpPort := flag.String("p", config.HttpPort, "HTTP服务端口")
	flag.Parse()

	// 更新配置
	config.ClientName = *clientName
	config.Interval = *interval
	config.HttpPort = *httpPort

	// 设置HTTP路由
	http.HandleFunc("/getip", getIPv6Handler)

	// 在新的goroutine中启动HTTP服务器
	go func() {
		fmt.Printf("HTTP服务器启动在端口 %s\n", config.HttpPort)
		if err := http.ListenAndServe(":"+config.HttpPort, nil); err != nil {
			fmt.Printf("HTTP服务器启动失败: %v\n", err)
		}
	}()

	sendIPNotification(config.ClientName, config.BaseURL)

	for {
		fmt.Printf("等待 %d 分钟后重新发送...\n", config.Interval)
		time.Sleep(time.Duration(config.Interval) * time.Minute)
		sendIPNotification(config.ClientName, config.BaseURL)
	}
}

func updateCloudflareRecord(ipv6 string) error {
	if config.CloudflareToken == "" || config.CloudflareZone == "" || config.CloudflareID == "" {
		return nil // 如果未配置Cloudflare，则跳过
	}

	baseURL := "https://api.cloudflare.com"
	if config.CloudflareBaseURL != "" {
		baseURL = config.CloudflareBaseURL
	}
	url := fmt.Sprintf("%s/client/v4/zones/%s/dns_records/%s",
		baseURL, config.CloudflareZone, config.CloudflareID)

	payload := map[string]interface{}{
		"type":    "AAAA",
		"name":    config.CloudflareDomain,
		"content": ipv6,
		"proxied": false,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("生成JSON数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("X-Auth-Email", config.CloudflareEmail)
	req.Header.Set("X-Auth-Key", config.CloudflareToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("更新DNS记录失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("更新DNS记录失败，状态码: %d", resp.StatusCode)
	}

	fmt.Printf("Cloudflare DNS记录已更新为: %s\n", ipv6)
	return nil
}
func sendIPNotification(clientName, baseURL string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("获取网络接口失败: %v\n", err)
		return
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("获取接口 %s 地址失败: %v\n", iface.Name, err)
			continue
		}

		if ipv6 := findIPv6(addrs); ipv6 != "" {
			// 使用配置文件中的 baseURL，如果参数中提供的 baseURL 为空
			targetURL := baseURL
			if targetURL == "" {
				targetURL = config.BaseURL
			}

			// 使用 url.QueryEscape 对消息进行编码
			message := url.QueryEscape(clientName + "更新通知/" + clientName + "已更新为:" + ipv6)
			resp, err := httpClient.Get(targetURL + message)
			if err != nil {
				fmt.Printf("发送通知失败: %v\n", err)
			} else {
				resp.Body.Close()
				fmt.Printf("%s 的 IP 更新通知已发送\n", clientName)
			}

			// 更新Cloudflare DNS记录
			if err := updateCloudflareRecord(ipv6); err != nil {
				fmt.Printf("更新Cloudflare DNS记录失败: %v\n", err)
			}

			return // 找到并发送后立即返回
		}
	}
}
func findIPv6(addrs []net.Addr) string {
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip != nil && ip.To16() != nil && ip.To4() == nil {
			// 检查是否为全局单播地址（不是内部地址）
			if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsPrivate() {
				return net.ParseIP(ip.String()).To16().String()
			}
		}
	}
	return ""
}

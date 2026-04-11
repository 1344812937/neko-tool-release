package service

import (
	"net"
	"net/url"
	"strings"
)

// ResolveWorkstationAddress 统一计算当前工作站应展示的地址。
func ResolveWorkstationAddress(configuredAddress, fallbackHost string) string {
	if configured := NormalizeAddressToken(configuredAddress); configured != "" {
		return configured
	}
	if detected := DetectLocalInterfaceAddress(); detected != "" {
		return detected
	}
	return NormalizeAddressToken(fallbackHost)
}

// NormalizeAddressToken 负责从 URL、host:port 或原始输入中提取主机部分。
func NormalizeAddressToken(value string) string {
	trimmed := strings.Trim(strings.TrimSpace(value), "\"")
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			trimmed = parsed.Host
		}
	}
	if strings.HasPrefix(trimmed, "[") {
		if endIndex := strings.Index(trimmed, "]"); endIndex > 0 {
			trimmed = trimmed[1:endIndex]
		}
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	} else {
		colonIndex := strings.LastIndex(trimmed, ":")
		if colonIndex > 0 && strings.Count(trimmed, ":") == 1 && strings.Contains(trimmed, ".") {
			trimmed = trimmed[:colonIndex]
		}
	}
	return strings.TrimSpace(trimmed)
}

// DetectLocalInterfaceAddress 返回首个可用的非回环 IPv4 地址。
func DetectLocalInterfaceAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, item := range interfaces {
		if item.Flags&net.FlagUp == 0 || item.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, addrErr := item.Addrs()
		if addrErr != nil {
			continue
		}
		for _, address := range addresses {
			switch value := address.(type) {
			case *net.IPNet:
				if ip := value.IP.To4(); ip != nil && !ip.IsLoopback() {
					return ip.String()
				}
			case *net.IPAddr:
				if ip := value.IP.To4(); ip != nil && !ip.IsLoopback() {
					return ip.String()
				}
			}
		}
	}
	return ""
}

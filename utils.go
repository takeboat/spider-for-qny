package spider

import (
	"bytes"
	"net"
	"strconv"
	"strings"
)

// 解码数据库存储的VPNIP
func ReverseLittleBinaryIp(ip uint32) string {
	ipStr := IpIntToString(int(ip))
	ipArr := strings.Split(ipStr, ".")
	rs := []string{}
	for i := len(ipArr) - 1; i >= 0; i-- {
		rs = append(rs, ipArr[i])
	}
	return strings.Join(rs, ".")
}

func IpIntToString(ipInt int) string {
	ipSegs := make([]string, 4)
	var len int = len(ipSegs)
	buffer := bytes.NewBufferString("")
	for i := 0; i < len; i++ {
		tempInt := ipInt & 0xFF
		ipSegs[len-i-1] = strconv.Itoa(tempInt)
		ipInt = ipInt >> 8
	}
	for i := 0; i < len; i++ {
		buffer.WriteString(ipSegs[i])
		if i < len-1 {
			buffer.WriteString(".")
		}
	}
	return buffer.String()
}

func IsValidIp(vpnIP string) bool {
	ip := net.ParseIP(vpnIP)
	if ip == nil {
		return false
	}

	// 拒绝IPv6（如果只需要IPv4）
	if ip.To4() == nil {
		return false
	}
	switch {
	case ip.IsLoopback(): // 拒绝回环地址 127.0.0.0/8
		return false
	case ip.Equal(net.IPv4zero): // 拒绝 0.0.0.0
		return false
	case ip.IsPrivate(): // 拒绝私有地址
		return false
	case ip.IsUnspecified(): // 拒绝未指定地址
		return false
	case ip.Equal(net.IPv4(169, 254, 0, 0)): // 拒绝链路本地地址
		return false
	case ip.IsMulticast(): // 拒绝多播地址
		return false
	default:
		return true // 接受公网IP
	}
}

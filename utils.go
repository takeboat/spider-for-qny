package spider

import (
	"bytes"
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

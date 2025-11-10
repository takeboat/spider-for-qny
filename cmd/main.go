package main

import (
	"fmt"
	"spider"
	"strings"
)

func main() {
	spider.MustInitDB()
	fmt.Println("Database initialized")
	m := make(map[ChargerStation][]Device)

	stations, err := GetChargerStations()
	if err != nil {
		fmt.Println("Error getting charger stations:", err)
		return
	}
	for _, station := range stations {
		fmt.Println("Charger station:")
		station.Print()
		devices, err := GetDevicesByStationID(station.StaionId)
		if err != nil {
			fmt.Println("Error getting devices:", err)
			continue
		}
		m[station] = devices
	}
	
}

type ChargerStation struct {
	StaionId    uint   `json:"station_id" gorm:"column:station_id"`
	StationName string `json:"station_name"`
	Province    string `json:"province"`
	City        string `json:"city"`
	County      string `json:"county"`
	StationAddr string `json:"station_addr" gorm:"column:station_addr"`
}
type Device struct {
	DeviceID    int    `json:"device_id"`
	DeviceVpnIP uint32 `json:"device_vpn_ip"`
	DeviceMac   string `json:"device_mac"`
	DeviceICCID string `json:"device_iccid"`
}

func GetChargerStations() ([]ChargerStation, error) {
	var stations []ChargerStation
	res := spider.GetDB().Raw("select ss.station_id, ss.station_name, ss.province, ss.city ,ss.county, ss.station_addr from sys_station ss where ss.is_deleted = 1").Scan(&stations)
	if res.Error != nil {
		return nil, res.Error
	}
	return stations, nil
}

func (cs *ChargerStation) Print() {
	fmt.Printf("充电站ID: %d, 充电站名称: %s, 省份: %s, 城市: %s, 区县: %s, 地址: %s\n", cs.StaionId, cs.StationName, cs.Province, cs.City, cs.County, cs.StationAddr)
}

func GetDevicesByStationID(stationID uint) ([]Device, error) {
	fmt.Println("Getting devices for station ID:", stationID)
	var devices []Device
	res := spider.GetDB().Raw("select cd.device_id, cd.device_vpn_ip from charger_device cd where cd.station_id = ? and cd.device_tcp_status = 1", stationID).Scan(&devices)
	if res.Error != nil {
		return nil, res.Error
	}
	return devices, nil
}

func (d *Device) GetDeviceNetWorkDetails() error {
	vpnIP := spider.ReverseLittleBinaryIp(d.DeviceVpnIP)
	host := fmt.Sprintf("%s:22", vpnIP)
	client, err := spider.InitSSH(host)
	if err != nil {
		return err
	}
	cmd := "cat /tmp/NET_details"
	output, err := spider.RunSSHCommand(client, cmd)
	if err != nil {
		return err
	}
	parts := strings.Split(output, ",")
	d.DeviceICCID = parts[2]          // 获取ICCID
	d.DeviceMac = parts[len(parts)-3] // 获取MAC地址
	return nil
}

func (d *Device) Print() {
	fmt.Printf("设备ID: %d, 设备VPN IP: %v 设备ICCID: %s 设备MAC: %s \n", d.DeviceID, spider.ReverseLittleBinaryIp(d.DeviceVpnIP), d.DeviceICCID, d.DeviceMac)
}

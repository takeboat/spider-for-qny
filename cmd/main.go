package main

import (
	"fmt"
	"spider"
	"strings"
	"sync"
)

var res = &Result{}

func main() {
	spider.MustInitDB()
	fmt.Println("Database initialized")
	m := make(map[ChargerStation][]Device)
	stations, err := GetChargerStations()
	if err != nil {
		fmt.Println("Error getting charger stations:", err)
		return
	}
	var wg sync.WaitGroup
	deviceChan := make(chan *Device, 500)
	workerCount := 10
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(deviceChan, &wg)
	}

	for _, station := range stations[:10] {
		devices, err := GetDevicesByStationID(station.StaionId)
		if err != nil {
			fmt.Println("Error getting devices:", err)
			res.WriteDeviceGetError(station.StaionId, err)
			continue
		}
		m[station] = devices
		for _, device := range devices {
			deviceChan <- &device
		}
	}
	close(deviceChan)
	wg.Wait()
	// Print results
	fmt.Println("done")
	for station, devices := range m {
		for _, device := range devices {
			device.Print()
		}
		station.Print()
	}
	fmt.Println(res)
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
	DeviceID    uint   `json:"device_id"`
	DeviceVpnIP uint32 `json:"device_vpn_ip"`
	DeviceMac   string `json:"device_mac"`
	DeviceICCID string `json:"device_iccid"`
}
type Result struct {
	DeviceGetError          string
	DeviceGetErrorTotal     int
	DeviceNetWorkError      string
	DeviceNetWorkErrorTotal int
}

func (r *Result) WriteDeviceGetError(stationID uint, err error) {
	r.DeviceGetError += fmt.Sprintf("设备ID: %d, 获取设备信息错误: %v\n", stationID, err)
	r.DeviceGetErrorTotal++
}
func (r *Result) WriteDeviceNetWorkError(deviceId uint, err error) {
	r.DeviceNetWorkError += fmt.Sprintf("设备ID: %d, 获取设备信息错误: %v\n", deviceId, err)
	r.DeviceNetWorkErrorTotal++
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
	if !spider.IsValidIp(vpnIP) {
		return fmt.Errorf("Invalid IP: %s Device_id: %d", vpnIP, d.DeviceID)
	}
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

func worker(devices chan *Device, wg *sync.WaitGroup) {
	defer wg.Done()
	for device := range devices {
		err := device.GetDeviceNetWorkDetails()
		if err != nil {
			fmt.Println("Error getting device network details:", err)
			res.WriteDeviceNetWorkError(device.DeviceID, err)
			continue
		}
		device.Print()
	}
}

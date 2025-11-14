package main

import (
	"fmt"
	"os"
	"spider"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gocarina/gocsv"
)

var res = &Result{}

var DeviceTotal atomic.Int32

func main() {
	spider.MustInitDB()
	// must init database connection
	fmt.Println("Database initialized")
	t := time.Now()
	m := make(map[ChargerStation][]Device)
	stations, err := GetChargerStations()
	if err != nil {
		fmt.Println("Error getting charger stations:", err)
		return
	}
	var wg sync.WaitGroup
	deviceChan := make(chan *Device, 500)
	workerCount := 15
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(deviceChan, &wg)
	}
	for _, station := range stations {
		if station.StaionId == 0 || station.StaionId == 1766 ||
			station.StaionId == 1674 || station.StaionId == 4933 {
			continue
		}
		devices, err := GetDevicesByStationID(station.StaionId)
		if err != nil {
			fmt.Println("Error getting devices:", err)
			res.WriteDeviceGetError(station.StaionId, err)
			continue
		}
		m[station] = devices
		for i := range devices {
			deviceChan <- &devices[i]
		}
	}
	close(deviceChan)
	wg.Wait()
	// Print results
	fmt.Println("done")
	fmt.Println("设备总数:", DeviceTotal.Load())
	res.WriteErrorToFile()
	// export to csv
	if err := exportToCSV(m); err != nil {
		fmt.Println("Error exporting to CSV:", err)
	}
	fmt.Println("Exported to CSV Success")
	fmt.Println("总共耗时:", time.Since(t))
}

type ChargerStation struct {
	StaionId        uint   `json:"station_id" gorm:"column:station_id"`
	StationName     string `json:"station_name"`
	Province        string `json:"province"`
	City            string `json:"city"`
	County          string `json:"county"`
	StationAddr     string `json:"station_addr" gorm:"column:station_addr"`
	StationContacts string `json:"station_contacts" gorm:"column:station_contacts"`
}
type Device struct {
	DeviceID        uint   `json:"device_id"`
	DeviceVpnIP     uint32 `json:"device_vpn_ip"`
	DeviceMac       string `json:"device_mac"`
	DeviceICCID     string `json:"device_iccid"`
	DeviceStationSn int    `json:"device_station_sn"`
	ProductID       uint32 `json:"product_id"`
}
type Result struct {
	mu                      sync.Mutex
	DeviceGetError          string
	DeviceGetErrorTotal     int
	DeviceNetWorkError      string
	DeviceNetWorkErrorTotal int
}

func (r *Result) WriteDeviceGetError(stationID uint, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.DeviceGetError += fmt.Sprintf("设备ID: %d, 获取设备信息错误: %v\n", stationID, err)
	r.DeviceGetErrorTotal++
}
func (r *Result) WriteDeviceNetWorkError(deviceId uint, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.DeviceNetWorkError += fmt.Sprintf("设备ID: %d, 获取设备信息错误: %v\n", deviceId, err)
	r.DeviceNetWorkErrorTotal++
}
func (*Result) WriteErrorToFile() error {
	if res.DeviceGetErrorTotal > 0 {
		deviceGetErrorFile, err := os.Create("device_get_errors.txt")
		if err != nil {
			return fmt.Errorf("创建设备获取错误文件失败: %v", err)
		}
		defer deviceGetErrorFile.Close()

		_, err = deviceGetErrorFile.WriteString(res.DeviceGetError)
		if err != nil {
			return fmt.Errorf("写入设备获取错误文件失败: %v", err)
		}
		fmt.Printf("设备获取错误已写入 device_get_errors.txt (共%d条)\n", res.DeviceGetErrorTotal)
	}
	if res.DeviceNetWorkErrorTotal > 0 {
		deviceNetWorkErrorFile, err := os.Create("device_network_errors.txt")
		if err != nil {
			return fmt.Errorf("创建设备网络错误文件失败: %v", err)
		}
		defer deviceNetWorkErrorFile.Close()

		_, err = deviceNetWorkErrorFile.WriteString(res.DeviceNetWorkError)
		if err != nil {
			return fmt.Errorf("写入设备网络错误文件失败: %v", err)
		}
		fmt.Printf("设备网络错误已写入 device_network_errors.txt (共%d条)\n", res.DeviceNetWorkErrorTotal)
	}
	return nil
}
func GetChargerStations() ([]ChargerStation, error) {
	var stations []ChargerStation
	res := spider.GetDB().Raw("select ss.station_id, ss.station_name, ss.province, ss.city ,ss.county, ss.station_addr, ss.station_contacts from sys_station ss where ss.is_deleted = 1").Scan(&stations)
	if res.Error != nil {
		return nil, res.Error
	}
	return stations, nil
}

func (cs *ChargerStation) Print() {
	fmt.Printf("充电站ID: %d, 充电站名称: %s, 省份: %s, 城市: %s, 区县: %s, 地址: %s 负责人: %s\n", cs.StaionId, cs.StationName, cs.Province, cs.City, cs.County, cs.StationAddr, cs.StationContacts)
}

func GetDevicesByStationID(stationID uint) ([]Device, error) {
	fmt.Println("Getting devices for station ID:", stationID)
	var devices []Device
	res := spider.GetDB().Raw("select cd.device_id, cd.device_vpn_ip, cd.device_station_sn, cd.product_id from charger_device cd where cd.station_id = ? and cd.device_tcp_status = 1", stationID).Scan(&devices)
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
		DeviceTotal.Add(1)
	}
}

type ExportData struct {
	StationName     string `csv:"场站名称"`
	Province        string `csv:"省份"`
	City            string `csv:"市区"`
	County          string `csv:"区/县"`
	StationAddr     string `csv:"详细地址"`
	StationContacts string `csv:"负责人"`
	DeviceMac       string `csv:"设备MAC地址"`
	DeviceICCID     string `csv:"设备ICCID"`
	DeviceStationSn string `csv:"设备场站序号"`
	DeviceVpnIP     string `csv:"vpnIP"`
	Product         string `csv:"产品型号"`
}

func exportToCSV(data map[ChargerStation][]Device) error {
	var exportData []ExportData
	// 转换数据格式
	for station, devices := range data {
		for _, device := range devices {
			if device.DeviceVpnIP == 0 {
				continue
			}
			exportData = append(exportData, ExportData{
				StationName:     station.StationName,
				Province:        station.Province,
				City:            station.City,
				County:          station.County,
				StationAddr:     station.StationAddr,
				StationContacts: station.StationContacts,
				DeviceMac:       device.DeviceMac,
				DeviceICCID:     device.DeviceICCID,
				DeviceStationSn: fmt.Sprintf("%d号桩", device.DeviceStationSn),
				DeviceVpnIP:     spider.ReverseLittleBinaryIp(device.DeviceVpnIP),
				Product:         spider.ProductStr(device.ProductID),
			})
		}
	}
	// 创建CSV文件
	filename := fmt.Sprintf("output_%s.csv", time.Now().Format("20060102_15-04-05"))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	// 写入CSV数据
	return gocsv.MarshalFile(&exportData, file)
}

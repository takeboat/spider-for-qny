package spider

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// JumpHost 跳板机连接结构
type JumpHost struct {
	client *ssh.Client
	host   string
	mutex  sync.Mutex
}

// NewJumpHost 创建一个新的跳板机连接
func NewJumpHost(host string, user string, password string) (*JumpHost, error) {
	client, err := initSSHWithPassword(host, user, password)
	if err != nil {
		return nil, fmt.Errorf("无法连接到跳板机 %s: %v", host, err)
	}
	
	return &JumpHost{
		client: client,
		host:   host,
	}, nil
}

// ConnectToTarget 通过跳板机连接到目标主机
func (j *JumpHost) ConnectToTarget(targetHost string) (*ssh.Client, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	
	if j.client == nil {
		return nil, fmt.Errorf("跳板机连接已关闭")
	}
	
	// 尝试使用预定义的密码连接目标主机
	passwords := []string{"123456", "ft0323"}
	for _, password := range passwords {
		config := &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}
		
		// 通过跳板机建立到目标主机的连接
		conn, err := j.client.Dial("tcp", targetHost)
		if err != nil {
			continue
		}
		
		ncc, chans, reqs, err := ssh.NewClientConn(conn, targetHost, config)
		if err != nil {
			conn.Close()
			continue
		}
		
		client := ssh.NewClient(ncc, chans, reqs)
		return client, nil
	}
	
	return nil, fmt.Errorf("通过跳板机 %s 连接目标主机 %s 失败: 所有密码都尝试失败", j.host, targetHost)
}

// Close 关闭跳板机连接
func (j *JumpHost) Close() error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	
	if j.client != nil {
		err := j.client.Close()
		j.client = nil
		return err
	}
	return nil
}

// IsConnected 检查跳板机是否连接
func (j *JumpHost) IsConnected() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	return j.client != nil
}

func InitSSH(host string) (client *ssh.Client, err error) {
	passwords := []string{"123456", "ft0323"}
	for _, password := range passwords {
		client, err = initSSHWithPassword(host, "root", password)
		if err == nil {
			return
		}
	}
	return nil, fmt.Errorf("failed to connect to %s: 所有密码都尝试失败 err: %v", host, err)
}

func initSSHWithPassword(host, user, password string) (client *ssh.Client, err error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	client, err = ssh.Dial("tcp", host, config)
	return
}

func RunSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %v", err)
	}
	return stdout.String(), nil
}
package userDb

import (
	"encoding/json"
	"netsvr/pkg/utils"
	"time"
)

type User struct {
	Id        int
	Name      string
	Password  string
	SessionId uint32
}

// ClientInfo 返回给客户的信息
type ClientInfo struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// NetSvrInfo 存储到网关的用户信息
type NetSvrInfo struct {
	Id             int
	Name           string
	LastUpdateTime time.Time
}

func (r *NetSvrInfo) Encode() string {
	b, _ := json.Marshal(*r)
	return string(b)
}

func ParseNetSvrInfo(netSvrInfo string) *NetSvrInfo {
	if netSvrInfo == "" {
		return nil
	}
	tmp := &NetSvrInfo{}
	err := json.Unmarshal(utils.StrToBytes(netSvrInfo), tmp)
	if err != nil {
		return nil
	}
	return tmp
}

// ToNetSvrInfo 返回需要存储到网关的信息
func (r User) ToNetSvrInfo() string {
	tmp := NetSvrInfo{
		Id:             r.Id,
		Name:           r.Name,
		LastUpdateTime: time.Now(),
	}
	return tmp.Encode()
}

// ToClientInfo 返回登录成功后给到客户端的信息
func (r User) ToClientInfo() ClientInfo {
	return ClientInfo{
		Id:   r.Id,
		Name: r.Name,
	}
}

type collect struct {
	idMap   map[int]*User
	nameMap map[string]*User
}

func (r *collect) Add(id int, name string, password string) {
	tmp := &User{id, name, password, 0}
	r.idMap[tmp.Id] = tmp
	r.nameMap[tmp.Name] = tmp
}

// GetUser 根据用户名字，查询一个用户
func (r *collect) GetUser(name string) *User {
	if ret, ok := r.nameMap[name]; ok {
		return &(*ret)
	}
	return nil
}

// SetSessionId 更新用户session id
func (r *collect) SetSessionId(id int, sessionId uint32) {
	if ret, ok := r.idMap[id]; ok {
		ret.SessionId = sessionId
	}
}

// GetSessionId 根据用户id查询用户的session
func (r *collect) GetSessionId(userId int) uint32 {
	if ret, ok := r.idMap[userId]; ok {
		return ret.SessionId
	}
	return 0
}

// Collect 模拟数据库信息
var Collect *collect

func init() {
	Collect = &collect{idMap: map[int]*User{}, nameMap: map[string]*User{}}
	Collect.Add(1, "刘备", "123456")
	Collect.Add(2, "关羽", "123456")
	Collect.Add(3, "张飞", "123456")
}

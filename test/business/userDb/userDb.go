package userDb

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	workerUtils "netsvr/test/business/utils"
	"time"
)

type User struct {
	Id       int
	Name     string
	Password string
	IsOnline bool
	Topics   []string
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
	err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(netSvrInfo), tmp)
	if err != nil {
		logging.Debug("Parse userDb.ParseNetSvrInfo error: %v", err)
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

func (r *collect) Add(id int, name string, password string, topics []string) {
	tmp := &User{id, name, password, false, []string{}}
	r.idMap[tmp.Id] = tmp
	r.nameMap[tmp.Name] = tmp
	for _, v := range topics {
		tmp.Topics = append(tmp.Topics, v)
	}
}

// GetUser 根据用户名字，查询一个用户
func (r *collect) GetUser(name string) *User {
	if ret, ok := r.nameMap[name]; ok {
		return &(*ret)
	}
	return nil
}

// GetUserById 根据用户id查询用户
func (r *collect) GetUserById(userId int) *User {
	if ret, ok := r.idMap[userId]; ok {
		return &(*ret)
	}
	return nil
}

func (r *collect) SetOnline(userId int, online bool) {
	if ret, ok := r.idMap[userId]; ok {
		ret.IsOnline = online
	}
}

// Collect 模拟数据库信息
var Collect *collect

func init() {
	Collect = &collect{idMap: map[int]*User{}, nameMap: map[string]*User{}}
	Collect.Add(1, "玄德", "123456", []string{"桃园结义", "小品频道"})
	Collect.Add(2, "云长", "123456", []string{"桃园结义", "相声频道"})
	Collect.Add(3, "翼德", "123456", []string{"桃园结义", "戏曲频道"})
	Collect.Add(4, "奉先", "123456", nil)
}

package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strconv"
	"sync"
	"time"
)

var (
	GameMap = make(map[string]*GameManager)
	GameNum = 0
)

type Chat struct {
	Pid      int
	Name     string
	Msg      string
	SendTime time.Time
}

type GameManager struct {
	GameId      string
	UuidStarter string
	GamePtr     *Game
	Changed     map[int]bool
	Ended       map[int]bool
	ChatList    []*Chat
	CreateTime  time.Time
	Started     bool
	ChangeLock  sync.RWMutex
}

// NewGameManager 创建新游戏管理器
func NewGameManager(gameId string) *GameManager {
	return &GameManager{
		GameId:      gameId,
		UuidStarter: uuid.New().String(),
		GamePtr:     NewGame(),
		Changed:     make(map[int]bool),
		Ended:       make(map[int]bool),
		ChatList:    make([]*Chat, 0),
		CreateTime:  time.Now(),
		Started:     false,
	}
}

// Poll 轮询游戏状态
func (m *GameManager) Poll(pid int) gin.H {
	for {
		m.ChangeLock.RLock()
		changed := m.Changed[pid]
		m.ChangeLock.RUnlock()
		// 若状态未改变则继续等待
		if !changed {
			time.Sleep(PollInterval * time.Millisecond)
		} else {
			break
		}
	}
	// 若游戏对于该玩家已结束则删除该玩家
	m.ChangeLock.Lock()
	if m.Ended[pid] {
		delete(m.Ended, pid)
		// 若所有玩家都已结束则删除游戏
		if len(m.Ended) == 0 {
			delete(GameMap, m.GameId)
		}
	}
	m.Changed[pid] = false
	m.ChangeLock.Unlock()

	res := make(gin.H)
	res["state"] = SerializeGame(m.GamePtr, pid)
	res["result"] = make(gin.H)
	res["chat"] = SerializeChatList(m.ChatList)

	return res
}

// JoinGame 加入游戏
func (m *GameManager) JoinGame() gin.H {
	num := m.GetPlayerNum()
	if num >= MaxPlayers {
		return gin.H{
			"error": "The game is full",
		}
	} else if m.Started {
		return gin.H{
			"error": "The game has already started",
		}
	}
	pid, uid := m.GamePtr.AddPlayer("Player " + strconv.Itoa(num+1))

	m.ChangeLock.Lock()
	m.Changed[pid] = false
	m.Ended[pid] = false
	m.ChangeLock.Unlock()

	m.ChangeStatus()
	return gin.H{
		"id":   pid,
		"uuid": uid,
	}
}

// WatchGame 观战游戏
func (m *GameManager) WatchGame() gin.H {
	pid, uid := m.GamePtr.AddSpectator()
	m.ChangeLock.Lock()
	m.Changed[pid] = false
	m.ChangeLock.Unlock()
	m.ChangeStatus()
	return gin.H{
		"id":   pid,
		"uuid": uid,
	}
}

// StartGame 开始游戏
func (m *GameManager) StartGame() gin.H {
	if m.GamePtr.StartGame() {
		m.Started = true
		m.ChangeStatus()
		return make(gin.H)
	}
	return gin.H{
		"error": "Cannot start the game",
	}
}

// Chat 发送消息
func (m *GameManager) Chat(pid int, msg string) gin.H {
	m.ChatList = append(m.ChatList, &Chat{
		Pid:      pid,
		Name:     m.GamePtr.Players[pid].Name,
		Msg:      msg,
		SendTime: time.Now(),
	})
	m.ChangeStatus()
	return gin.H{
		"state":  SerializeGame(m.GamePtr, pid),
		"result": make(gin.H),
		"chat":   SerializeChatList(m.ChatList),
	}
}

// ChangeStatus 修改状态
func (m *GameManager) ChangeStatus() {
	m.ChangeLock.Lock()
	defer m.ChangeLock.Unlock()
	if m.GamePtr.State == EndedState {
		for i := range m.Ended {
			m.Ended[i] = true
		}
	}
	for i := range m.Changed {
		m.Changed[i] = true
	}
}

// GetPlayerNum 获取玩家数量
func (m *GameManager) GetPlayerNum() int {
	return m.GamePtr.PlayerNum
}

func queryManager(pid int, playerUuid, gameId string) *GameManager {
	res, exists := GameMap[gameId]
	if !exists {
		return nil
	}
	// fmt.Println(res.GamePtr.Players)
	if pid < 0 || pid >= len(res.GamePtr.Players) || res.GamePtr.Players[pid] == nil {
		fmt.Printf("Players=%v, pid=%d\n", res.GamePtr.Players, pid)
		return nil
	} else if res.GamePtr.Players[pid].Uuid != playerUuid {
		return nil
	}
	return res
}

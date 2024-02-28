package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strconv"
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
	ChangedChan map[int]chan bool
}

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
		ChangedChan: make(map[int]chan bool),
	}
}

func (m *GameManager) Poll(pid int) gin.H {
	// 若状态未改变则继续等待
	for !m.Changed[pid] {
		time.Sleep(PollInterval * time.Millisecond)
	}
	// 若游戏对于该玩家已结束则删除该玩家
	if m.Ended[pid] {
		delete(m.Ended, pid)
		// 若所有玩家都已结束则删除游戏
		if len(m.Ended) == 0 {
			delete(GameMap, m.GameId)
		}
	}

	m.Changed[pid] = false

	res := make(gin.H)
	res["state"] = SerializeGame(m.GamePtr, pid)
	res["result"] = make(gin.H)
	res["chat"] = SerializeChatList(m.ChatList)

	return res
}

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
	m.Changed[pid] = false
	m.Ended[pid] = false
	m.doChange()
	return gin.H{
		"id":   pid,
		"uuid": uid,
	}
}

func (m *GameManager) WatchGame() gin.H {
	pid, uid := m.GamePtr.AddSpectator()
	m.Changed[pid] = false
	m.doChange()
	return gin.H{
		"id":   pid,
		"uuid": uid,
	}
}

func (m *GameManager) StartGame() gin.H {
	if m.GamePtr.StartGame() {
		m.Started = true
		m.doChange()
		return make(gin.H)
	}
	return gin.H{
		"error": "Cannot start the game",
	}
}

func (m *GameManager) Chat(pid int, msg string) gin.H {
	m.ChatList = append(m.ChatList, &Chat{
		Pid:      pid,
		Name:     m.GamePtr.Players[pid].Name,
		Msg:      msg,
		SendTime: time.Now(),
	})
	m.doChange()
	return gin.H{
		"state":  SerializeGame(m.GamePtr, pid),
		"result": make(gin.H),
		"chat":   SerializeChatList(m.ChatList),
	}
}

func (m *GameManager) GetPlayerNum() int {
	return m.GamePtr.PlayerNum
}

func (m *GameManager) doChange() {
	if m.GamePtr.State == EndedState {
		for i := range m.Ended {
			m.Ended[i] = true
		}
	}
	for i := range m.Changed {
		m.Changed[i] = true
	}
}

func ValidatePlayer(pid int, playerUuid, gameUuid string) *GameManager {
	res, exists := GameMap[gameUuid]
	if !exists {
		return nil
	}
	if pid < 0 || pid >= len(res.GamePtr.Players) {
		return nil
	} else if res.GamePtr.Players[pid].Uuid != playerUuid {
		return nil
	}
	return res
}

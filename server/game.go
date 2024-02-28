package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"time"
)

var (
	L1, L2, L3, Nobles = LoadCards()
)

type Game struct {
	Players        []*Player
	PlayerNum      int    `json:"-"`
	State          string `json:"-"`
	ActivePlayerId int
	SpectatorIndex int `json:"-"`
	Gems           map[string]int
	Golds          int
	Table          [][]*DevCard
	Piles          [][]*DevCard
	CardMap        map[string]*DevCard `json:"-"`
	Nobles         []*Noble
	LastRound      bool `json:"-"`
	Winner         *Player
	Records        []gin.H
	UpdatedTime    time.Time `json:"-"`
}

func NewGame() *Game {
	L1, L2, L3, Nobles = LoadCards()
	table := make([][]*DevCard, 3)
	for i := 0; i < 3; i++ {
		table[i] = make([]*DevCard, 0)
	}
	shuffleNobles(Nobles)
	cardMap := make(map[string]*DevCard, L1Num+L2Num+L3Num)
	for _, cards := range [][]*DevCard{L1, L2, L3} {
		for _, card := range cards {
			cardMap[card.Uuid] = card
		}
	}
	g := &Game{
		Players:        make([]*Player, MaxPlayers),
		PlayerNum:      0,
		State:          WaitingState,
		ActivePlayerId: -1,
		SpectatorIndex: MaxPlayers,
		Gems:           make(map[string]int),
		Golds:          TotalGolds,
		Table:          table,
		Piles:          [][]*DevCard{L1, L2, L3},
		CardMap:        cardMap,
		Nobles:         Nobles[:1],
		LastRound:      false,
		Winner:         nil,
		Records:        make([]gin.H, 0),
		UpdatedTime:    time.Now(),
	}
	return g
}

func (g *Game) AddPlayer(name string) (pid int, uuid string) {
	pid = g.PlayerNum
	player := NewPlayer(g, pid, name)
	// 由于 g.Players 至少有 4 个元素，所以不会越界
	g.Players[pid] = player
	g.PlayerNum++
	uuid = player.Uuid
	// 添加一个贵族
	g.Nobles = append(g.Nobles, Nobles[g.PlayerNum])
	return
}

func (g *Game) RenamePlayer(pid int, name string) {
	g.Players[pid].Name = name
}

func (g *Game) AddSpectator() (sid int, uuid string) {
	sid = g.SpectatorIndex
	spectator := NewPlayer(g, sid, fmt.Sprintf("Spec-%d", sid))
	g.Players = append(g.Players, spectator)
	g.SpectatorIndex++
	uuid = spectator.Uuid
	return
}

func (g *Game) StartGame() bool {
	if g.PlayerNum < 2 {
		return false
	}
	// 初始化所对应的宝石
	num := []int{4, 5, 7}[g.PlayerNum-2]
	for _, color := range ColorList {
		g.Gems[color] = num
	}
	// 处理发展卡
	shuffleCards(L1)
	shuffleCards(L2)
	shuffleCards(L3)
	g.Table[0] = L1[:TableSize]
	g.Table[1] = L2[:TableSize]
	g.Table[2] = L3[:TableSize]
	g.Piles[0] = L1[TableSize:]
	g.Piles[1] = L2[TableSize:]
	g.Piles[2] = L3[TableSize:]
	// 修改状态
	g.State = PlayingState
	g.NextTurn()
	return true
}

func (g *Game) NextTurn() {
	player := g.GetActivePlayer()
	if player != nil && player.Points >= WinPoints {
		g.LastRound = true
	}
	g.ActivePlayerId = (g.ActivePlayerId + 1) % g.PlayerNum
	// 如果已经结束
	if g.LastRound && g.ActivePlayerId == 0 {
		g.State = EndedState
		g.Winner = g.CreateWinner()
		g.ActivePlayerId = -1
	}
	g.GetActivePlayer().StartTurn()
}

func (g *Game) GetActivePlayer() *Player {
	index := g.ActivePlayerId
	if index < 0 || index >= g.PlayerNum {
		return nil
	}
	return g.Players[index]
}

func (g *Game) CreateWinner() *Player {
	if g.Winner != nil {
		return g.Winner
	}
	maxPoints := 0
	var winner *Player
	for _, p := range g.Players {
		points := p.Points
		if points >= maxPoints {
			maxPoints = points
			winner = p
		}
	}
	return winner
}

func (g *Game) Take(color string) gin.H {
	player := g.GetActivePlayer()
	info := player.Take(color)
	if info != "" {
		return gin.H{"error": info}
	}
	if player.Finished {
		g.NextTurn()
	}
	return nil
}

func (g *Game) Discard(color string) gin.H {
	player := g.GetActivePlayer()
	info := player.Discard(color)
	if info != "" {
		return gin.H{"error": info}
	}
	return nil
}

func (g *Game) Buy(uuid string) gin.H {
	player := g.GetActivePlayer()
	info := player.Buy(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	// 检查贵族
	nobles := player.CheckNobles()
	// 无贵族或一个贵族，直接结算并进入下一回合
	if len(nobles) <= 1 {
		if len(nobles) == 1 {
			player.doVisit(nobles[0])
		}
		return g.EndTurn()
	}
	// 有多个贵族，返回贵族信息
	uuids := make([]string, len(nobles))
	for i, n := range nobles {
		uuids[i] = n.Uuid
	}
	return gin.H{"nobles": uuids}
}

func (g *Game) Reserve(uuid string) gin.H {
	player := g.GetActivePlayer()
	info := player.Reserve(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	return nil
}

func (g *Game) VisitNoble(uuid string) gin.H {
	player := g.GetActivePlayer()
	info := player.VisitNoble(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	return g.EndTurn()
}

func (g *Game) EndTurn() gin.H {
	g.NextTurn()
	return nil
}

func (g *Game) FindCard(uuid string) *DevCard {
	return g.CardMap[uuid]
}

func (g *Game) RemoveCardFromTable(card *DevCard) bool {
	index := card.Level - 1
	uuid := card.Uuid
	for i, c := range g.Table[index] {
		if c.Uuid == uuid {
			if len(g.Piles[index]) > 0 {
				g.Table[index][i] = g.Piles[index][0]
				g.Piles[index] = g.Piles[index][1:]
			} else {
				g.Table[index][i] = nil
			}
			return true
		}
	}
	return false
}

func (g *Game) RemoveCardFromPiles(level int) *DevCard {
	index := level - 1
	if len(g.Piles[index]) == 0 {
		return nil
	}
	card := g.Piles[index][0]
	g.Piles[index] = g.Piles[index][1:]
	return card
}

func (g *Game) Log(msg string) {
	g.Records = append(g.Records, gin.H{
		"pid":  g.ActivePlayerId,
		"msg":  msg,
		"time": time.Now().Format("2006-01-02 15:04:05"),
	})
}

func shuffleCards(cards []*DevCard) {
	n := len(cards)
	for i := 0; i < n; i++ {
		j := i + rand.Intn(n-i)
		cards[i], cards[j] = cards[j], cards[i]
	}
}

func shuffleNobles(nobles []*Noble) {
	n := len(nobles)
	for i := 0; i < n; i++ {
		j := i + rand.Intn(n-i)
		nobles[i], nobles[j] = nobles[j], nobles[i]
	}
}

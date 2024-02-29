package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"time"
)

var (
	LoadedNobles []*Noble
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
	L1, L2, L3, loadedNobles := LoadCards()
	shuffleNobles(loadedNobles)
	LoadedNobles = loadedNobles

	table := make([][]*DevCard, 3)
	for i := 0; i < 3; i++ {
		table[i] = make([]*DevCard, 0)
	}
	piles := [][]*DevCard{L1, L2, L3}
	cardMap := make(map[string]*DevCard)
	for _, pile := range piles {
		for _, card := range pile {
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
		Piles:          piles,
		CardMap:        cardMap,
		Nobles:         loadedNobles[:1],
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
	g.Nobles = append(g.Nobles, LoadedNobles[g.PlayerNum])
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
	for i := 0; i < 3; i++ {
		shuffleCards(g.Piles[i])
		g.Table[i] = g.Piles[i][:TableSize]
		g.Piles[i] = g.Piles[i][TableSize:]
	}
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
		g.Winner = g.DetermineWinner()
		g.ActivePlayerId = -1
	} else {
		g.GetActivePlayer().StartTurn()
	}
}

func (g *Game) GetActivePlayer() *Player {
	index := g.ActivePlayerId
	if index < 0 || index >= g.PlayerNum {
		return nil
	}
	return g.Players[index]
}

func (g *Game) DetermineWinner() *Player {
	if g.Winner != nil {
		return g.Winner
	}
	maxPoints := 0
	var winner *Player
	for _, p := range g.Players[:g.PlayerNum] {
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
	info := player.TakeOne(color)
	if info != "" {
		return gin.H{"error": info}
	}
	if !player.Finished {
		return nil
	}
	log := fmt.Sprintf("%s takes %d gems: ", player.Name, valueSum(player.Taken))
	for c, n := range player.Taken {
		if n > 0 {
			log += fmt.Sprintf("%d%s ", n, ColorDict[c])
		}
	}
	g.Log(log)
	return g.CheckingNobleNextTurn()
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
	return g.CheckingNobleNextTurn()
}

func (g *Game) Reserve(uuid string) gin.H {
	player := g.GetActivePlayer()
	info := player.Reserve(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	return g.CheckingNobleNextTurn()
}

func (g *Game) VisitNoble(uuid string) gin.H {
	player := g.GetActivePlayer()
	info := player.VisitNoble(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	// 访问成功则立即结束回合
	g.NextTurn()
	return nil
}

func (g *Game) FindCard(uuid string) *DevCard {
	return g.CardMap[uuid]
}

func (g *Game) RemoveCardFromTable(card *DevCard) bool {
	l := card.Level - 1
	uuid := card.Uuid
	for i, c := range g.Table[l] {
		if c.Uuid == uuid {
			if len(g.Piles[l]) > 0 {
				g.Table[l][i] = g.Piles[l][0]
				g.Piles[l] = g.Piles[l][1:]
			} else {
				g.Table[l] = append(g.Table[l][:i], g.Table[l][i+1:]...)
			}
			return true
		}
	}
	return false
}

func (g *Game) RemoveCardFromPiles(level int) *DevCard {
	l := level - 1
	if len(g.Piles[l]) == 0 {
		return nil
	}
	card := g.Piles[l][0]
	g.Piles[l] = g.Piles[l][1:]
	return card
}

func (g *Game) Log(msg string) {
	g.Records = append(g.Records, gin.H{
		"pid":  g.ActivePlayerId,
		"msg":  msg,
		"time": time.Now().Format("2006-01-02 15:04:05"),
	})
}

func (g *Game) CheckingNobleNextTurn() gin.H {
	player := g.GetActivePlayer()
	// 检查贵族
	nobles := player.CheckNobles()
	// 有多个贵族，返回贵族信息
	if len(nobles) > 1 {
		uuids := make([]string, len(nobles))
		for i, n := range nobles {
			uuids[i] = n.Uuid
		}
		return gin.H{"nobles": uuids}
	}
	// 有一个贵族则自动访问
	if len(nobles) == 1 {
		player.doVisit(nobles[0])
	}
	// 轮到下一个玩家
	g.NextTurn()
	return nil
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

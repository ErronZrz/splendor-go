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

// NewGame 创建新游戏
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

// AddPlayer 添加玩家并返回其编号和 UUID
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

// RenamePlayer 重命名玩家
func (g *Game) RenamePlayer(pid int, name string) {
	g.Players[pid].Name = name
}

// AddSpectator 添加观众并返回其编号和 UUID
func (g *Game) AddSpectator() (sid int, uuid string) {
	sid = g.SpectatorIndex
	spectator := NewPlayer(g, sid, fmt.Sprintf("Spec-%d", sid))
	g.Players = append(g.Players, spectator)
	g.SpectatorIndex++
	uuid = spectator.Uuid
	return
}

// StartGame 开始游戏
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

// Take 宝石拿取
func (g *Game) Take(color string) gin.H {
	player := g.getActivePlayer()
	info := player.TakeOne(color)
	if info != "" {
		return gin.H{"error": info}
	}
	if !player.Finished {
		return nil
	}
	return g.NextTurn()
}

// Discard 宝石丢弃
func (g *Game) Discard(color string) gin.H {
	player := g.getActivePlayer()
	info := player.Discard(color)
	if info != "" {
		return gin.H{"error": info}
	}
	return nil
}

// Buy 购买发展卡
func (g *Game) Buy(uuid string) gin.H {
	player := g.getActivePlayer()
	info := player.Buy(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	return g.NextTurn()
}

// Reserve 预定发展卡
func (g *Game) Reserve(uuid string) gin.H {
	player := g.getActivePlayer()
	info := player.Reserve(uuid)
	if info != "" {
		return gin.H{"error": info}
	}
	return g.NextTurn()
}

// VisitNobleActively 主动访问贵族
func (g *Game) VisitNobleActively(uuid string) gin.H {
	player := g.getActivePlayer()
	nobles := player.CheckNobles()
	for _, n := range nobles {
		if n.Uuid == uuid {
			player.DoVisit(n)
			return g.NextTurn()
		}
	}
	return gin.H{
		"error": "You can't visit this noble",
	}
}

// NextTurn 下一个回合
func (g *Game) NextTurn() gin.H {
	player := g.getActivePlayer()
	// 如果本回合拿过宝石则记录
	logTakenGems(player)
	// 检查贵族，如果有多个贵族则暂不跳过回合，否则结束回合
	nobles := g.checkingNobleAndAutoVisit()
	if nobles != nil {
		return nobles
	}
	// 检查是否触发最后一回合
	if player != nil && player.Points >= WinPoints {
		g.LastRound = true
	}
	// 下一个玩家
	g.ActivePlayerId = (g.ActivePlayerId + 1) % g.PlayerNum
	// 如果已经结束
	if g.LastRound && g.ActivePlayerId == 0 {
		g.State = EndedState
		g.Winner = g.determineWinner()
		g.ActivePlayerId = -1
	} else {
		g.getActivePlayer().StartTurn()
	}
	return nil
}

// Log 记录日志
func (g *Game) Log(msg string) {
	g.Records = append(g.Records, gin.H{
		"pid":  g.ActivePlayerId,
		"msg":  msg,
		"time": time.Now().Format("2006-01-02 15:04:05"),
	})
}

func (g *Game) checkingNobleAndAutoVisit() gin.H {
	player := g.getActivePlayer()
	if player == nil || player.Visited {
		return nil
	}
	// 检查贵族
	nobles := player.CheckNobles()
	// 有多个贵族，返回贵族信息，暂时不跳过回合
	if len(nobles) > 1 {
		uuids := make([]string, len(nobles))
		for i, n := range nobles {
			uuids[i] = n.Uuid
		}
		return gin.H{"nobles": uuids}
	}
	// 有一个贵族则自动访问
	if len(nobles) == 1 {
		player.DoVisit(nobles[0])
	}
	return nil
}

func (g *Game) getActivePlayer() *Player {
	index := g.ActivePlayerId
	if index < 0 || index >= g.PlayerNum {
		return nil
	}
	return g.Players[index]
}

func (g *Game) determineWinner() *Player {
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

func logTakenGems(p *Player) {
	if p == nil {
		return
	}
	sum := p.TakenNum()
	if sum == 0 {
		return
	}
	log := fmt.Sprintf("%s takes %d gems: ", p.Name, sum)
	for c, n := range p.Taken {
		if n > 0 {
			log += fmt.Sprintf("%d%s ", n, ColorDict[c])
		}
	}
	// 此处清空 Taken 是为了防止一回合中重复记录，且由于 Finished=true，玩家无法继续拿宝石
	p.Taken = make(map[string]int)
	p.Game.Log(log)
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

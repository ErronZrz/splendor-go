package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Game struct {
	Players           []*Player
	PlayerNum         int    `json:"-"`
	State             string `json:"-"`
	ActivePlayerIndex int
	SpectatorIndex    int `json:"-"`
	Gems              map[string]int
	Golds             int
	Table             [][]*DevCard
	Piles             [][]*DevCard
	CardMap           map[string]*DevCard `json:"-"`
	Nobles            []*Noble
	LastRound         bool `json:"-"`
	Winner            *Player
	Records           []string
	UpdatedTime       time.Time `json:"-"`
}

func NewGame() *Game {
	l1, l2, l3, nobles := LoadCards()
	shuffleCards(l1)
	shuffleCards(l2)
	shuffleCards(l3)
	cardMap := make(map[string]*DevCard, L1Num+L2Num+L3Num)
	for _, cards := range [][]*DevCard{l1, l2, l3} {
		for _, card := range cards {
			cardMap[card.Uuid] = card
		}
	}
	shuffleNobles(nobles)
	g := &Game{
		Players:           make([]*Player, MaxPlayers),
		PlayerNum:         0,
		State:             WaitingState,
		ActivePlayerIndex: -1,
		SpectatorIndex:    MaxPlayers,
		Gems:              make(map[string]int),
		Golds:             TotalGolds,
		Table:             [][]*DevCard{l1[:TableSize], l2[:TableSize], l3[:TableSize]},
		Piles:             [][]*DevCard{l1[TableSize:], l2[TableSize:], l3[TableSize:]},
		CardMap:           cardMap,
		Nobles:            nobles,
		LastRound:         false,
		Winner:            nil,
		Records:           make([]string, 0),
		UpdatedTime:       time.Now(),
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
	// 选择贵族
	g.selectNobles()
	g.State = PlayingState
	g.NextTurn()
	return true
}

func (g *Game) NextTurn() {
	if g.GetActivePlayer().Points >= WinPoints {
		g.LastRound = true
	}
	g.ActivePlayerIndex = (g.ActivePlayerIndex + 1) % g.PlayerNum
	// 如果已经结束
	if g.LastRound && g.ActivePlayerIndex == 0 {
		g.State = EndedState
		g.Winner = g.GetWinner()
		g.ActivePlayerIndex = -1
	}
	g.GetActivePlayer().StartTurn()
}

func (g *Game) GetActivePlayer() *Player {
	index := g.ActivePlayerIndex
	if index < 0 || index >= g.PlayerNum {
		return nil
	}
	return g.Players[index]
}

func (g *Game) GetWinner() *Player {
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

func (g *Game) Take(color string) map[string][]string {
	player := g.GetActivePlayer()
	info := player.Take(color)
	if info != "" {
		return map[string][]string{"error": {info}}
	}
	if player.Finished {
		g.NextTurn()
	}
	return nil
}

func (g *Game) Discard(color string) map[string][]string {
	player := g.GetActivePlayer()
	info := player.Discard(color)
	if info != "" {
		return map[string][]string{"error": {info}}
	}
	return nil
}

func (g *Game) Buy(uuid string) map[string][]string {
	player := g.GetActivePlayer()
	info := player.Buy(uuid)
	if info != "" {
		return map[string][]string{"error": {info}}
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
	return map[string][]string{"nobles": uuids}
}

func (g *Game) Reserve(uuid string) map[string][]string {
	player := g.GetActivePlayer()
	info := player.Reserve(uuid)
	if info != "" {
		return map[string][]string{"error": {info}}
	}
	return nil
}

func (g *Game) VisitNoble(uuid string) map[string][]string {
	player := g.GetActivePlayer()
	info := player.VisitNoble(uuid)
	if info != "" {
		return map[string][]string{"error": {info}}
	}
	return g.EndTurn()
}

func (g *Game) EndTurn() map[string][]string {
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

func (g *Game) Log(s string) {
	g.Records = append(g.Records, s)
}

func (g *Game) selectNobles() {
	shuffleNobles(g.Nobles)
	g.Nobles = g.Nobles[:len(g.Players)+1]
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

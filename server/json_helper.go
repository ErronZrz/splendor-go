package main

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

var (
	ColorMap = map[string]string{
		"W": "w",
		"B": "u",
		"G": "g",
		"R": "r",
		"K": "b",
	}
	GoldKey = "*"
)

func SerializeDevCard(d *DevCard) gin.H {
	return gin.H{
		"uuid":   d.Uuid,
		"color":  ColorMap[d.Color],
		"points": d.Points,
		"cost":   transformMapColors(d.Cost),
		"level":  "level" + strconv.Itoa(d.Level),
	}
}

func SerializeHiddenDevCard(d *DevCard) gin.H {
	return gin.H{
		"uuid":  d.Uuid,
		"level": d.Level,
	}
}

func SerializeNoble(n *Noble) gin.H {
	return gin.H{
		"uuid":        n.Uuid,
		"id":          n.Sequence,
		"points":      NoblePoints,
		"requirement": transformMapColors(n.Cost),
	}
}

func SerializePlayer(p *Player, hide bool) gin.H {
	// 处理宝石数量
	gems := transformMapColors(p.Gems)
	gems[GoldKey] = p.Golds
	// 处理已购买的发展卡
	cards := make(map[string][]gin.H)
	for c, cardSlice := range p.Cards {
		color := ColorMap[c]
		cards[color] = make([]gin.H, len(cardSlice))
		for i, card := range cardSlice {
			cards[color][i] = SerializeDevCard(card)
		}
	}
	// 处理已访问的贵族
	nobles := make([]gin.H, len(p.Nobles))
	for i, n := range p.Nobles {
		nobles[i] = SerializeNoble(n)
	}
	// 处理已预购的发展卡
	reserved := make([]gin.H, len(p.Reserved))
	for i, card := range p.Reserved {
		if hide {
			reserved[i] = SerializeHiddenDevCard(card)
		} else {
			reserved[i] = SerializeDevCard(card)
		}
	}
	return gin.H{
		"uuid":     p.Uuid,
		"id":       p.Id,
		"name":     p.Name,
		"gems":     gems,
		"cards":    cards,
		"nobles":   nobles,
		"reserved": reserved,
		"score":    p.Points,
	}
}

func SerializeGame(g *Game, pid int) gin.H {
	// 处理玩家
	players := make([]gin.H, g.PlayerNum)
	for i, p := range g.Players[:g.PlayerNum] {
		players[i] = SerializePlayer(p, i != pid)
	}
	// 处理宝石数量
	gems := transformMapColors(g.Gems)
	gems[GoldKey] = g.Golds
	// 处理发展卡
	table := make(gin.H)
	piles := make(gin.H)
	for level := 1; level <= 3; level++ {
		levelStr := "level" + strconv.Itoa(level)
		// 处理公开的发展卡
		cards := make([]gin.H, len(g.Table[level-1]))
		for i, card := range g.Table[level-1] {
			if card != nil {
				cards[i] = SerializeDevCard(card)
			}
		}
		table[levelStr] = cards
		// 处理牌堆
		piles[levelStr] = len(g.Piles[level-1])
	}
	// 处理贵族
	nobles := make([]gin.H, len(g.Nobles))
	for i, n := range g.Nobles {
		nobles[i] = SerializeNoble(n)
	}
	// 处理赢家
	var winnerId *int
	if g.Winner != nil {
		winnerId = &g.Winner.Id
	}

	return gin.H{
		"players": players,
		"gems":    gems,
		"cards":   table,
		"decks":   piles,
		"nobles":  nobles,
		"log":     g.Records,
		"winner":  winnerId,
		"turn":    g.ActivePlayerId,
	}
}

func SerializeGameManager(m *GameManager) gin.H {
	return gin.H{
		"uuid":        m.GameId,
		"n_players":   m.GetPlayerNum(),
		"in_progress": m.Started,
	}
}

func SerializeChatList(chatList []*Chat) []gin.H {
	result := make([]gin.H, len(chatList))
	for i, chat := range chatList {
		result[i] = gin.H{
			"pid":  chat.Pid,
			"name": chat.Name,
			"msg":  chat.Msg,
			"time": chat.SendTime.Format("2006-01-02 15:04:05"),
		}
	}
	return result
}

func transformMapColors(m map[string]int) map[string]int {
	result := make(map[string]int)
	for color, count := range m {
		result[ColorMap[color]] = count
	}
	return result
}

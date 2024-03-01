package main

import (
	"fmt"
	"github.com/google/uuid"
	"strings"
)

type Player struct {
	Id       int
	Name     string
	Uuid     string
	Game     *Game `json:"-"`
	Gems     map[string]int
	Golds    int
	Cards    map[string][]*DevCard
	Reserved []*DevCard
	Nobles   []*Noble
	Points   int
	Taken    map[string]int `json:"-"`
	Visited  bool           `json:"-"`
	Finished bool           `json:"-"`
}

// NewPlayer 创建新玩家
func NewPlayer(game *Game, id int, name string) *Player {
	gems := make(map[string]int)
	cards := make(map[string][]*DevCard)
	for _, c := range ColorList {
		gems[c] = 0
		cards[c] = make([]*DevCard, 0)
	}

	return &Player{
		Id:       id,
		Name:     name,
		Uuid:     uuid.New().String(),
		Game:     game,
		Gems:     gems,
		Cards:    cards,
		Reserved: make([]*DevCard, 0),
		Nobles:   make([]*Noble, 0),
		Points:   0,
		Taken:    make(map[string]int),
		Visited:  false,
		Finished: false,
	}
}

// TakeOne 拿取宝石
func (p *Player) TakeOne(color string) string {
	if p.Finished {
		return "You have already acted"
	} else if p.totalGems() >= MaxGems {
		return "You already have 10 gems"
	} else if color == GoldKey {
		return "You can't take a 🟡"
	} else if p.Game.Gems[color] == 0 {
		return fmt.Sprintf("No %s left", ColorDict[color])
	} else if p.Taken[color] == 1 && p.TakenNum() == 2 {
		return "You have already taken 2 different gems"
	} else if p.Taken[color] == 1 && p.Game.Gems[color] < 3 {
		return fmt.Sprintf("There are not enough %s left", ColorDict[color])
	}
	p.Game.Gems[color]--
	p.Gems[color]++
	p.Taken[color]++
	if p.TakenNum() == 3 || p.Taken[color] == 2 {
		p.Finished = true
	}
	return ""
}

// Discard 丢弃宝石
func (p *Player) Discard(color string) string {
	if color == GoldKey {
		if p.Golds == 0 {
			return "You don't have any 🟡"
		}
		p.Golds--
		p.Game.Golds++
		return ""
	} else if p.Gems[color] == 0 {
		return fmt.Sprintf("You don't have any %s", ColorDict[color])
	}
	p.Gems[color]--
	p.Game.Gems[color]++
	if p.Taken[color] > 0 {
		p.Taken[color]--
	} else {
		p.Game.Log(fmt.Sprintf("%s discards 1%s", p.Name, ColorDict[color]))
	}
	return ""
}

// Buy 购买卡牌
func (p *Player) Buy(uuid string) string {
	if p.Finished {
		return "You have already acted"
	} else if p.TakenNum() > 0 {
		return "You have already taken gems"
	}
	card := p.findCard(uuid)
	pay := make(map[string]int)
	var goldNeeded int
	// 计算需要支付的宝石
	for _, c := range ColorList {
		if card.Cost[c] > p.powerOf(c) {
			goldNeeded += card.Cost[c] - p.powerOf(c)
			if goldNeeded > p.Golds {
				return "Not enough gems"
			}
		}
		pay[c] = card.Cost[c] - len(p.Cards[c])
	}
	log := fmt.Sprintf("%s buys", p.Name)
	// 移除卡牌
	if p.removeCardAfterBuying(card) {
		log += " reserved"
	}
	// 添加卡牌
	p.Cards[card.Color] = append(p.Cards[card.Color], card)
	log += fmt.Sprintf(": %s, paying ", card.Caption)
	// 支付宝石
	for c, num := range pay {
		if num <= 0 {
			continue
		}
		if p.Gems[c] < num {
			num = p.Gems[c]
		}
		p.Gems[c] -= num
		p.Game.Gems[c] += num
		log += fmt.Sprintf("%d%s", num, ColorDict[c])
	}
	// 支付黄金
	p.Golds -= goldNeeded
	p.Game.Golds += goldNeeded
	if goldNeeded > 0 {
		log += fmt.Sprintf("%d🟡", goldNeeded)
	}
	if log[len(log)-1] == ' ' {
		log += "nothing"
	}
	// 修改分数
	p.Points += card.Points
	// 记录日志
	p.Game.Log(log)
	p.Finished = true
	return ""
}

// Reserve 预购卡牌
func (p *Player) Reserve(uuid string) string {
	if p.Finished {
		return "You have already acted"
	} else if p.TakenNum() > 0 {
		return "You have already taken gems"
	} else if len(p.Reserved) >= MaxReserve {
		return "You have already reserved 3 cards"
	} else if p.totalGems() >= MaxGems && p.Game.Golds > 0 {
		return "Discard a gem first"
	}
	var card *DevCard
	var log string
	// 首先检查是否是牌堆中的牌
	if strings.Contains(uuid, "level") {
		level := int(uuid[len(uuid)-1] - '0')
		card = p.takeCardFromPile(level)
		if card == nil {
			return fmt.Sprintf("No card left in level %d", level)
		}
		log = fmt.Sprintf("%s reserves a card of level %d", p.Name, level)
	} else {
		// 否则检查是否是桌上的牌
		card = p.findCard(uuid)
		if !p.removeCardFromTable(card) {
			return "This card is not available"
		}
		log = fmt.Sprintf("%s reserves: %s", p.Name, card.Caption)
	}
	p.Reserved = append(p.Reserved, card)
	// 获取黄金
	if p.Game.Golds > 0 {
		p.Golds++
		p.Game.Golds--
		log += ", getting 1🟡"
	}
	p.Game.Log(log)
	p.Finished = true
	return ""
}

// CheckNobles 检查能够访问的贵族
func (p *Player) CheckNobles() []*Noble {
	var nobles []*Noble
	for _, n := range p.Game.Nobles {
		able := true
		for c, v := range n.Cost {
			if len(p.Cards[c]) < v {
				able = false
				break
			}
		}
		if able {
			nobles = append(nobles, n)
		}
	}
	return nobles
}

// DoVisit 执行访问贵族
func (p *Player) DoVisit(noble *Noble) {
	// 移除贵族
	for i, v := range p.Game.Nobles {
		if v.Uuid == noble.Uuid {
			p.Game.Nobles = append(p.Game.Nobles[:i], p.Game.Nobles[i+1:]...)
			break
		}
	}
	// 添加贵族
	p.Nobles = append(p.Nobles, noble)
	p.Game.Log(fmt.Sprintf("%s visits a noble: %s", p.Name, noble.Caption))
	// 修改分数
	p.Points += NoblePoints
	// 标记已访问
	p.Visited = true
}

// StartTurn 开始回合
func (p *Player) StartTurn() {
	p.Finished = false
	p.Visited = false
	p.Taken = make(map[string]int)
}

// TakenNum 返回玩家在本回合已拿取的宝石数量
func (p *Player) TakenNum() int {
	return valueSum(p.Taken)
}

func (p *Player) powerOf(color string) int {
	return len(p.Cards[color]) + p.Gems[color]
}

func (p *Player) totalGems() int {
	return valueSum(p.Gems) + p.Golds
}

func (p *Player) findCard(uuid string) *DevCard {
	return p.Game.CardMap[uuid]
}

func (p *Player) removeCardAfterBuying(card *DevCard) (reserved bool) {
	for i, c := range p.Reserved {
		if c.Uuid == card.Uuid {
			p.Reserved = append(p.Reserved[:i], p.Reserved[i+1:]...)
			return true
		}
	}
	p.removeCardFromTable(card)
	return false
}

func (p *Player) removeCardFromTable(card *DevCard) bool {
	g := p.Game
	l := card.Level - 1
	uid := card.Uuid
	for i, c := range g.Table[l] {
		if c.Uuid == uid {
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

func (p *Player) takeCardFromPile(level int) *DevCard {
	g := p.Game
	l := level - 1
	if len(g.Piles[l]) == 0 {
		return nil
	}
	card := g.Piles[l][0]
	g.Piles[l] = g.Piles[l][1:]
	return card
}

func valueSum(m map[string]int) int {
	var sum int
	for _, v := range m {
		sum += v
	}
	return sum
}

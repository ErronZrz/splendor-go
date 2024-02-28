package main

import (
	"fmt"
	"github.com/google/uuid"
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

// Power 计算玩家某颜色的购买力
func (p *Player) Power(color string) int {
	return len(p.Cards[color]) + p.Gems[color]
}

// TotalGems 计算玩家宝石总数
func (p *Player) TotalGems() int {
	return valueSum(p.Gems) + p.Golds
}

// Take 获取宝石
func (p *Player) Take(color string) string {
	if p.Finished {
		return "You have already acted"
	} else if p.TotalGems() >= MaxGems {
		return "You already have 10 gems"
	} else if color == GoldKey {
		return "You can't take a 🟡"
	} else if p.Game.Gems[color] == 0 {
		return fmt.Sprintf("No %s left", ColorDict[color])
	} else if p.Taken[color] == 1 && p.Game.Gems[color] < 3 {
		return fmt.Sprintf("There are not enough %s left", ColorDict[color])
	} else if p.Taken[color] == 1 && valueSum(p.Taken) == 2 {
		return "You have already taken 2 different gems"
	}
	p.Game.Gems[color]--
	p.Gems[color]++
	p.Taken[color]++
	p.Game.Log(fmt.Sprintf("%s takes a %s", p.Name, ColorDict[color]))
	if valueSum(p.Taken) == 3 || p.Taken[color] == 2 {
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
	p.Taken[color] = max(0, p.Taken[color]-1)
	return ""
}

// Buy 购买卡牌
func (p *Player) Buy(uuid string) string {
	if p.Finished {
		return "You have already acted"
	} else if valueSum(p.Taken) > 0 {
		return "You have already taken gems"
	}
	card := p.Game.FindCard(uuid)
	pay := make(map[string]int)
	var goldNeeded int
	// 计算需要支付的宝石
	for _, c := range ColorList {
		if card.Cost[c] > p.Power(c) {
			goldNeeded += card.Cost[c] - p.Power(c)
			if goldNeeded > p.Golds {
				return "Not enough gems"
			}
		}
		pay[c] = card.Cost[c] - len(p.Cards[c])
	}
	// 移除卡牌
	p.removeAfterBuying(card)
	// 添加卡牌
	p.Cards[card.Color] = append(p.Cards[card.Color], card)
	p.Game.Log(fmt.Sprintf("%s buys a card %s", p.Name, card.Caption))
	// 支付宝石
	for c, v := range pay {
		actual := min(p.Gems[c], max(0, v))
		p.Gems[c] -= actual
		p.Game.Gems[c] += actual
		if actual > 0 {
			p.Game.Log(fmt.Sprintf("%s pays %d %s", p.Name, actual, ColorDict[c]))
		}
	}
	// 支付黄金
	p.Golds -= goldNeeded
	p.Game.Golds += goldNeeded
	if goldNeeded > 0 {
		p.Game.Log(fmt.Sprintf("%s pays %d 🟡", p.Name, goldNeeded))
	}
	// 修改分数
	p.Points += card.Points
	return ""
}

// Reserve 预购卡牌
func (p *Player) Reserve(uuid string) string {
	if p.Finished {
		return "You have already acted"
	} else if valueSum(p.Taken) > 0 {
		return "You have already taken gems"

	} else if len(p.Reserved) >= MaxReserve {
		return "You have already reserved 3 cards"

	} else if p.TotalGems() >= MaxGems && p.Game.Golds > 0 {
		return "Discard a gem first"
	}
	// 预购卡牌
	card := p.Game.FindCard(uuid)
	if !p.Game.RemoveCardFromTable(card) {
		card = p.Game.RemoveCardFromPiles(card.Level)
		if card == nil {
			return fmt.Sprintf("No card left in level %d", card.Level)
		}
	}
	p.Reserved = append(p.Reserved, card)
	p.Game.Log(fmt.Sprintf("%s reserves a card %s", p.Name, card.Caption))
	// 获取黄金
	if p.Game.Golds > 0 {
		p.Golds++
		p.Game.Golds--
		p.Game.Log(fmt.Sprintf("%s takes a 🟡", p.Name))
	}
	return ""
}

// VisitNoble 贵族访问
func (p *Player) VisitNoble(uuid string) string {
	if p.Visited {
		return "You have already visited a noble"
	}
	nobles := p.CheckNobles()
	for _, n := range nobles {
		if n.Uuid == uuid {
			p.doVisit(n)
			return ""
		}
	}
	return "You can't visit this noble"
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

// StartTurn 开始回合
func (p *Player) StartTurn() {
	p.Visited = false
	p.Finished = false
	p.Taken = make(map[string]int)
}

func (p *Player) removeAfterBuying(card *DevCard) {
	for i, c := range p.Reserved {
		if c.Uuid == card.Uuid {
			p.Reserved = append(p.Reserved[:i], p.Reserved[i+1:]...)
			return
		}
	}
	p.Game.RemoveCardFromTable(card)
}

func (p *Player) doVisit(noble *Noble) {
	p.Visited = true
	// 移除贵族
	for i, v := range p.Game.Nobles {
		if v.Uuid == noble.Uuid {
			p.Game.Nobles = append(p.Game.Nobles[:i], p.Game.Nobles[i+1:]...)
			break
		}
	}
	// 添加贵族
	p.Nobles = append(p.Nobles, noble)
	p.Game.Log(fmt.Sprintf("%s visits a noble %s", p.Name, noble.Caption))
	// 修改分数
	p.Points += NoblePoints
}

func valueSum(m map[string]int) int {
	var sum int
	for _, v := range m {
		sum += v
	}
	return sum
}

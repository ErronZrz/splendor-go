package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"os"
	"strings"
)

const (
	MaxPlayers        = 4
	NoblePoints       = 3
	MaxGems           = 10
	MaxReserve        = 3
	TotalGolds        = 5
	WinPoints         = 15
	L1Num             = 40
	L2Num             = 30
	L3Num             = 20
	NobleNum          = 10
	TableSize         = 4
	PollInterval      = 400
	DeleteWaitingGame = 10
	DeletePlayingGame = 24
	WaitingState      = "waiting"
	PlayingState      = "playing"
	EndedState        = "ended"
)

var (
	ColorList = []string{"W", "B", "G", "R", "K"}

	ColorDict = map[string]string{
		"W": "⚪",
		"B": "🔵",
		"G": "🟢",
		"R": "🔴",
		"K": "⚫",
	}

	SuggestWords = make([]string, 0)
)

type DevCard struct {
	Uuid    string
	Level   int
	Color   string
	Points  int
	Cost    map[string]int
	Caption string
}

type Noble struct {
	Uuid     string
	Sequence int
	Cost     map[string]int
	Caption  string
}

func LoadCards() (l1, l2, l3 []*DevCard, nobles []*Noble) {
	l1 = make([]*DevCard, L1Num)
	l2 = make([]*DevCard, L2Num)
	l3 = make([]*DevCard, L3Num)
	nobles = make([]*Noble, NobleNum)
	// 打开 ../resources/cards.txt
	file, err := os.Open("resources/cards.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)
	// 创建读取器
	scanner := bufio.NewScanner(file)
	// 读取
	for i := 0; i < L1Num; i++ {
		scanner.Scan()
		l1[i] = NewDevCard(1, ColorList[i/8], scanner.Text())
	}
	for i := 0; i < L2Num; i++ {
		scanner.Scan()
		l2[i] = NewDevCard(2, ColorList[i/6], scanner.Text())
	}
	for i := 0; i < L3Num; i++ {
		scanner.Scan()
		l3[i] = NewDevCard(3, ColorList[i/4], scanner.Text())
	}
	for i := 0; i < NobleNum; i++ {
		scanner.Scan()
		nobles[i] = NewNoble(i, scanner.Text())
	}
	return
}

// NewDevCard 从字符串创建一张开发卡
func NewDevCard(level int, color, line string) *DevCard {
	// 先处理分数
	n := len(line)
	var points int
	if line[n-2] == '+' {
		points = int(line[n-1] - '0')
		line = line[:n-2]
		n -= 2
	}
	// 再处理宝石
	cost := make(map[string]int)
	for _, c := range ColorList {
		cost[c] = 0
	}
	for i := 0; i < n; i += 2 {
		cost[line[i+1:i+2]] = int(line[i] - '0')
	}
	var pointStr string
	if points > 0 {
		pointStr = fmt.Sprintf("+%d🔸", points)
	}
	caption := fmt.Sprintf("(%s%s)[%s]", color, pointStr, line)
	// 返回
	return &DevCard{
		Uuid:    uuid.New().String(),
		Level:   level,
		Color:   color,
		Points:  points,
		Cost:    cost,
		Caption: BeautifyCaption(caption),
	}
}

// NewNoble 从字符串创建一个贵族
func NewNoble(seq int, line string) *Noble {
	cost := make(map[string]int)
	for _, c := range ColorList {
		cost[c] = 0
	}
	for i := 0; i < len(line); i += 2 {
		cost[line[i+1:i+2]] = int(line[i] - '0')
	}
	caption := fmt.Sprintf("(+3🔸)[%s]", BeautifyCaption(line))
	return &Noble{
		Uuid:     uuid.New().String(),
		Sequence: seq,
		Cost:     cost,
		Caption:  caption,
	}
}

func InitSuggestWords() {
	// 打开 ../resources/words.txt
	file, err := os.Open("resources/words.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)
	// 创建读取器
	scanner := bufio.NewScanner(file)
	// 读取
	for scanner.Scan() {
		SuggestWords = append(SuggestWords, scanner.Text())
	}
}

func GetRandomSuggestion() string {
	return SuggestWords[rand.Intn(len(SuggestWords))]
}

func BeautifyCaption(str string) string {
	for _, c := range ColorList {
		str = strings.ReplaceAll(str, c, ColorDict[c])
	}
	return str
}

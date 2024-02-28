package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)
import "github.com/gin-gonic/gin"

var (
	ReverseMap = map[string]string{
		"w": "W",
		"u": "B",
		"g": "G",
		"r": "R",
		"b": "K",
		"*": "*",
	}
)

func CreateGameRouter(c *gin.Context) {
	fmt.Println("This is create!")
	gameId := c.Param("game")

	if _, exists := GameMap[gameId]; exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"result": gin.H{"error": "Game already exists, try another name"},
		})
		return
	}

	manager := NewGameManager(gameId)

	GameMap[gameId] = manager
	GameNum++

	c.JSON(http.StatusOK, gin.H{
		"game":  gameId,
		"start": manager.UuidStarter,
		"state": SerializeGame(manager.GamePtr, -1),
	})
}

func JoinGameRouter(c *gin.Context) {
	fmt.Println("This is join!")
	gameId := c.Param("game")

	manager, exists := GameMap[gameId]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"result": gin.H{"error": "Game not found"},
		})
		return
	}

	c.JSON(http.StatusOK, manager.JoinGame())
}

func WatchGameRouter(c *gin.Context) {
	fmt.Println("This is watch!")
	gameId := c.Param("game")

	manager, exists := GameMap[gameId]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"result": gin.H{"error": "Game not found"},
		})
		return
	}

	c.JSON(http.StatusOK, manager.WatchGame())
}

func StartGameRouter(c *gin.Context) {
	fmt.Println("This is start!")
	gameId := c.Param("game")
	uuidStarter := c.Param("starter")

	manager, exists := GameMap[gameId]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Game not found"})
		return
	}
	if manager.UuidStarter != uuidStarter {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not the starter"})
		return
	}
	if manager.Started {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Game has already started"})
		return
	}
	c.JSON(http.StatusOK, manager.StartGame())
}

func ChatRouter(c *gin.Context) {
	fmt.Println("This is chat!")
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	// 从 JSON 请求体中读取 msg 字段
	var msg string
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid msg"})
		return
	}

	c.JSON(http.StatusOK, manager.Chat(pid, msg))
}

func NextGameRouter(c *gin.Context) {
	fmt.Println("This is next!")
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	if pid != manager.GamePtr.ActivePlayerId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Now is not your turn"})
		return
	}

	manager.GamePtr.NextTurn()
	manager.doChange()

	c.JSON(http.StatusOK, gin.H{
		"state":  SerializeGame(manager.GamePtr, pid),
		"result": make(gin.H),
	})
}

func ActionRouter(c *gin.Context) {
	fmt.Println("This is action!")
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	game := manager.GamePtr
	if pid != game.ActivePlayerId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Now is not your turn"})
		return
	}

	act := c.Param("action")
	target := c.Param("target")

	var result gin.H

	switch act {
	case "take":
		result = game.Take(ReverseMap[target])
	case "discard":
		result = game.Discard(ReverseMap[target])
	case "buy":
		result = game.Buy(target)
	case "reserve":
		result = game.Reserve(target)
	case "noble_visit":
		result = game.VisitNoble(target)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}
	if result == nil {
		manager.doChange()
		result = make(gin.H)
	}
	c.JSON(http.StatusOK, gin.H{
		"state":  SerializeGame(game, pid),
		"result": result,
	})
}

func RenamePlayerRouter(c *gin.Context) {
	fmt.Println("This is rename!")
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	name := c.Param("name")
	manager.GamePtr.RenamePlayer(pid, name)
	manager.doChange()

	c.JSON(http.StatusOK, gin.H{
		// why?
		"status": "ok",
	})
}

func SuggestRouter(c *gin.Context) {
	fmt.Println("This is suggest!")
	word := GetRandomSuggestion()
	for _, exists := GameMap[word]; exists; {
		word = GetRandomSuggestion()
	}

	c.JSON(http.StatusOK, gin.H{
		"result": gin.H{
			"game": word,
		},
	})
}

func StatRouter(c *gin.Context) {
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"state": SerializeGame(manager.GamePtr, pid),
		"chat":  SerializeChatList(manager.ChatList),
	})
}

func PollRouter(c *gin.Context) {
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	result := manager.Poll(pid)

	// 返回结果
	c.JSON(http.StatusOK, result)
}

func ListRouter(c *gin.Context) {
	fmt.Println("This is list!")
	for k, manager := range GameMap {
		// 游戏未开始则 10 分钟后删除
		if !manager.Started && manager.CreateTime.Add(DeleteWaitingGame*time.Minute).Before(time.Now()) {
			delete(GameMap, k)
			GameNum--
		}
		// 游戏已开始则 24 小时后删除
		if manager.Started && manager.CreateTime.Add(DeletePlayingGame*time.Hour).Before(time.Now()) {
			delete(GameMap, k)
			GameNum--
		}
	}
	jsonList := make([]gin.H, 0)
	for _, manager := range GameMap {
		jsonList = append(jsonList, SerializeGameManager(manager))
	}
	c.JSON(http.StatusOK, gin.H{
		"games": jsonList,
	})
}

func validatePlayer(c *gin.Context) (*GameManager, int) {
	gameId := c.Param("game")
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pid"})
		return nil, -1
	}
	uid := c.Query("uuid")

	manager := ValidatePlayer(pid, uid, gameId)

	if manager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gameId / pid / uuid"})
	}

	return manager, pid
}

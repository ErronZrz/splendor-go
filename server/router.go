package main

import (
	"net/http"
	"strconv"
	"time"
)
import "github.com/gin-gonic/gin"

func CreateGameRouter(c *gin.Context) {
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
		"state": manager.GamePtr,
	})
}

func JoinGameRouter(c *gin.Context) {
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
	gameId := c.Param("game")
	uuidStarter := c.Query("starter")

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
	manager, pid := validatePlayer(c)

	if manager == nil {
		return
	}

	if pid != manager.GamePtr.ActivePlayerIndex {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Now is not your turn"})
		return
	}

	manager.GamePtr.NextTurn()
	manager.doChange()

	c.JSON(http.StatusOK, gin.H{
		"state":  manager.GamePtr,
		"result": make(gin.H),
	})
}

func ActionRouter(c *gin.Context) {
	manager, _ := validatePlayer(c)

	if manager == nil {
		return
	}

	act := c.Param("act")
	target := c.Param("target")

	game := manager.GamePtr
	var result map[string][]string

	switch act {
	case "take":
		result = game.Take(target)
	case "discard":
		result = game.Discard(target)
	case "buy":
		result = game.Buy(target)
	case "reserve":
		result = game.Reserve(target)
	case "visit":
		result = game.VisitNoble(target)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}
	if result == nil {
		manager.doChange()
		result = make(map[string][]string)
	}
	c.JSON(http.StatusOK, gin.H{
		"state":  game,
		"result": result,
	})
}

func RenamePlayerRouter(c *gin.Context) {
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
	manager, _ := validatePlayer(c)

	if manager == nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"state": manager.GamePtr,
		"chat":  manager.ChatList,
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
	for k, manager := range GameMap {
		jsonList = append(jsonList, gin.H{
			"game":    k,
			"players": manager.GetPlayerNum(),
			"started": manager.Started,
		})
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

	return manager, -1
}

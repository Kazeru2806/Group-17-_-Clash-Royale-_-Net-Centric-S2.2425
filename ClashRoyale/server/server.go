package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/user/tcr/configs"
	"github.com/user/tcr/game"
	"github.com/user/tcr/models"
)

// TCPServer represents a TCP server for the game
type TCPServer struct {
	listener       net.Listener
	manager        *game.Manager
	clients        map[string]*Client
	mutex          sync.Mutex
	gameIDGen      int
	waitingForGame *Client
}

// Client represents a connected client
type Client struct {
	conn      net.Conn
	username  string
	player    *models.Player
	server    *TCPServer
	sendMutex sync.Mutex
}

// Message represents a message from the client to the server
type Message struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// NewTCPServer creates a new TCP server
func NewTCPServer(address string) (*TCPServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	return &TCPServer{
		listener:  listener,
		manager:   game.NewManager(),
		clients:   make(map[string]*Client),
		mutex:     sync.Mutex{},
		gameIDGen: 1,
	}, nil
}

// Start starts the server
func (s *TCPServer) Start() {
	log.Printf("Server started on %s", s.listener.Addr().String())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		log.Printf("New connection from %s", conn.RemoteAddr())

		// Create a new client
		client := &Client{
			conn:   conn,
			server: s,
		}

		go client.handleConnection()
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message string) error {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	_, err := c.conn.Write([]byte(message + "\n"))
	return err
}

// handleConnection handles a client connection
func (c *Client) handleConnection() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in handleConnection: %v", r)
			c.SendMessage(fmt.Sprintf("Server encountered an error: %v", r))
			c.conn.Close()
		}
	}()

	defer c.conn.Close()

	// Send welcome message
	c.SendMessage("Welcome to Text-Based Clash Royale!")
	c.SendMessage("Please log in with: LOGIN <username> <password>")

	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		line := scanner.Text()
		if err := c.handleCommand(line); err != nil {
			c.SendMessage(fmt.Sprintf("Error: %v", err))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from client: %v", err)
	}

	// Cleanup if client disconnects
	if c.username != "" {
		c.server.mutex.Lock()
		delete(c.server.clients, c.username)
		c.server.mutex.Unlock()
		log.Printf("Client %s disconnected", c.username)
	}
}

// handleCommand handles a command from the client
func (c *Client) handleCommand(line string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in handleCommand: %v", r)
		}
	}()

	parts := strings.Split(line, " ")
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	command := strings.ToUpper(parts[0])
	args := parts[1:]

	switch command {
	case "LOGIN":
		return c.handleLogin(args)
	case "REGISTER":
		return c.handleRegister(args)
	case "PLAY":
		return c.handlePlay(args)
	case "DEPLOY":
		return c.handleDeploy(args)
	case "HELP":
		return c.handleHelp()
	case "STATS":
		return c.handleStats()
	case "TROOPS":
		return c.handleTroops()
	case "LEVEL":
		return c.handleLevel()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleLogin handles the LOGIN command
func (c *Client) handleLogin(args []string) error {
	// Recover from any panic that might occur
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in handleLogin: %v", r)
		}
	}()

	if len(args) < 2 {
		return fmt.Errorf("usage: LOGIN <username> <password>")
	}

	username := args[0]
	password := args[1]

	// Check if already logged in
	if c.username != "" {
		return fmt.Errorf("already logged in as %s", c.username)
	}

	// Check if username is already in use
	c.server.mutex.Lock()
	if _, exists := c.server.clients[username]; exists {
		c.server.mutex.Unlock()
		return fmt.Errorf("username %s is already in use", username)
	}
	c.server.mutex.Unlock()

	// Print debug information
	log.Printf("Attempting to load player data for: %s", username)

	// Try to load player data
	playerData, err := configs.LoadPlayerData(username)
	if err != nil {
		// Check if it's just because the file doesn't exist
		if os.IsNotExist(err) {
			return fmt.Errorf("user %s does not exist, please register first", username)
		}
		log.Printf("Error loading player data: %v", err)
		return fmt.Errorf("login failed: %v", err)
	}

	// Check password
	if playerData.Password != password {
		return fmt.Errorf("wrong password")
	}

	// Debug log
	log.Printf("Creating player for: %s", username)

	// Create player
	player := configs.CreatePlayer(username, password)
	player.EXP = playerData.EXP
	player.Level = playerData.Level

	// Debug log
	log.Printf("Setting tower levels for: %s", username)

	// Set tower levels
	for _, towerLevel := range playerData.Towers {
		for i, tower := range player.Towers {
			if tower.Type == towerLevel.Type {
				player.Towers[i].Level = towerLevel.Level
			}
		}
	}

	// Debug log
	log.Printf("Setting troop levels for: %s", username)

	// Set troop levels
	for _, troopLevel := range playerData.Troops {
		for i, troop := range player.Troops {
			if troop.Name == troopLevel.Name {
				player.Troops[i].Level = troopLevel.Level
			}
		}
	}

	// Debug log
	log.Printf("Applying level bonuses for: %s", username)

	// Apply level bonuses
	configs.ApplyLevelBonus(player)

	// Debug log
	log.Printf("Setting client data for: %s", username)

	// Set client data
	c.username = username
	c.player = player

	// Debug log
	log.Printf("Registering player with manager: %s", username)

	// Register with the game manager
	c.server.manager.RegisterPlayer(player, c)

	// Debug log
	log.Printf("Storing client in server map: %s", username)

	// Store client in the server's map
	c.server.mutex.Lock()
	c.server.clients[username] = c
	c.server.mutex.Unlock()

	// Debug log
	log.Printf("Sending success message to: %s", username)

	// Get level name and next level requirements
	levelName := configs.GetLevelName(player.Level)
	nextLevelExp := configs.GetExpRequiredForNextLevel(player.Level)

	// Send success message
	c.SendMessage(fmt.Sprintf("Logged in as %s (Level %d - %s)", username, player.Level, levelName))
	c.SendMessage(fmt.Sprintf("EXP: %d", player.EXP))

	// If not max level, show EXP needed for next level
	if nextLevelExp > 0 {
		expNeeded := nextLevelExp - (player.EXP % nextLevelExp)
		c.SendMessage(fmt.Sprintf("EXP needed for next level: %d", expNeeded))
	} else {
		c.SendMessage("You have reached the maximum level!")
	}

	return nil
}

// handleRegister handles the REGISTER command
func (c *Client) handleRegister(args []string) error {
	// Recover from any panic that might occur
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in handleRegister: %v", r)
		}
	}()

	if len(args) < 2 {
		return fmt.Errorf("usage: REGISTER <username> <password>")
	}

	username := args[0]
	password := args[1]

	// Check if already logged in
	if c.username != "" {
		return fmt.Errorf("already logged in as %s", c.username)
	}

	// Check if username is already in use
	c.server.mutex.Lock()
	if _, exists := c.server.clients[username]; exists {
		c.server.mutex.Unlock()
		return fmt.Errorf("username %s is already in use", username)
	}
	c.server.mutex.Unlock()

	// Check if user already exists in data store
	_, err := configs.LoadPlayerData(username)
	if err == nil {
		return fmt.Errorf("username %s already exists", username)
	}

	// Create player
	player := configs.CreatePlayer(username, password)

	// Save player data
	if err := configs.SavePlayerData(player); err != nil {
		return fmt.Errorf("error saving player data: %v", err)
	}

	// Set client data
	c.username = username
	c.player = player

	// Register with the game manager
	c.server.manager.RegisterPlayer(player, c)

	// Store client in the server's map
	c.server.mutex.Lock()
	c.server.clients[username] = c
	c.server.mutex.Unlock()

	// Get level name and next level requirements
	levelName := configs.GetLevelName(player.Level)
	nextLevelExp := configs.GetExpRequiredForNextLevel(player.Level)

	// Send success message
	c.SendMessage(fmt.Sprintf("Registered as %s (Level %d - %s)", username, player.Level, levelName))
	c.SendMessage(fmt.Sprintf("EXP: %d", player.EXP))
	c.SendMessage(fmt.Sprintf("EXP needed for next level: %d", nextLevelExp))

	return nil
}

// handlePlay handles the PLAY command
func (c *Client) handlePlay(args []string) error {
	if c.username == "" {
		return fmt.Errorf("you must log in first")
	}

	var gameMode string
	if len(args) > 0 {
		gameMode = strings.ToLower(args[0])
		if gameMode != "simple" && gameMode != "enhanced" {
			return fmt.Errorf("invalid game mode: %s (must be 'simple' or 'enhanced')", gameMode)
		}
	} else {
		gameMode = "simple" // Default mode
	}

	c.SendMessage(fmt.Sprintf("Looking for a game in %s mode...", gameMode))

	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()

	if c.server.waitingForGame == nil {
		// No one is waiting, so we'll wait
		c.server.waitingForGame = c
	} else if c.server.waitingForGame == c {
		// We're already waiting
		return fmt.Errorf("you're already in the queue")
	} else {
		// We found someone to play with
		opponent := c.server.waitingForGame
		c.server.waitingForGame = nil

		// Create a new game
		gameID := fmt.Sprintf("game_%d", c.server.gameIDGen)
		c.server.gameIDGen++

		game, err := c.server.manager.CreateGame(gameID, c.username, opponent.username, gameMode)
		if err != nil {
			return fmt.Errorf("error creating game: %v", err)
		}

		// Start the game based on mode
		if gameMode == "enhanced" {
			if err := c.server.manager.PlayEnhancedMode(gameID); err != nil {
				return fmt.Errorf("error starting enhanced mode: %v", err)
			}
		} else {
			// In simple mode, notify whose turn it is
			firstPlayer := game.Players[game.CurrentTurn]
			c.SendMessage(fmt.Sprintf("Game started! %s's turn", firstPlayer.Username))
			opponent.SendMessage(fmt.Sprintf("Game started! %s's turn", firstPlayer.Username))
		}
	}

	return nil
}

// handleDeploy handles the DEPLOY command
func (c *Client) handleDeploy(args []string) error {
	// Check if logged in
	if c.username == "" {
		return fmt.Errorf("you must log in first")
	}

	// Check if in a game
	gameID := c.getGameID()
	if gameID == "" {
		return fmt.Errorf("you are not in a game")
	}

	// Validate arguments
	if len(args) < 2 {
		return fmt.Errorf("usage: DEPLOY <troop_index> <tower_position>")
	}

	// Trim any whitespace or special characters
	troopIndexStr := strings.TrimSpace(args[0])
	towerPosStr := strings.TrimSpace(args[1])

	// Parse arguments
	troopIndex, err := strconv.Atoi(troopIndexStr)
	if err != nil {
		return fmt.Errorf("invalid troop index: %v", err)
	}

	// Make sure troop index is valid (0, 1, or 2 as per game rules)
	if troopIndex < 0 || troopIndex > 2 {
		return fmt.Errorf("invalid troop index: must be 0, 1, or 2")
	}

	towerPos, err := strconv.Atoi(towerPosStr)
	if err != nil {
		return fmt.Errorf("invalid tower position: %v", err)
	}

	// Get the game
	c.server.mutex.Lock()
	game, exists := c.server.manager.Games[gameID]
	c.server.mutex.Unlock()

	if !exists {
		return fmt.Errorf("game not found")
	}

	// Deploy the troop
	if game.GameMode == "simple" {
		if err := c.server.manager.PlayTurnWithMap(gameID, c.username, troopIndex, towerPos); err != nil {
			return err
		}
	} else {
		if err := c.server.manager.DeployTroop(gameID, c.username, troopIndex, towerPos); err != nil {
			return err
		}
	}

	return nil
}

// handleHelp displays help information
func (c *Client) handleHelp() error {
	help := `Available commands:
LOGIN <username> <password> - Log in with your username and password
REGISTER <username> <password> - Register a new account
PLAY [simple|enhanced] - Queue for a game (default: simple)
DEPLOY <troop_index> <tower_position> - Deploy a troop to attack an enemy tower
STATS - Display your stats and troop information
TROOPS - Show which of your troops are still alive or destroyed
LEVEL - View detailed information about your level and stat bonuses
HELP - Display this help information`

	return c.SendMessage(help)
}

// handleStats handles the STATS command
func (c *Client) handleStats() error {
	// Check if logged in
	if c.username == "" {
		return fmt.Errorf("you must log in first")
	}

	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()

	player, exists := c.server.manager.Players[c.username]
	if !exists {
		return fmt.Errorf("player data not found")
	}

	// Get level name and next level requirements
	levelName := configs.GetLevelName(player.Level)
	nextLevelExp := configs.GetExpRequiredForNextLevel(player.Level)

	// Build stats string
	stats := fmt.Sprintf("Player: %s (Level %d - %s)\n", player.Username, player.Level, levelName)
	stats += fmt.Sprintf("EXP: %d\n", player.EXP)

	// If not max level, show EXP required for next level
	if nextLevelExp > 0 {
		expNeeded := nextLevelExp - (player.EXP % nextLevelExp)
		stats += fmt.Sprintf("EXP needed for next level: %d\n", expNeeded)
	} else {
		stats += "You have reached the maximum level!\n"
	}

	// Add tower info
	stats += "Towers:\n"
	for _, tower := range player.Towers {
		stats += fmt.Sprintf("- %s %s (Level %d): HP %d/%d, ATK %d, DEF %d, CRIT %d%%\n",
			tower.GetSymbol(), tower.Type, tower.Level, tower.HP, tower.MaxHP, tower.ATK, tower.DEF, tower.CRIT)
	}

	// Add troop info - make sure we only show the first 3 troops as per game rules
	stats += "\nTroops (you have 3 troops):\n"

	// Ensure we only display 3 troops
	troopCount := len(player.Troops)
	if troopCount > 3 {
		troopCount = 3
	}

	for i := 0; i < troopCount; i++ {
		troop := player.Troops[i]
		stats += fmt.Sprintf("%d. %s %s (Level %d): HP %d/%d, ATK %d, DEF %d, MANA %d\n",
			i, troop.GetSymbol(), troop.Name, troop.Level, troop.HP, troop.MaxHP, troop.ATK, troop.DEF, troop.MANA)
		if troop.HasSpecial {
			stats += fmt.Sprintf("   Special: %s\n", troop.Special)
		}
	}

	return c.SendMessage(stats)
}

// handleTroops displays only the troop information with focus on which ones are still alive
func (c *Client) handleTroops() error {
	// Check if logged in
	if c.username == "" {
		return fmt.Errorf("you must log in first")
	}

	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()

	player, exists := c.server.manager.Players[c.username]
	if !exists {
		return fmt.Errorf("player data not found")
	}

	// Build troops list with status
	troops := "Your troops:\n"
	hasAliveTroops := false

	// Only display first 3 troops as per game rules
	troopCount := len(player.Troops)
	if troopCount > 3 {
		troopCount = 3
	}

	for i := 0; i < troopCount; i++ {
		troop := player.Troops[i]
		status := "ALIVE"
		if troop.HP <= 0 {
			status = "DESTROYED"
		} else {
			hasAliveTroops = true
		}

		troops += fmt.Sprintf("%d. %s %s (Level %d): HP %d/%d, ATK %d, DEF %d - Status: %s\n",
			i, troop.GetSymbol(), troop.Name, troop.Level, troop.HP, troop.MaxHP, troop.ATK, troop.DEF, status)
		if troop.HasSpecial {
			troops += fmt.Sprintf("   Special: %s\n", troop.Special)
		}
	}

	if !hasAliveTroops {
		troops += "\nWARNING: All your troops have been destroyed!\n"
	}

	return c.SendMessage(troops)
}

// getGameID returns the game ID that the client is in
func (c *Client) getGameID() string {
	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()

	for id, game := range c.server.manager.Games {
		for _, player := range game.Players {
			if player.Username == c.username {
				return id
			}
		}
	}
	return ""
}

// displayContinuousStats displays the current stats for both players in a game
func (c *Client) displayContinuousStats(gameID string) error {
	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()

	game, exists := c.server.manager.Games[gameID]
	if !exists {
		return fmt.Errorf("game not found")
	}

	// Build stats string for both players
	stats := "\n=== Current Game Stats ===\n\n"

	for i, player := range game.Players {
		// Get level name
		levelName := configs.GetLevelName(player.Level)

		stats += fmt.Sprintf("Player %d: %s (Level %d - %s)\n", i+1, player.Username, player.Level, levelName)
		stats += fmt.Sprintf("MANA: %d/%d\n", player.MANA, player.MaxMANA)

		// Add tower info
		stats += "Towers:\n"
		for _, tower := range player.Towers {
			status := "ALIVE"
			if tower.HP <= 0 {
				status = "DESTROYED"
			}
			stats += fmt.Sprintf("- %s %s (Level %d): HP %d/%d, ATK %d, DEF %d - Status: %s\n",
				tower.GetSymbol(), tower.Type, tower.Level, tower.HP, tower.MaxHP, tower.ATK, tower.DEF, status)
		}

		// Add troop info
		stats += "\nTroops:\n"
		for i, troop := range player.Troops {
			status := "ALIVE"
			if troop.HP <= 0 {
				status = "DESTROYED"
			}
			stats += fmt.Sprintf("%d. %s %s (Level %d): HP %d/%d, ATK %d, DEF %d, MANA %d - Status: %s\n",
				i, troop.GetSymbol(), troop.Name, troop.Level, troop.HP, troop.MaxHP, troop.ATK, troop.DEF, troop.MANA, status)
			if troop.HasSpecial {
				stats += fmt.Sprintf("   Special: %s\n", troop.Special)
			}
		}
		stats += "\n"
	}

	stats += "========================\n"
	return c.SendMessage(stats)
}

// handleLevel displays detailed level information
func (c *Client) handleLevel() error {
	// Check if logged in
	if c.username == "" {
		return fmt.Errorf("you must log in first")
	}

	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()

	player, exists := c.server.manager.Players[c.username]
	if !exists {
		return fmt.Errorf("player data not found")
	}

	// Get level name and next level requirements
	levelName := configs.GetLevelName(player.Level)
	nextLevelExp := configs.GetExpRequiredForNextLevel(player.Level)

	// Build level info string
	levelInfo := fmt.Sprintf("======== LEVEL INFORMATION ========\n")
	levelInfo += fmt.Sprintf("Player: %s\n", player.Username)
	levelInfo += fmt.Sprintf("Current Level: %d (%s)\n", player.Level, levelName)
	levelInfo += fmt.Sprintf("Current EXP: %d\n", player.EXP)

	// If not max level, show EXP needed for next level
	if nextLevelExp > 0 {
		expNeeded := nextLevelExp - (player.EXP % nextLevelExp)
		levelInfo += fmt.Sprintf("EXP needed for next level: %d\n", expNeeded)

		// Show progress
		totalExpNeeded := nextLevelExp
		currentProgress := nextLevelExp - expNeeded
		progressPercent := float64(currentProgress) / float64(totalExpNeeded) * 100
		levelInfo += fmt.Sprintf("Level Progress: %.1f%%\n", progressPercent)
	} else {
		levelInfo += "You have reached the maximum level!\n"
	}

	// Show level bonuses
	levelBonus := (player.Level - 1) * 10
	if levelBonus > 0 {
		levelInfo += fmt.Sprintf("\nLevel Bonus: +%d%% to all stats\n", levelBonus)
	} else {
		levelInfo += "\nLevel Bonus: None (Level 1)\n"
	}

	// Display the base stats and boosted stats for towers
	levelInfo += "\n--- Tower Stats With Level Bonuses ---\n"
	towerConfigs, _ := configs.LoadTowerConfig()

	for _, tower := range player.Towers {
		var baseConfig struct {
			Type string `json:"type"`
			HP   int    `json:"hp"`
			ATK  int    `json:"atk"`
			DEF  int    `json:"def"`
			CRIT int    `json:"crit"`
			EXP  int    `json:"exp"`
		}

		if tower.Type == "King Tower" {
			baseConfig = towerConfigs["king_tower"]
		} else {
			baseConfig = towerConfigs["guard_tower"]
		}

		levelInfo += fmt.Sprintf("%s (Level %d):\n", tower.Type, tower.Level)
		levelInfo += fmt.Sprintf("  HP: %d → %d\n", baseConfig.HP, tower.MaxHP)
		levelInfo += fmt.Sprintf("  ATK: %d → %d\n", baseConfig.ATK, tower.ATK)
		levelInfo += fmt.Sprintf("  DEF: %d → %d\n", baseConfig.DEF, tower.DEF)
	}

	// Display the base stats and boosted stats for troops
	levelInfo += "\n--- Troop Stats With Level Bonuses ---\n"
	troopConfigs, _ := configs.LoadTroopConfig()

	for _, troop := range player.Troops {
		var baseConfig struct {
			Name       string `json:"name"`
			HP         int    `json:"hp"`
			ATK        int    `json:"atk"`
			DEF        int    `json:"def"`
			MANA       int    `json:"mana"`
			EXP        int    `json:"exp"`
			Special    string `json:"special"`
			HasSpecial bool   `json:"has_special"`
		}

		// Find base config
		found := false
		for _, config := range troopConfigs {
			if config.Name == troop.Name {
				baseConfig = config
				found = true
				break
			}
		}

		if !found {
			continue
		}

		levelInfo += fmt.Sprintf("%s (Level %d):\n", troop.Name, troop.Level)
		levelInfo += fmt.Sprintf("  HP: %d → %d\n", baseConfig.HP, troop.MaxHP)
		levelInfo += fmt.Sprintf("  ATK: %d → %d\n", baseConfig.ATK, troop.ATK)
		levelInfo += fmt.Sprintf("  DEF: %d → %d\n", baseConfig.DEF, troop.DEF)
	}

	levelInfo += "===================================\n"

	return c.SendMessage(levelInfo)
}

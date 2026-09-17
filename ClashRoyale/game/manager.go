package game

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/user/tcr/configs"
	"github.com/user/tcr/models"
)

// Manager is responsible for managing the game state
type Manager struct {
	Games             map[string]*models.GameState
	Players           map[string]*models.Player
	GamesMutex        sync.Mutex
	ClientConnections map[string]Client
}

// Client interface for sending messages to clients
type Client interface {
	SendMessage(message string) error
}

// NewManager creates a new game manager
func NewManager() *Manager {
	return &Manager{
		Games:             make(map[string]*models.GameState),
		Players:           make(map[string]*models.Player),
		ClientConnections: make(map[string]Client),
	}
}

// RegisterPlayer registers a player with the manager
func (m *Manager) RegisterPlayer(player *models.Player, client Client) {
	m.GamesMutex.Lock()
	defer m.GamesMutex.Unlock()

	m.Players[player.Username] = player
	m.ClientConnections[player.Username] = client
}

// CreateGame creates a new game with the given mode and players
func (m *Manager) CreateGame(gameID string, player1, player2 string, gameMode string) (*models.GameState, error) {
	m.GamesMutex.Lock()
	defer m.GamesMutex.Unlock()

	if _, exists := m.Games[gameID]; exists {
		return nil, fmt.Errorf("game with ID %s already exists", gameID)
	}

	p1, ok1 := m.Players[player1]
	p2, ok2 := m.Players[player2]

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("one or both players not found")
	}

	// Log player levels before reset
	log.Printf("Creating game with player1: %s (Level %d) and player2: %s (Level %d)",
		p1.Username, p1.Level, p2.Username, p2.Level)

	// Reset player state for a new game
	m.resetPlayerForGame(p1)
	m.resetPlayerForGame(p2)

	// Create the game state
	game := &models.GameState{
		Players:     [2]*models.Player{p1, p2},
		CurrentTurn: 0,   // Player 1 starts
		TimeLeft:    180, // 3 minutes
		GameMode:    gameMode,
	}

	m.Games[gameID] = game

	// Notify players
	m.notifyPlayers(gameID, fmt.Sprintf("Game created: %s vs %s (Mode: %s)", player1, player2, gameMode))
	m.notifyPlayers(gameID, fmt.Sprintf("Game started! %s's turn", player1))

	// Show instructions for simple mode
	if gameMode == "simple" {
		m.notifyPlayers(gameID, "\n========= BATTLE GUIDE =========")
		m.notifyPlayers(gameID, "• Players start with 3 units & 3 defensive structures (1 Main, 2 Secondary)")
		m.notifyPlayers(gameID, "• Secondary Defense 1 must fall before targeting Secondary Defense 2 or Main Defense")
		m.notifyPlayers(gameID, "• Units that destroy a structure may immediately launch another attack")
		m.notifyPlayers(gameID, "• Command: DEPLOY <troops_number> <tower_number> to begin assault")
		m.notifyPlayers(gameID, "• Command: STATS to view current units and defenses")
		m.notifyPlayers(gameID, "===================================\n")

		// Highlight that it's player 1's turn
		m.notifyPlayers(gameID, "\n------------------------------")
		m.notifyPlayers(gameID, fmt.Sprintf("It's now %s's turn", player1))
		m.notifyPlayers(gameID, "------------------------------\n")
	}

	// Display the initial stats to show level bonuses are applied
	m.displayContinuousStats(gameID)

	battlefield := m.NewBattlefieldMap(gameID)
	battlefield.drawMap()

	return game, nil
}

// resetPlayerForGame prepares a player for a new game
func (m *Manager) resetPlayerForGame(player *models.Player) {
	// Apply level bonuses to get correct stats based on player level
	configs.ApplyLevelBonus(player)

	// Reset MANA for enhanced mode
	player.MANA = 5
	player.MaxMANA = 10

	// Reset HP for all troops and towers to their proper max values
	for _, tower := range player.Towers {
		tower.HP = tower.MaxHP
	}

	for _, troop := range player.Troops {
		troop.HP = troop.MaxHP
	}

	// Log that we've reset and applied level bonuses
	log.Printf("Reset player %s (Level %d) for game - applied level bonuses",
		player.Username, player.Level)
}

// notifyPlayers sends a message to all players in a game
func (m *Manager) notifyPlayers(gameID string, message string) {
	game, exists := m.Games[gameID]
	if !exists {
		return
	}

	for _, player := range game.Players {
		if client, ok := m.ClientConnections[player.Username]; ok {
			client.SendMessage(message)
		}
	}
}

// BattlefieldMap represents the visual battlefield state
type BattlefieldMap struct {
	width   int
	height  int
	gameID  string
	manager *Manager
}

// NewBattlefieldMap creates a new battlefield visualization
func (m *Manager) NewBattlefieldMap(gameID string) *BattlefieldMap {
	return &BattlefieldMap{
		width:   60,
		height:  20,
		gameID:  gameID,
		manager: m,
	}
}

// getTowerPosition returns the position of towers on the map
func (bm *BattlefieldMap) getTowerPosition(playerIndex, towerIndex int) (int, int) {
	// Player 1 towers (left side)
	if playerIndex == 0 {
		switch towerIndex {
		case 0: // King Tower
			return 8, 10
		case 1: // Guard Tower 1
			return 5, 6
		case 2: // Guard Tower 2
			return 5, 14
		}
	}
	// Player 2 towers (right side)
	if playerIndex == 1 {
		switch towerIndex {
		case 0: // King Tower
			return 52, 10
		case 1: // Guard Tower 1
			return 55, 6
		case 2: // Guard Tower 2
			return 55, 14
		}
	}
	return 0, 0
}

func (bm *BattlefieldMap) clearScreen() {
	// Just add some newlines to separate content
	bm.manager.notifyPlayers(bm.gameID, "\n\n")
}

func (bm *BattlefieldMap) drawMap() {
	game, exists := bm.manager.Games[bm.gameID]
	if !exists {
		return
	}

	var mapStr strings.Builder

	// Header
	mapStr.WriteString("\n" + strings.Repeat("=", 70) + "\n")
	mapStr.WriteString(fmt.Sprintf("                         BATTLEFIELD\n"))
	mapStr.WriteString(fmt.Sprintf(" Player 1: %-15s  VS  Player 2: %-15s\n",
		game.Players[0].Username, game.Players[1].Username))
	mapStr.WriteString(strings.Repeat("=", 70) + "\n")

	// Create visual battlefield
	battlefield := bm.createBattlefieldVisual(game)

	// Display battlefield
	for _, row := range battlefield {
		mapStr.WriteString("| " + row + " |\n")
	}

	mapStr.WriteString(strings.Repeat("=", 70) + "\n")

	// Game status
	mapStr.WriteString(fmt.Sprintf("Current Turn: %s | Time Left: %d seconds\n",
		game.Players[game.CurrentTurn].Username, game.TimeLeft))

	// Tower status with ASCII
	mapStr.WriteString("\nTower Status:\n")
	for _, player := range game.Players {
		mapStr.WriteString(fmt.Sprintf("  %s: ", player.Username))
		for _, tower := range player.Towers {
			status := "OK"
			symbol := "[G]" // Guard tower
			if tower.HP <= 0 {
				status = "XX"
				symbol = "[X]"
			} else if tower.Type == "King Tower" {
				symbol = "[K]"
			}
			mapStr.WriteString(fmt.Sprintf("%s %s %s(%d) ", status, symbol, tower.Type, tower.HP))
		}
		mapStr.WriteString("\n")
	}

	mapStr.WriteString(strings.Repeat("=", 70) + "\n")

	bm.manager.notifyPlayers(bm.gameID, mapStr.String())
}

func (bm *BattlefieldMap) createBattlefieldVisual(game *models.GameState) []string {
	battlefield := make([]string, 12)

	// Get tower symbols with ASCII
	p1Towers := make([]string, 3)
	p2Towers := make([]string, 3)

	for i, tower := range game.Players[0].Towers {
		if tower.HP <= 0 {
			p1Towers[i] = "[X]"
		} else if tower.Type == "King Tower" {
			p1Towers[i] = "[K]"
		} else {
			p1Towers[i] = "[G]"
		}
	}

	for i, tower := range game.Players[1].Towers {
		if tower.HP <= 0 {
			p2Towers[i] = "[X]"
		} else if tower.Type == "King Tower" {
			p2Towers[i] = "[K]"
		} else {
			p2Towers[i] = "[G]"
		}
	}

	// Create ASCII battlefield layout
	battlefield[0] = "                                                              "
	battlefield[1] = fmt.Sprintf("    %s                                           %s    ", p1Towers[1], p2Towers[1])
	battlefield[2] = "      |                                         |            "
	battlefield[3] = "      |                    vs                   |            "
	battlefield[4] = "      |                    ||                   |            "
	battlefield[5] = fmt.Sprintf("   %s  |                    ||                   |  %s   ", p1Towers[0], p2Towers[0])
	battlefield[6] = "      |                    ||                   |            "
	battlefield[7] = "      |                    vs                   |            "
	battlefield[8] = "      |                                         |            "
	battlefield[9] = fmt.Sprintf("    %s                                           %s    ", p1Towers[2], p2Towers[2])
	battlefield[10] = "                                                              "
	battlefield[11] = "   PLAYER 1 SIDE              ||              PLAYER 2 SIDE  "

	return battlefield
}

// ASCII-only animation that works in all terminals
func (bm *BattlefieldMap) animateTroopMovement(attackerPlayerIndex, troopIndex, targetPlayerIndex, targetTowerIndex int) {
	game, exists := bm.manager.Games[bm.gameID]
	if !exists {
		return
	}

	attacker := game.Players[attackerPlayerIndex]
	opponent := game.Players[targetPlayerIndex]
	troop := attacker.Troops[troopIndex]
	targetTower := opponent.Towers[targetTowerIndex]

	// Show troop symbol with ASCII
	troopSymbol := "[T]"
	if troop.Name == "Queen" {
		troopSymbol = "[Q]"
	}

	// Show movement sequence with text
	movementSteps := []string{
		fmt.Sprintf(">>> %s deploys %s %s", attacker.Username, troopSymbol, troop.Name),
		fmt.Sprintf(">>> %s %s advances across the battlefield...", troopSymbol, troop.Name),
		fmt.Sprintf(">>> %s %s charges toward %s's %s!", troopSymbol, troop.Name, opponent.Username, targetTower.Type),
		fmt.Sprintf(">>> %s %s reaches the target!", troopSymbol, troop.Name),
		fmt.Sprintf(">>> ATTACK! %s %s attacks %s's %s!", troopSymbol, troop.Name, opponent.Username, targetTower.Type),
	}

	for _, step := range movementSteps {
		bm.manager.notifyPlayers(bm.gameID, step)
		time.Sleep(400 * time.Millisecond)
	}
}

// drawMapWithTroop draws the map with a troop at specified position
func (bm *BattlefieldMap) drawMapWithTroop(troopX, troopY int, troopSymbol, message string) {
	bm.clearScreen()

	game, exists := bm.manager.Games[bm.gameID]
	if !exists {
		return
	}

	// Create battlefield (same as drawMap but with troop)
	battlefield := make([][]string, bm.height)
	for i := range battlefield {
		battlefield[i] = make([]string, bm.width)
		for j := range battlefield[i] {
			battlefield[i][j] = " "
		}
	}

	// Draw border and lines (same as drawMap)
	for i := 0; i < bm.height; i++ {
		battlefield[i][0] = "║"
		battlefield[i][bm.width-1] = "║"
	}
	for j := 0; j < bm.width; j++ {
		battlefield[0][j] = "═"
		battlefield[bm.height-1][j] = "═"
	}
	battlefield[0][0] = "╔"
	battlefield[0][bm.width-1] = "╗"
	battlefield[bm.height-1][0] = "╚"
	battlefield[bm.height-1][bm.width-1] = "╝"

	// Draw center line
	centerX := bm.width / 2
	for i := 1; i < bm.height-1; i++ {
		battlefield[i][centerX] = "│"
	}

	// Draw towers
	for playerIndex, player := range game.Players {
		for towerIndex, tower := range player.Towers {
			x, y := bm.getTowerPosition(playerIndex, towerIndex)
			if y < bm.height && x < bm.width {
				symbol := "🏰"
				if tower.HP <= 0 {
					symbol = "💥"
				} else if tower.Type == "King Tower" {
					symbol = "👑"
				} else {
					symbol = "🏯"
				}
				battlefield[y][x] = symbol
			}
		}
	}

	// Draw troop at current position
	if troopY < bm.height && troopX < bm.width && troopY >= 0 && troopX >= 0 {
		battlefield[troopY][troopX] = troopSymbol
	}

	// Display the map
	var mapStr strings.Builder
	mapStr.WriteString(fmt.Sprintf("╔═══════════════════════════ BATTLEFIELD ═══════════════════════════╗\n"))
	mapStr.WriteString(fmt.Sprintf("║ Player 1: %-12s                    Player 2: %-12s ║\n",
		game.Players[0].Username, game.Players[1].Username))
	mapStr.WriteString(fmt.Sprintf("╠════════════════════════════════════════════════════════════════════╣\n"))

	for i := 0; i < bm.height; i++ {
		mapStr.WriteString("║")
		for j := 0; j < bm.width; j++ {
			mapStr.WriteString(battlefield[i][j])
		}
		mapStr.WriteString("║\n")
	}

	mapStr.WriteString(fmt.Sprintf("╚════════════════════════════════════════════════════════════════════╝\n"))
	mapStr.WriteString(fmt.Sprintf(">>> %s\n", message))

	bm.manager.notifyPlayers(bm.gameID, mapStr.String())
}

// showCombatImpact shows combat effects at the target location
func (bm *BattlefieldMap) showCombatImpact(targetX, targetY int) {
	effects := []string{"💥", "⚡", "🔥", "💢"}

	for _, effect := range effects {
		bm.drawMapWithTroop(targetX, targetY, effect, "IMPACT!")
		time.Sleep(150 * time.Millisecond)
	}
}

// PlayTurnWithMap handles a player's turn in simple mode with battlefield visualization
func (m *Manager) PlayTurnWithMap(gameID string, playerUsername string, troopIndex int, targetTowerPos int) error {
	m.GamesMutex.Lock()
	defer m.GamesMutex.Unlock()

	// Create battlefield map
	battlefield := m.NewBattlefieldMap(gameID)

	// Show initial map
	battlefield.drawMap()

	game, exists := m.Games[gameID]
	if !exists {
		return fmt.Errorf("game with ID %s not found", gameID)
	}

	if game.GameMode != "simple" {
		return fmt.Errorf("PlayTurnWithMap is only for simple mode")
	}

	// Find player index
	var playerIndex int
	found := false
	for i, player := range game.Players {
		if player.Username == playerUsername {
			playerIndex = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("player %s not found in game", playerUsername)
	}
	// Check if it's this player's turn
	if game.CurrentTurn != playerIndex {
		return fmt.Errorf("not your turn")
	}

	player := game.Players[playerIndex]
	opponent := game.Players[(playerIndex+1)%2]

	// Validate troop index - must be 0, 1, or 2 as per game rules
	if troopIndex < 0 || troopIndex > 2 {
		return fmt.Errorf("invalid troop index: must be 0, 1, or 2")
	}

	// Ensure we have enough troops
	if len(player.Troops) <= troopIndex {
		return fmt.Errorf("troop with index %d not found", troopIndex)
	}

	// Check if the troop is still alive
	troop := player.Troops[troopIndex]
	if troop.HP <= 0 {
		// Show which troops are still available
		availableTroops := []int{}
		for i, t := range player.Troops {
			if t.HP > 0 {
				availableTroops = append(availableTroops, i)
			}
		}

		if len(availableTroops) == 0 {
			return fmt.Errorf("%s's %s has been destroyed! You have no troops left to deploy", playerUsername, troop.Name)
		}

		availableTroopsStr := ""
		for i, idx := range availableTroops {
			if i > 0 {
				availableTroopsStr += ", "
			}
			availableTroopsStr += fmt.Sprintf("%d (%s)", idx, player.Troops[idx].Name)
		}

		return fmt.Errorf("%s's %s has been destroyed! Available troops: %s", playerUsername, troop.Name, availableTroopsStr)
	}

	// Validate target tower position
	if targetTowerPos < 0 || targetTowerPos > 2 {
		return fmt.Errorf("invalid tower position")
	}

	// Make sure player follows the rule: must destroy guard towers before king tower
	if targetTowerPos == 0 { // King Tower
		// Check if both guard towers are destroyed
		if opponent.Towers[1].HP > 0 || opponent.Towers[2].HP > 0 {
			return fmt.Errorf("must destroy both guard towers before attacking king tower")
		}
	} else if targetTowerPos == 2 { // Second Guard Tower
		// Check if first guard tower is destroyed
		if opponent.Towers[1].HP > 0 {
			return fmt.Errorf("must destroy the first guard tower before attacking the second guard tower")
		}
	}

	// Check if target tower still has HP
	targetTower := opponent.Towers[targetTowerPos]
	if targetTower.HP <= 0 {
		// Suggest next valid target based on game rules
		nextTargetSuggestion := ""

		// Logic to suggest next target
		if targetTowerPos == 0 {
			// King Tower already destroyed - game should have ended
			return fmt.Errorf("%s's %s has already been destroyed! Game should be over.", opponent.Username, targetTower.Type)
		} else if targetTowerPos == 1 {
			// First Guard Tower destroyed
			if opponent.Towers[2].HP > 0 {
				nextTargetSuggestion = "Try attacking Guard Tower 2 (position 2)"
			} else if opponent.Towers[0].HP > 0 {
				nextTargetSuggestion = "Try attacking King Tower (position 0)"
			}
		} else if targetTowerPos == 2 {
			// Second Guard Tower destroyed
			if opponent.Towers[0].HP > 0 {
				nextTargetSuggestion = "Try attacking King Tower (position 0)"
			}
		}

		if nextTargetSuggestion != "" {
			return fmt.Errorf("%s's %s has already been destroyed! %s", opponent.Username, targetTower.Type, nextTargetSuggestion)
		} else {
			return fmt.Errorf("%s's %s has already been destroyed! Choose another target.", opponent.Username, targetTower.Type)
		}
	}

	m.notifyPlayers(gameID, fmt.Sprintf("\n%s deploys %s %s to attack %s's %s %s",
		playerUsername, troop.GetSymbol(), troop.Name, opponent.Username, targetTower.GetSymbol(), targetTower.Type))

	// Show troop movement animation
	battlefield.animateTroopMovement(playerIndex, troopIndex, (playerIndex+1)%2, targetTowerPos)

	if troop.HasSpecial && troop.Name == "Queen" {
		// Queen's special ability: heal the friendly tower with lowest HP
		lowestHPTower := m.findLowestHPTower(player)
		if lowestHPTower != nil {
			healAmount := 300
			// Calculate actual heal amount based on remaining HP space
			actualHeal := healAmount
			if lowestHPTower.HP+healAmount > lowestHPTower.MaxHP {
				actualHeal = lowestHPTower.MaxHP - lowestHPTower.HP
			}
			lowestHPTower.HP += actualHeal
			m.notifyPlayers(gameID, fmt.Sprintf("%s's Queen healed %s by %d HP", playerUsername, lowestHPTower.Type, actualHeal))
		}
	} else {
		// Normal attack
		damage := models.CalculateDamage(troop.ATK, targetTower.DEF, 0, "simple") // No crit in simple mode
		targetTower.HP -= damage

		// Ensure HP doesn't go below 0
		if targetTower.HP < 0 {
			targetTower.HP = 0
		}

		m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s dealt %d damage to %s's %s %s (ATK %d - DEF %d)",
			playerUsername, troop.GetSymbol(), troop.Name, damage, opponent.Username,
			targetTower.GetSymbol(), targetTower.Type, troop.ATK, targetTower.DEF))

		// Display tower remaining HP
		m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has %d/%d HP remaining",
			opponent.Username, targetTower.GetSymbol(), targetTower.Type, targetTower.HP, targetTower.MaxHP))

		// Check if tower is destroyed
		if targetTower.HP <= 0 {
			targetTower.HP = 0
			m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has been destroyed!", opponent.Username, targetTower.GetSymbol(), targetTower.Type))

			// Check if game is over (King Tower destroyed)
			if targetTowerPos == 0 {
				// Show final map before ending game
				time.Sleep(1 * time.Second)
				battlefield.drawMap()
				m.endGame(gameID, playerIndex)
				return nil
			}

			// If the troop destroyed a tower, it can continue attacking
			m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s can continue attacking!", playerUsername, troop.GetSymbol(), troop.Name))

			// Find the next valid target
			nextTarget := -1
			if targetTowerPos == 1 {
				// If first guard tower was destroyed, next is second guard tower
				if opponent.Towers[2].HP > 0 {
					nextTarget = 2
				} else if opponent.Towers[0].HP > 0 {
					// If second guard is already destroyed, next is king tower
					nextTarget = 0
				}
			} else if targetTowerPos == 2 {
				// If second guard tower was destroyed, next is king tower
				if opponent.Towers[0].HP > 0 {
					nextTarget = 0
				}
			}

			// Attack the next target if available
			if nextTarget != -1 {
				nextTower := opponent.Towers[nextTarget]
				m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s continues to attack %s's %s %s",
					playerUsername, troop.GetSymbol(), troop.Name, opponent.Username, nextTower.GetSymbol(), nextTower.Type))

				// Show continued attack animation
				battlefield.animateTroopMovement(playerIndex, troopIndex, (playerIndex+1)%2, nextTarget)

				damage := models.CalculateDamage(troop.ATK, nextTower.DEF, 0, "simple")
				nextTower.HP -= damage

				// Ensure HP doesn't go below 0
				if nextTower.HP < 0 {
					nextTower.HP = 0
				}

				m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s dealt %d damage to %s's %s %s (ATK %d - DEF %d)",
					playerUsername, troop.GetSymbol(), troop.Name, damage, opponent.Username,
					nextTower.GetSymbol(), nextTower.Type, troop.ATK, nextTower.DEF))

				// Display tower remaining HP
				m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has %d/%d HP remaining",
					opponent.Username, nextTower.GetSymbol(), nextTower.Type, nextTower.HP, nextTower.MaxHP))

				// Check if the next tower is also destroyed
				if nextTower.HP <= 0 {
					nextTower.HP = 0
					m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has been destroyed!", opponent.Username, nextTower.GetSymbol(), nextTower.Type))

					// Check if game is over (King Tower destroyed)
					if nextTarget == 0 {
						// Show final map before ending game
						time.Sleep(1 * time.Second)
						battlefield.drawMap()
						m.endGame(gameID, playerIndex)
						return nil
					}
				}
			}

			// Turn is complete after continuing attack
			game.CurrentTurn = (game.CurrentTurn + 1) % 2
			m.notifyPlayers(gameID, "\n------------------------------")
			m.notifyPlayers(gameID, fmt.Sprintf("It's now %s's turn", game.Players[game.CurrentTurn].Username))
			m.notifyPlayers(gameID, "------------------------------\n")
		} else {
			// Tower counter-attacks if not destroyed
			counterDamage := models.CalculateDamage(targetTower.ATK, troop.DEF, 0, "simple") // No crit in simple mode

			// Apply damage to the troop
			troop.HP -= counterDamage

			// Simulate troop taking damage
			if counterDamage > 0 {
				m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s counter-attacks and deals %d damage to %s's %s %s (ATK %d - DEF %d)",
					opponent.Username,
					targetTower.GetSymbol(),
					targetTower.Type,
					counterDamage,
					playerUsername,
					troop.GetSymbol(),
					troop.Name,
					targetTower.ATK,
					troop.DEF,
				))
				// Check if troop is destroyed
				if troop.HP <= 0 {
					troop.HP = 0
					m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has been destroyed!", playerUsername, troop.GetSymbol(), troop.Name))
				} else {
					m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has %d/%d HP remaining", playerUsername, troop.GetSymbol(), troop.Name, troop.HP, troop.MaxHP))
				}
			}
		}
	}

	// Turn is complete
	game.CurrentTurn = (game.CurrentTurn + 1) % 2
	m.notifyPlayers(gameID, ("\n------------------------------"))
	m.notifyPlayers(gameID, fmt.Sprintf("It's now %s's turn", game.Players[game.CurrentTurn].Username))
	m.notifyPlayers(gameID, ("------------------------------\n"))

	// Move cursor below map for status messages
	m.notifyPlayers(gameID, "\n")

	// Display continuous stats
	m.displayContinuousStats(gameID)

	return nil
}

// displayContinuousStats displays the current stats for both players in a game
// Each player sees full details of their own units but limited info about opponent's units
func (m *Manager) displayContinuousStats(gameID string) {
	game, exists := m.Games[gameID]
	if !exists {
		return
	}

	// Send personalized stats to each player
	for playerIndex, currentPlayer := range game.Players {
		opponentIndex := (playerIndex + 1) % 2
		opponent := game.Players[opponentIndex]

		stats := m.buildPersonalizedStats(currentPlayer, opponent, playerIndex+1, opponentIndex+1)

		// Send personalized stats to this specific player
		if client, ok := m.ClientConnections[currentPlayer.Username]; ok {
			client.SendMessage(stats)
		}
	}
}

// buildPersonalizedStats creates a personalized stats view for a player
func (m *Manager) buildPersonalizedStats(player *models.Player, opponent *models.Player, playerNum int, opponentNum int) string {
	stats := "\n=== Current Game Stats ===\n\n"

	// Show full details for the player themselves
	levelName := configs.GetLevelName(player.Level)
	stats += fmt.Sprintf("You (Player %d): %s (Level %d - %s)", playerNum, player.Username, player.Level, levelName)

	// Show level bonus if player is higher than level 1
	if player.Level > 1 {
		levelBonus := (player.Level - 1) * 10
		stats += fmt.Sprintf(" [+%d%% to all stats]", levelBonus)
	}

	stats += "\n"
	stats += fmt.Sprintf("MANA: %d/%d\n", player.MANA, player.MaxMANA)

	// Add player's tower info (full details)
	stats += "Your Towers:\n"
	for _, tower := range player.Towers {
		status := "ALIVE"
		if tower.HP <= 0 {
			status = "DESTROYED"
		}
		stats += fmt.Sprintf("- %s %s (Level %d): HP %d/%d, ATK %d, DEF %d - Status: %s\n",
			tower.GetSymbol(), tower.Type, tower.Level, tower.HP, tower.MaxHP, tower.ATK, tower.DEF, status)
	}

	// Add player's troop info (full details)
	stats += "\nYour Troops:\n"
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

	// Show limited details for the opponent
	opponentLevelName := configs.GetLevelName(opponent.Level)
	stats += fmt.Sprintf("Opponent (Player %d): %s (Level %d - %s)\n", opponentNum, opponent.Username, opponent.Level, opponentLevelName)

	// Add opponent's tower info (limited details - no ATK/DEF/MANA)
	stats += "Opponent's Towers:\n"
	for _, tower := range opponent.Towers {
		status := "ALIVE"
		if tower.HP <= 0 {
			status = "DESTROYED"
		}
		stats += fmt.Sprintf("- %s %s (Level %d): HP %d/%d - Status: %s\n",
			tower.GetSymbol(), tower.Type, tower.Level, tower.HP, tower.MaxHP, status)
	}

	// Add opponent's troop info (limited details - no ATK/DEF/MANA)
	stats += "\nOpponent's Troops:\n"
	for i, troop := range opponent.Troops {
		status := "ALIVE"
		if troop.HP <= 0 {
			status = "DESTROYED"
		}
		stats += fmt.Sprintf("%d. %s %s (Level %d): HP %d/%d - Status: %s\n",
			i, troop.GetSymbol(), troop.Name, troop.Level, troop.HP, troop.MaxHP, status)
		// Don't show opponent's special abilities details
		if troop.HasSpecial {
			stats += fmt.Sprintf("   Special: [Hidden]\n")
		}
	}
	stats += "\n"

	stats += "========================\n"
	return stats
}

// findLowestHPTower finds the tower with the lowest HP percentage
func (m *Manager) findLowestHPTower(player *models.Player) *models.Tower {
	var lowestTower *models.Tower
	lowestPercentage := 101.0 // Start with a value higher than possible

	for _, tower := range player.Towers {
		if tower.HP <= 0 {
			continue // Skip destroyed towers
		}

		percentage := float64(tower.HP) / float64(tower.MaxHP) * 100
		if percentage < lowestPercentage {
			lowestPercentage = percentage
			lowestTower = tower
		}
	}

	return lowestTower
}

// PlayEnhancedMode starts and manages a game in enhanced mode
func (m *Manager) PlayEnhancedMode(gameID string) error {
	game, exists := m.Games[gameID]
	if !exists {
		return fmt.Errorf("game with ID %s not found", gameID)
	}

	if game.GameMode != "enhanced" {
		return fmt.Errorf("PlayEnhancedMode is only for enhanced mode")
	}

	// Show enhanced mode game rules and instructions
	m.notifyPlayers(gameID, "\n--------- ENHANCED MODE RULES ---------")
	m.notifyPlayers(gameID, "- No turns. The game lasts 3 minutes, players attack continuously")
	m.notifyPlayers(gameID, "- MANA starts at 5, regenerates 1/sec, max is 10")
	m.notifyPlayers(gameID, "- Use MANA to deploy troops during the game")
	m.notifyPlayers(gameID, "- Critical hits: King Tower 10%, Guard Tower 5% chance")
	m.notifyPlayers(gameID, "- Critical hits deal 20% more damage")
	m.notifyPlayers(gameID, "- You must destroy Guard Tower 1 before attacking Guard Tower 2 or King Tower")
	m.notifyPlayers(gameID, "- Win by destroying King Tower or most towers when time ends")
	m.notifyPlayers(gameID, "- Use DEPLOY <troop_index> <tower_position> to attack")
	m.notifyPlayers(gameID, "--------------------------------------\n")

	// Display stats for players to see available troops
	m.displayContinuousStats(gameID)

	// Start the game timer in a separate goroutine
	go m.runEnhancedGameLoop(gameID)

	return nil
}

// runEnhancedGameLoop runs the game loop for enhanced mode
func (m *Manager) runEnhancedGameLoop(gameID string) {
	m.notifyPlayers(gameID, "Enhanced game started! 3 minutes on the clock.")

	// Initial notify that game has started and players can deploy troops
	m.notifyPlayers(gameID, "You can now deploy troops using your MANA!")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.GamesMutex.Lock()

		game, exists := m.Games[gameID]
		if !exists {
			m.GamesMutex.Unlock()
			return
		}

		// Decrease time
		game.TimeLeft--

		// Regenerate MANA for both players
		for _, player := range game.Players {
			if player.MANA < player.MaxMANA {
				player.MANA++
			}
		}

		// Check if game time is up
		if game.TimeLeft <= 0 {
			m.endEnhancedGame(gameID)
			m.GamesMutex.Unlock()
			return
		}

		// Notify time remaining every 10 seconds
		if game.TimeLeft%10 == 0 {
			m.notifyPlayers(gameID, fmt.Sprintf("Time remaining: %d seconds", game.TimeLeft))
		}

		m.GamesMutex.Unlock()
	}
}

// DeployTroop handles a player's turn in enhanced mode
func (m *Manager) DeployTroop(gameID string, playerUsername string, troopIndex int, targetTowerPos int) error {
	m.GamesMutex.Lock()
	defer m.GamesMutex.Unlock()

	game, exists := m.Games[gameID]
	if !exists {
		return fmt.Errorf("game with ID %s not found", gameID)
	}

	if game.GameMode != "enhanced" {
		return fmt.Errorf("DeployTroop is only for enhanced mode")
	}

	// Find player index
	playerIndex := -1
	for i, player := range game.Players {
		if player.Username == playerUsername {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		return fmt.Errorf("player %s not found in game", playerUsername)
	}

	player := game.Players[playerIndex]
	opponent := game.Players[(playerIndex+1)%2]

	// Validate troop index
	if troopIndex < 0 || troopIndex >= len(player.Troops) {
		// List available troops
		availableTroops := ""
		for i, t := range player.Troops {
			availableTroops += fmt.Sprintf("%d. %s (MANA: %d), ", i, t.Name, t.MANA)
		}
		return fmt.Errorf("invalid troop index. Available troops: %s", availableTroops[:len(availableTroops)-2])
	}

	troop := player.Troops[troopIndex]

	if player.MANA < troop.MANA {
		// List troops they can afford
		affordableTroops := ""
		for i, t := range player.Troops {
			if player.MANA >= t.MANA {
				affordableTroops += fmt.Sprintf("%d. %s (MANA: %d), ", i, t.Name, t.MANA)
			}
		}
		if affordableTroops == "" {
			return fmt.Errorf("not enough MANA (%d/%d) for %s. Wait for more MANA", player.MANA, troop.MANA, troop.Name)
		} else {
			return fmt.Errorf("not enough MANA (%d/%d) for %s. Affordable troops: %s", player.MANA, troop.MANA, troop.Name, affordableTroops[:len(affordableTroops)-2])
		}
	}

	// Validate target tower position
	if targetTowerPos < 0 || targetTowerPos > 2 {
		return fmt.Errorf("invalid tower position (0: King Tower, 1: Guard Tower 1, 2: Guard Tower 2)")
	}

	// Make sure player follows the rule: must destroy guard towers before king tower
	if targetTowerPos == 0 { // King Tower
		// Check if both guard towers are destroyed
		if opponent.Towers[1].HP > 0 || opponent.Towers[2].HP > 0 {
			return fmt.Errorf("must destroy both guard towers before attacking king tower")
		}
	} else if targetTowerPos == 2 { // Second Guard Tower
		// Check if first guard tower is destroyed
		if opponent.Towers[1].HP > 0 {
			return fmt.Errorf("must destroy the first guard tower before attacking the second guard tower")
		}
	}

	// Check if target tower still has HP
	targetTower := opponent.Towers[targetTowerPos]
	if targetTower.HP <= 0 {
		// Suggest next valid target based on game rules
		nextTargetSuggestion := ""

		// Logic to suggest next target
		if targetTowerPos == 0 {
			// King Tower already destroyed - game should have ended
			return fmt.Errorf("%s's %s has already been destroyed! Game should be over.", opponent.Username, targetTower.Type)
		} else if targetTowerPos == 1 {
			// First Guard Tower destroyed
			if opponent.Towers[2].HP > 0 {
				nextTargetSuggestion = "Try attacking Guard Tower 2 (position 2)"
			} else if opponent.Towers[0].HP > 0 {
				nextTargetSuggestion = "Try attacking King Tower (position 0)"
			}
		} else if targetTowerPos == 2 {
			// Second Guard Tower destroyed
			if opponent.Towers[0].HP > 0 {
				nextTargetSuggestion = "Try attacking King Tower (position 0)"
			}
		}

		if nextTargetSuggestion != "" {
			return fmt.Errorf("%s's %s has already been destroyed! %s", opponent.Username, targetTower.Type, nextTargetSuggestion)
		} else {
			return fmt.Errorf("%s's %s has already been destroyed! Choose another target.", opponent.Username, targetTower.Type)
		}
	}

	// Deduct MANA
	player.MANA -= troop.MANA

	// Process the troop action
	if troop.HasSpecial && troop.Name == "Queen" {
		// Queen's special ability: heal the friendly tower with lowest HP
		lowestHPTower := m.findLowestHPTower(player)
		if lowestHPTower != nil {
			healAmount := 300
			// Calculate actual heal amount based on remaining HP space
			actualHeal := healAmount
			if lowestHPTower.HP+healAmount > lowestHPTower.MaxHP {
				actualHeal = lowestHPTower.MaxHP - lowestHPTower.HP
			}
			lowestHPTower.HP += actualHeal
			m.notifyPlayers(gameID, fmt.Sprintf("%s's Queen healed %s by %d HP", playerUsername, lowestHPTower.Type, actualHeal))
		}
	} else {
		// Normal attack
		targetTower := opponent.Towers[targetTowerPos]

		// Calculate damage with potential crit
		critChance := 0
		if targetTower.Type == "King Tower" {
			critChance = 10
		} else {
			critChance = 5
		}

		damage := models.CalculateDamage(troop.ATK, targetTower.DEF, critChance, "enhanced")
		targetTower.HP -= damage

		// Ensure HP doesn't go below 0
		if targetTower.HP < 0 {
			targetTower.HP = 0
		}

		m.notifyPlayers(gameID, fmt.Sprintf("%s's %s dealt %d damage to %s's %s",
			playerUsername, troop.Name, damage, opponent.Username, targetTower.Type))

		// Display tower remaining HP
		m.notifyPlayers(gameID, fmt.Sprintf("%s's %s has %d/%d HP remaining",
			opponent.Username, targetTower.Type, targetTower.HP, targetTower.MaxHP))

		// Check if tower is destroyed
		if targetTower.HP <= 0 {
			targetTower.HP = 0
			m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has been destroyed!", opponent.Username, targetTower.GetSymbol(), targetTower.Type))

			// Check if game is over (King Tower destroyed)
			if targetTowerPos == 0 {
				m.endGame(gameID, playerIndex)
				return nil
			}
		} else {
			// Tower counter-attacks if not destroyed
			counterDamage := models.CalculateDamage(targetTower.ATK, troop.DEF, critChance, "enhanced")

			// Apply damage to the troop
			troop.HP -= counterDamage

			// Simulate troop taking damage
			if counterDamage > 0 {
				m.notifyPlayers(gameID, fmt.Sprintf("%s's %s counter-attacks and deals %d damage to %s's %s %s (ATK %d - DEF %d)",
					opponent.Username,
					targetTower.GetSymbol(),
					counterDamage,
					playerUsername,
					troop.GetSymbol(),
					troop.Name,
					targetTower.ATK,
					troop.DEF,
				))
				// Check if troop is destroyed
				if troop.HP <= 0 {
					troop.HP = 0
					m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has been destroyed!", playerUsername, troop.GetSymbol(), troop.Name))
				} else {
					m.notifyPlayers(gameID, fmt.Sprintf("%s's %s %s has %d/%d HP remaining", playerUsername, troop.GetSymbol(), troop.Name, troop.HP, troop.MaxHP))
				}
			}
		}
	}

	// Display updated stats after turn
	m.displayContinuousStats(gameID)

	return nil
}

// endGame ends the game with a winner
func (m *Manager) endGame(gameID string, winnerIndex int) {
	game, exists := m.Games[gameID]
	if !exists {
		return
	}

	winner := game.Players[winnerIndex]
	loser := game.Players[(winnerIndex+1)%2]

	// Base victory EXP
	baseExp := 30

	// Display detailed EXP summary for winner and get total EXP to award
	totalWinnerExp := m.displayExpSummary(gameID, winner, loser, baseExp, true)

	// Display detailed EXP summary for loser (they get 0 base EXP for losing)
	// But they might get some EXP from destroying towers
	totalLoserExp := m.displayExpSummary(gameID, loser, winner, 0, false)

	// Award EXP with all bonuses included
	m.AddPlayerEXP(winner, totalWinnerExp)
	if totalLoserExp > 0 {
		m.AddPlayerEXP(loser, totalLoserExp)
	}

	// Notify players of game outcome
	m.notifyPlayers(gameID, fmt.Sprintf("Game over! %s wins!", winner.Username))
	m.notifyPlayers(gameID, fmt.Sprintf("%s gained EXP (see summary above)", winner.Username))
	if totalLoserExp > 0 {
		m.notifyPlayers(gameID, fmt.Sprintf("%s also gained some EXP for towers destroyed", loser.Username))
	}

	// Save player data
	if err := configs.SavePlayerData(winner); err != nil {
		log.Printf("Error saving winner data: %v", err)
	}

	if err := configs.SavePlayerData(loser); err != nil {
		log.Printf("Error saving loser data: %v", err)
	}

	// Remove the game
	delete(m.Games, gameID)
}

// endEnhancedGame ends an enhanced mode game by checking tower counts
func (m *Manager) endEnhancedGame(gameID string) {
	game, exists := m.Games[gameID]
	if !exists {
		return
	}

	player1 := game.Players[0]
	player2 := game.Players[1]

	// Count destroyed towers
	p1DestroyedTowers := 0
	p2DestroyedTowers := 0

	for _, tower := range player1.Towers {
		if tower.HP <= 0 {
			p2DestroyedTowers++
		}
	}

	for _, tower := range player2.Towers {
		if tower.HP <= 0 {
			p1DestroyedTowers++
		}
	}

	// Determine winner
	if p1DestroyedTowers > p2DestroyedTowers {
		// Player 1 wins
		winnerBaseExp := 30
		m.notifyPlayers(gameID, fmt.Sprintf("Time's up! %s wins by destroying more towers (%d vs %d).",
			player1.Username, p1DestroyedTowers, p2DestroyedTowers))

		// Display detailed EXP summary for both players
		totalWinnerExp := m.displayExpSummary(gameID, player1, player2, winnerBaseExp, true) // Winner
		totalLoserExp := m.displayExpSummary(gameID, player2, player1, 0, false)             // Loser

		// Award EXP (including all bonuses)
		m.AddPlayerEXP(player1, totalWinnerExp)
		if totalLoserExp > 0 {
			m.AddPlayerEXP(player2, totalLoserExp)
		}

		m.notifyPlayers(gameID, fmt.Sprintf("%s gained EXP (see summary above)", player1.Username))
		if totalLoserExp > 0 {
			m.notifyPlayers(gameID, fmt.Sprintf("%s also gained some EXP for towers destroyed", player2.Username))
		}

	} else if p2DestroyedTowers > p1DestroyedTowers {
		// Player 2 wins
		winnerBaseExp := 30
		m.notifyPlayers(gameID, fmt.Sprintf("Time's up! %s wins by destroying more towers (%d vs %d).",
			player2.Username, p2DestroyedTowers, p1DestroyedTowers))

		// Display detailed EXP summary for both players
		totalWinnerExp := m.displayExpSummary(gameID, player2, player1, winnerBaseExp, true) // Winner
		totalLoserExp := m.displayExpSummary(gameID, player1, player2, 0, false)             // Loser

		// Award EXP (including all bonuses)
		m.AddPlayerEXP(player2, totalWinnerExp)
		if totalLoserExp > 0 {
			m.AddPlayerEXP(player1, totalLoserExp)
		}

		m.notifyPlayers(gameID, fmt.Sprintf("%s gained EXP (see summary above)", player2.Username))
		if totalLoserExp > 0 {
			m.notifyPlayers(gameID, fmt.Sprintf("%s also gained some EXP for towers destroyed", player1.Username))
		}

	} else {
		// Draw - both players get EXP
		drawBaseExp := 10
		m.notifyPlayers(gameID, "Time's up! The game ends in a draw.")

		// Display detailed EXP summary for both players
		totalP1Exp := m.displayExpSummary(gameID, player1, player2, drawBaseExp, false) // Draw for player 1
		totalP2Exp := m.displayExpSummary(gameID, player2, player1, drawBaseExp, false) // Draw for player 2

		// Award EXP (including all bonuses)
		m.AddPlayerEXP(player1, totalP1Exp)
		m.AddPlayerEXP(player2, totalP2Exp)
		m.notifyPlayers(gameID, "Both players gained EXP (see summary above)")
	}

	// Save player data
	if err := configs.SavePlayerData(player1); err != nil {
		log.Printf("Error saving player1 data: %v", err)
	}

	if err := configs.SavePlayerData(player2); err != nil {
		log.Printf("Error saving player2 data: %v", err)
	}

	// Remove the game
	delete(m.Games, gameID)
}

// AddPlayerEXP adds EXP to a player and handles level up if applicable
func (m *Manager) AddPlayerEXP(player *models.Player, expAmount int) {
	if player == nil {
		log.Printf("Warning: Attempted to add EXP to nil player")
		return
	}

	// Add the EXP directly - all bonuses should already be included in expAmount
	player.EXP += expAmount

	// Check for level up
	leveledUp, newLevel, levelName := configs.CheckLevelUp(player)
	if leveledUp {
		// If player levels up, notify them and apply new level bonuses
		client, exists := m.ClientConnections[player.Username]
		if exists {
			client.SendMessage(fmt.Sprintf("LEVEL UP! You are now Level %d (%s)!", newLevel, levelName))

			// Store HP percentages before applying level bonus
			towerHPPercentages := make([]float64, len(player.Towers))
			troopHPPercentages := make([]float64, len(player.Troops))

			for i, tower := range player.Towers {
				if tower.MaxHP > 0 {
					towerHPPercentages[i] = float64(tower.HP) / float64(tower.MaxHP)
				} else {
					towerHPPercentages[i] = 0
				}
			}

			for i, troop := range player.Troops {
				if troop.MaxHP > 0 {
					troopHPPercentages[i] = float64(troop.HP) / float64(troop.MaxHP)
				} else {
					troopHPPercentages[i] = 0
				}
			}

			// Apply level bonuses
			configs.ApplyLevelBonus(player)

			// Restore HP percentages
			for i, tower := range player.Towers {
				if tower.MaxHP > 0 && towerHPPercentages[i] > 0 {
					tower.HP = int(float64(tower.MaxHP) * towerHPPercentages[i])
					if tower.HP < 1 && towerHPPercentages[i] > 0 {
						tower.HP = 1 // Ensure towers with damage don't get set to 0 HP
					}
				}
			}

			for i, troop := range player.Troops {
				if troop.MaxHP > 0 && troopHPPercentages[i] > 0 {
					troop.HP = int(float64(troop.MaxHP) * troopHPPercentages[i])
					if troop.HP < 1 && troopHPPercentages[i] > 0 {
						troop.HP = 1 // Ensure troops with damage don't get set to 0 HP
					}
				}
			}

			// Also update the player in any active game
			var activeGameID string
			m.GamesMutex.Lock()
			for gameID, game := range m.Games {
				for i, gamePlayer := range game.Players {
					if gamePlayer.Username == player.Username {
						// Update the player's stats in the game
						game.Players[i] = player
						activeGameID = gameID
						break
					}
				}
				if activeGameID != "" {
					break
				}
			}
			m.GamesMutex.Unlock()

			// Display new stats after level up
			client.SendMessage(fmt.Sprintf("Your troops and towers are now 10%% stronger!"))

			// If player is in a game, display updated stats to all players
			if activeGameID != "" {
				m.displayContinuousStats(activeGameID)
			}
		}
	}
}

// EndGameTimeExpired handles the game's end when the timer expires
func (m *Manager) EndGameTimeExpired(gameID string) {
	game, exists := m.Games[gameID]
	if !exists {
		return
	}

	player1 := game.Players[0]
	player2 := game.Players[1]

	// Count destroyed towers
	p1DestroyedTowers := 0
	p2DestroyedTowers := 0

	for _, tower := range player1.Towers {
		if tower.HP <= 0 {
			p2DestroyedTowers++
		}
	}

	for _, tower := range player2.Towers {
		if tower.HP <= 0 {
			p1DestroyedTowers++
		}
	}

	// Determine winner
	if p1DestroyedTowers > p2DestroyedTowers {
		// Player 1 wins
		m.AddPlayerEXP(player1, 30)
		m.notifyPlayers(gameID, fmt.Sprintf("Time's up! %s wins by destroying more towers (%d vs %d).",
			player1.Username, p1DestroyedTowers, p2DestroyedTowers))
		m.notifyPlayers(gameID, fmt.Sprintf("%s gained 30 EXP", player1.Username))
	} else if p2DestroyedTowers > p1DestroyedTowers {
		// Player 2 wins
		m.AddPlayerEXP(player2, 30)
		m.notifyPlayers(gameID, fmt.Sprintf("Time's up! %s wins by destroying more towers (%d vs %d).",
			player2.Username, p2DestroyedTowers, p1DestroyedTowers))
		m.notifyPlayers(gameID, fmt.Sprintf("%s gained 30 EXP", player2.Username))
	} else {
		// Draw
		m.AddPlayerEXP(player1, 10)
		m.AddPlayerEXP(player2, 10)
		m.notifyPlayers(gameID, "Time's up! The game ends in a draw.")
		m.notifyPlayers(gameID, "Both players gained 10 EXP")
	}

	// Save player data
	if err := configs.SavePlayerData(player1); err != nil {
		log.Printf("Error saving player1 data: %v", err)
	}

	if err := configs.SavePlayerData(player2); err != nil {
		log.Printf("Error saving player2 data: %v", err)
	}

	// Remove the game
	delete(m.Games, gameID)
}

// displayExpSummary generates and displays a detailed EXP summary for a player at the end of the game
// and returns the total EXP to award
func (m *Manager) displayExpSummary(gameID string, player *models.Player, opponentPlayer *models.Player, baseExp int, isWinner bool) int {
	// Get game details
	game, exists := m.Games[gameID]
	if !exists {
		// Fallback to default values if game not found
		game = &models.GameState{TimeLeft: 0}
	}

	// Get tower configs for EXP values
	towerConfigs, err := configs.LoadTowerConfig()
	if err != nil {
		log.Printf("Failed to load tower configs: %v", err)
		return baseExp // Return basic EXP if config can't be loaded
	}

	// Count towers destroyed by player
	towersDestroyed := 0
	intactTowers := 0
	towerExpGained := 0

	// Count King Tower and Guard Towers separately for clarity
	kingTowerDestroyed := false
	guardTowersDestroyed := 0

	for _, tower := range opponentPlayer.Towers {
		if tower.HP <= 0 {
			towersDestroyed++
			// Count type-specific destructions
			if tower.Type == "King Tower" {
				kingTowerDestroyed = true
				towerExpGained += towerConfigs["king_tower"].EXP
			} else {
				guardTowersDestroyed++
				towerExpGained += towerConfigs["guard_tower"].EXP
			}
		}
	}

	// Count player's intact towers
	for _, tower := range player.Towers {
		if tower.HP > 0 {
			intactTowers++
		}
	}

	// Calculate victory bonus
	victoryBonus := 0
	if isWinner && intactTowers >= 2 {
		victoryBonus = 10
	}

	// Count the number of troops deployed (estimated based on MANA spent)
	// Starting MANA is 5, max is 10, regenerates 1/sec
	// In a 3-minute game with constant deployment, max theoretical MANA is
	// 5 (starting) + 180 (seconds) = 185 MANA
	// We'll estimate how many troops were deployed based on MANA used
	initialMana := 5
	// For a just-ended game, timeLeft should be 0, but we'll use a variable to be safe
	maxPossibleMana := initialMana + (180 - game.TimeLeft) // assuming 1 MANA per second over 3 minutes
	manaUsed := maxPossibleMana - player.MANA

	// Estimate average troop cost
	totalTroopMana := 0
	troopCount := 0
	for _, troop := range player.Troops {
		totalTroopMana += troop.MANA
		troopCount++
	}
	avgTroopCost := float64(totalTroopMana) / float64(troopCount)

	// Estimate troops deployed (with safety check for division by zero)
	estimatedTroopsDeployed := 0
	if avgTroopCost > 0 {
		estimatedTroopsDeployed = int(float64(manaUsed) / avgTroopCost)
	}

	// Calculate total EXP gained
	totalExp := baseExp + towerExpGained + victoryBonus

	// Generate summary
	summary := "\n=========== EXP SUMMARY ===========\n"
	summary += fmt.Sprintf("Player: %s (Level %d - %s)\n", player.Username, player.Level, configs.GetLevelName(player.Level))
	summary += fmt.Sprintf("Base EXP: %d\n", baseExp)

	// Tower destruction breakdown
	summary += fmt.Sprintf("Towers Destroyed: %d/%d\n", towersDestroyed, len(opponentPlayer.Towers))
	if towersDestroyed > 0 {
		if kingTowerDestroyed {
			summary += fmt.Sprintf("- King Tower: +%d EXP\n", towerConfigs["king_tower"].EXP)
		}
		if guardTowersDestroyed > 0 {
			summary += fmt.Sprintf("- Guard Towers: %d x %d = +%d EXP\n",
				guardTowersDestroyed,
				towerConfigs["guard_tower"].EXP,
				guardTowersDestroyed*towerConfigs["guard_tower"].EXP)
		}
		summary += fmt.Sprintf("Tower EXP Bonus: +%d\n", towerExpGained)
	}

	// Victory bonus
	if victoryBonus > 0 {
		summary += fmt.Sprintf("Victory Bonus (2+ towers intact): +%d\n", victoryBonus)
	}

	// Deployment statistics
	summary += fmt.Sprintf("Estimated Troops Deployed: %d\n", estimatedTroopsDeployed)

	// Total EXP
	summary += fmt.Sprintf("\nTOTAL EXP GAINED: %d\n", totalExp)

	// Next level information
	nextLevelExp := configs.GetExpRequiredForNextLevel(player.Level)
	if nextLevelExp > 0 {
		// Calculate how much EXP is needed for next level after this gain
		newTotalExp := player.EXP + totalExp
		expNeededForNextLevel := 0

		// Find the level requirement for the current player level
		levelConfig, err := configs.LoadLevelConfig()
		if err == nil {
			for _, lvl := range levelConfig.Levels {
				if lvl.Level == player.Level {
					// Calculate EXP needed for next level
					expNeededForNextLevel = lvl.RequiredExp + lvl.NextLevelExp - newTotalExp
					break
				}
			}
		}

		if expNeededForNextLevel > 0 {
			summary += fmt.Sprintf("EXP needed for next level: %d\n", expNeededForNextLevel)
		} else {
			summary += "You will level up after this match!\n"
		}
	} else {
		summary += "You have reached the maximum level!\n"
	}

	summary += "==================================\n"

	// Send the summary to the player
	if client, exists := m.ClientConnections[player.Username]; exists {
		client.SendMessage(summary)
	}

	// Return the total EXP so it can be applied
	return totalExp
}

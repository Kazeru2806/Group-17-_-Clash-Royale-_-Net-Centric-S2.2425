package configs

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/user/tcr/models"
)

// TowerConfig represents the structure of towers.json
type TowerConfig map[string]struct {
	Type string `json:"type"`
	HP   int    `json:"hp"`
	ATK  int    `json:"atk"`
	DEF  int    `json:"def"`
	CRIT int    `json:"crit"`
	EXP  int    `json:"exp"`
}

// TroopConfig represents the structure of troops.json
type TroopConfig map[string]struct {
	Name       string `json:"name"`
	HP         int    `json:"hp"`
	ATK        int    `json:"atk"`
	DEF        int    `json:"def"`
	MANA       int    `json:"mana"`
	EXP        int    `json:"exp"`
	Special    string `json:"special"`
	HasSpecial bool   `json:"has_special"`
}

// PlayerData represents a player's saved data
type PlayerData struct {
	Username string       `json:"username"`
	Password string       `json:"password"`
	EXP      int          `json:"exp"`
	Level    int          `json:"level"`
	Towers   []TowerLevel `json:"towers"`
	Troops   []TroopLevel `json:"troops"`
}

// TowerLevel represents tower level data for a player
type TowerLevel struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

// TroopLevel represents troop level data for a player
type TroopLevel struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// LevelConfig represents the structure of levels.json
type LevelConfig struct {
	Levels []struct {
		Level        int    `json:"level"`
		Name         string `json:"name"`
		RequiredExp  int    `json:"required_exp"`
		NextLevelExp int    `json:"next_level_exp"`
	} `json:"levels"`
}

// LoadTowerConfig loads the tower configuration from the JSON file
func LoadTowerConfig() (TowerConfig, error) {
	configPath := filepath.Join("configs", "towers.json")
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config TowerConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// LoadTroopConfig loads the troop configuration from the JSON file
func LoadTroopConfig() (TroopConfig, error) {
	configPath := filepath.Join("configs", "troops.json")
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config TroopConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// LoadLevelConfig loads the level configuration from the JSON file
func LoadLevelConfig() (*LevelConfig, error) {
	configPath := filepath.Join("configs", "levels.json")
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config LevelConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SavePlayerData saves a player's data to a JSON file
func SavePlayerData(player *models.Player) error {
	// Convert player model to PlayerData
	playerData := PlayerData{
		Username: player.Username,
		Password: player.Password,
		EXP:      player.EXP,
		Level:    player.Level,
		Towers:   make([]TowerLevel, 0),
		Troops:   make([]TroopLevel, 0),
	}

	for _, tower := range player.Towers {
		playerData.Towers = append(playerData.Towers, TowerLevel{
			Type:  tower.Type,
			Level: tower.Level,
		})
	}

	for _, troop := range player.Troops {
		playerData.Troops = append(playerData.Troops, TroopLevel{
			Name:  troop.Name,
			Level: troop.Level,
		})
	}

	// Create players directory if it doesn't exist
	playersDir := filepath.Join("data", "players")
	if err := os.MkdirAll(playersDir, 0755); err != nil {
		return err
	}

	// Save player data to file
	filePath := filepath.Join(playersDir, player.Username+".json")
	data, err := json.MarshalIndent(playerData, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, data, 0644)
}

// LoadPlayerData loads a player's data from a JSON file
func LoadPlayerData(username string) (*PlayerData, error) {
	filePath := filepath.Join("data", "players", username+".json")

	// Print debug info
	log.Printf("Attempting to load player data from: %s", filePath)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File does not exist: %s", filePath)
		} else {
			log.Printf("Error checking file: %s - %v", filePath, err)
		}
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %s - %v", filePath, err)
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file contents: %s - %v", filePath, err)
		return nil, err
	}

	// Print file contents for debugging
	log.Printf("Read file contents: %s", string(data))

	var playerData PlayerData
	err = json.Unmarshal(data, &playerData)
	if err != nil {
		log.Printf("Error unmarshaling JSON from file: %s - %v", filePath, err)
		return nil, err
	}

	// Validate expected fields
	if playerData.Username == "" {
		log.Printf("Warning: Username field is empty in file: %s", filePath)
	}

	return &playerData, nil
}

// CreatePlayer creates a new player with given username and password
func CreatePlayer(username, password string) *models.Player {
	towerConfigs, err := LoadTowerConfig()
	if err != nil {
		log.Fatalf("Failed to load tower configs: %v", err)
	}

	troopConfigs, err := LoadTroopConfig()
	if err != nil {
		log.Fatalf("Failed to load troop configs: %v", err)
	}

	// Create player with default values
	player := &models.Player{
		Username: username,
		Password: password,
		EXP:      0,
		Level:    1,
		Towers:   make([]*models.Tower, 0),
		Troops:   make([]*models.Troop, 0),
		MANA:     5,
		MaxMANA:  10,
	}

	// Add towers: 1 King Tower and 2 Guard Towers
	kingTowerConfig := towerConfigs["king_tower"]
	guardTowerConfig := towerConfigs["guard_tower"]

	// Add King Tower (position 0)
	kingTower := &models.Tower{
		Type:     kingTowerConfig.Type,
		HP:       kingTowerConfig.HP,
		MaxHP:    kingTowerConfig.HP,
		ATK:      kingTowerConfig.ATK,
		DEF:      kingTowerConfig.DEF,
		CRIT:     kingTowerConfig.CRIT,
		EXP:      kingTowerConfig.EXP,
		Level:    1,
		Position: 0,
	}
	player.Towers = append(player.Towers, kingTower)

	// Add 2 Guard Towers (positions 1 and 2)
	for i := 1; i <= 2; i++ {
		guardTower := &models.Tower{
			Type:     guardTowerConfig.Type,
			HP:       guardTowerConfig.HP,
			MaxHP:    guardTowerConfig.HP,
			ATK:      guardTowerConfig.ATK,
			DEF:      guardTowerConfig.DEF,
			CRIT:     guardTowerConfig.CRIT,
			EXP:      guardTowerConfig.EXP,
			Level:    1,
			Position: i,
		}
		player.Towers = append(player.Towers, guardTower)
	}

	// Randomly select exactly 3 troops from the available troop types
	troopTypes := []string{"pawn", "bishop", "rook", "knight", "prince", "queen"}

	// Shuffle the troop types
	for i := range troopTypes {
		j := rand.Intn(i + 1)
		troopTypes[i], troopTypes[j] = troopTypes[j], troopTypes[i]
	}

	// Select only the first 3 troops from the shuffled list
	for i := 0; i < 3; i++ {
		troopType := troopTypes[i]
		troopConfig := troopConfigs[troopType]

		troop := &models.Troop{
			Name:       troopConfig.Name,
			HP:         troopConfig.HP,
			MaxHP:      troopConfig.HP,
			ATK:        troopConfig.ATK,
			DEF:        troopConfig.DEF,
			MANA:       troopConfig.MANA,
			EXP:        troopConfig.EXP,
			Level:      1,
			Special:    troopConfig.Special,
			HasSpecial: troopConfig.HasSpecial,
		}
		player.Troops = append(player.Troops, troop)
	}

	// Make sure we have exactly 3 troops
	if len(player.Troops) != 3 {
		log.Printf("Warning: Player %s created with %d troops instead of 3", username, len(player.Troops))
	}

	return player
}

// ToPlayer converts PlayerData to Player model
func ToPlayer(data *PlayerData) (*models.Player, error) {
	towerConfigs, err := LoadTowerConfig()
	if err != nil {
		return nil, err
	}

	troopConfigs, err := LoadTroopConfig()
	if err != nil {
		return nil, err
	}

	player := &models.Player{
		Username: data.Username,
		Password: data.Password,
		EXP:      data.EXP,
		Level:    data.Level,
		Towers:   make([]*models.Tower, 0),
		Troops:   make([]*models.Troop, 0),
		MANA:     5,
		MaxMANA:  10,
	}

	// Load towers
	for _, towerLevel := range data.Towers {
		var towerConfig struct {
			Type string `json:"type"`
			HP   int    `json:"hp"`
			ATK  int    `json:"atk"`
			DEF  int    `json:"def"`
			CRIT int    `json:"crit"`
			EXP  int    `json:"exp"`
		}

		if towerLevel.Type == "King Tower" {
			towerConfig = towerConfigs["king_tower"]
		} else if towerLevel.Type == "Guard Tower" {
			towerConfig = towerConfigs["guard_tower"]
		}

		var position int
		if towerLevel.Type == "King Tower" {
			position = 0
		} else {
			// For Guard Towers, assign position 1 and 2
			position = len(player.Towers)
		}

		tower := &models.Tower{
			Type:     towerLevel.Type,
			HP:       towerConfig.HP,
			MaxHP:    towerConfig.HP,
			ATK:      towerConfig.ATK,
			DEF:      towerConfig.DEF,
			CRIT:     towerConfig.CRIT,
			EXP:      towerConfig.EXP,
			Level:    towerLevel.Level,
			Position: position,
		}

		player.Towers = append(player.Towers, tower)
	}

	// Check if we need to generate towers according to the rules
	if len(player.Towers) != 3 {
		// Clear towers and create them according to rules
		player.Towers = make([]*models.Tower, 0)

		// Add King Tower (position 0)
		kingTowerConfig := towerConfigs["king_tower"]
		kingTower := &models.Tower{
			Type:     kingTowerConfig.Type,
			HP:       kingTowerConfig.HP,
			MaxHP:    kingTowerConfig.HP,
			ATK:      kingTowerConfig.ATK,
			DEF:      kingTowerConfig.DEF,
			CRIT:     kingTowerConfig.CRIT,
			EXP:      kingTowerConfig.EXP,
			Level:    1,
			Position: 0,
		}
		player.Towers = append(player.Towers, kingTower)

		// Add 2 Guard Towers (positions 1 and 2)
		guardTowerConfig := towerConfigs["guard_tower"]
		for i := 1; i <= 2; i++ {
			guardTower := &models.Tower{
				Type:     guardTowerConfig.Type,
				HP:       guardTowerConfig.HP,
				MaxHP:    guardTowerConfig.HP,
				ATK:      guardTowerConfig.ATK,
				DEF:      guardTowerConfig.DEF,
				CRIT:     guardTowerConfig.CRIT,
				EXP:      guardTowerConfig.EXP,
				Level:    1,
				Position: i,
			}
			player.Towers = append(player.Towers, guardTower)
		}
	}

	// Load troops from saved data, or generate new ones
	player.Troops = make([]*models.Troop, 0) // Reset troops array

	// We'll create exactly 3 randomly selected troops
	troopTypes := []string{"pawn", "bishop", "rook", "knight", "prince", "queen"}

	// Shuffle the troop types
	for i := range troopTypes {
		j := rand.Intn(i + 1)
		troopTypes[i], troopTypes[j] = troopTypes[j], troopTypes[i]
	}

	// Create 3 troops with random types
	for i := 0; i < 3; i++ {
		troopType := troopTypes[i]
		troopConfig := troopConfigs[troopType]

		troop := &models.Troop{
			Name:       troopConfig.Name,
			HP:         troopConfig.HP,
			MaxHP:      troopConfig.HP,
			ATK:        troopConfig.ATK,
			DEF:        troopConfig.DEF,
			MANA:       troopConfig.MANA,
			EXP:        troopConfig.EXP,
			Level:      1, // Start with level 1, look for saved data later
			Special:    troopConfig.Special,
			HasSpecial: troopConfig.HasSpecial,
		}

		// Try to find level info from saved data
		for _, savedTroop := range data.Troops {
			if savedTroop.Name == troop.Name {
				troop.Level = savedTroop.Level
				break
			}
		}

		player.Troops = append(player.Troops, troop)
	}

	// Apply level bonuses to stats based on current levels
	ApplyLevelBonus(player)

	return player, nil
}

// ApplyLevelBonus applies level bonuses to a player's towers and towers
func ApplyLevelBonus(player *models.Player) {
	// Load base configs
	towerConfigs, err := LoadTowerConfig()
	if err != nil {
		log.Printf("Failed to load tower configs: %v", err)
		return
	}

	troopConfigs, err := LoadTroopConfig()
	if err != nil {
		log.Printf("Failed to load troop configs: %v", err)
		return
	}

	// Use player's level for all calculations
	playerLevel := player.Level

	// Apply level bonuses to towers
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

		// Apply bonus: 10% per player level above 1
		// A level 1 player has no bonus (multiplier is 1.0)
		// A level 2 player has 10% bonus (multiplier is 1.1)
		// A level 3 player has 20% bonus (multiplier is 1.2)
		// And so on...
		levelBonus := 1.0 + float64(playerLevel-1)*0.1
		tower.MaxHP = int(float64(baseConfig.HP) * levelBonus)
		tower.HP = tower.MaxHP // Reset HP to max
		tower.ATK = int(float64(baseConfig.ATK) * levelBonus)
		tower.DEF = int(float64(baseConfig.DEF) * levelBonus)

		// Set tower level to match player level (for display purposes)
		tower.Level = playerLevel
	}

	// Apply level bonuses to troops
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

		// Find the base config for this troop
		for _, config := range troopConfigs {
			if config.Name == troop.Name {
				baseConfig = config
				break
			}
		}

		// Apply bonus: 10% per player level above 1
		// A level 1 player has no bonus (multiplier is 1.0)
		// A level 2 player has 10% bonus (multiplier is 1.1)
		// A level 3 player has 20% bonus (multiplier is 1.2)
		// And so on...
		levelBonus := 1.0 + float64(playerLevel-1)*0.1
		troop.MaxHP = int(float64(baseConfig.HP) * levelBonus)

		// Only reset HP to max if the troop isn't destroyed (HP > 0)
		if troop.HP > 0 {
			troop.HP = troop.MaxHP
		}

		troop.ATK = int(float64(baseConfig.ATK) * levelBonus)
		troop.DEF = int(float64(baseConfig.DEF) * levelBonus)

		// Set troop level to match player level (for display purposes)
		troop.Level = playerLevel
	}
}

// CheckLevelUp checks if a player can level up based on their current EXP
// Returns true if the player leveled up, and the new level and level name
func CheckLevelUp(player *models.Player) (bool, int, string) {
	levelConfig, err := LoadLevelConfig()
	if err != nil {
		log.Printf("Failed to load level config: %v", err)
		return false, player.Level, ""
	}

	// Check if player's EXP is enough for next level
	currentLevel := player.Level
	if currentLevel >= len(levelConfig.Levels) {
		// Already at max level
		return false, currentLevel, ""
	}

	// Get next level info
	nextLevelIdx := currentLevel
	nextLevel := levelConfig.Levels[nextLevelIdx]

	if player.EXP >= nextLevel.RequiredExp {
		// Player can level up
		player.Level = nextLevel.Level
		return true, nextLevel.Level, nextLevel.Name
	}

	return false, currentLevel, ""
}

// GetLevelName returns the name of a given level
func GetLevelName(level int) string {
	levelConfig, err := LoadLevelConfig()
	if err != nil {
		log.Printf("Failed to load level config: %v", err)
		return "Unknown"
	}

	// Find level in config
	for _, lvl := range levelConfig.Levels {
		if lvl.Level == level {
			return lvl.Name
		}
	}

	return "Unknown"
}

// GetExpRequiredForNextLevel returns the EXP required to reach the next level
func GetExpRequiredForNextLevel(level int) int {
	levelConfig, err := LoadLevelConfig()
	if err != nil {
		log.Printf("Failed to load level config: %v", err)
		return 0
	}

	// Find level in config
	for _, lvl := range levelConfig.Levels {
		if lvl.Level == level {
			return lvl.NextLevelExp
		}
	}

	return 0
}

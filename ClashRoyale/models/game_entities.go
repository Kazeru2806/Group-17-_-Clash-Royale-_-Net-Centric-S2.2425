package models

import (
	"github.com/user/tcr/utils"
)

// Tower represents a tower in the game
type Tower struct {
	Type     string `json:"type"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	ATK      int    `json:"atk"`
	DEF      int    `json:"def"`
	CRIT     int    `json:"crit"` // Percentage chance of critical hit
	EXP      int    `json:"exp"`
	Level    int    `json:"level"`
	Position int    `json:"position"` // 0 for King Tower, 1-2 for Guard Towers
}

// Troop represents a troop in the game
type Troop struct {
	Name       string `json:"name"`
	HP         int    `json:"hp"`
	MaxHP      int    `json:"max_hp"`
	ATK        int    `json:"atk"`
	DEF        int    `json:"def"`
	MANA       int    `json:"mana"`
	EXP        int    `json:"exp"`
	Level      int    `json:"level"`
	Special    string `json:"special"`     // Special ability description
	HasSpecial bool   `json:"has_special"` // Whether the troop has a special ability
}

// Player represents a player in the game
type Player struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	EXP      int      `json:"exp"`
	Level    int      `json:"level"`
	Towers   []*Tower `json:"towers"`
	Troops   []*Troop `json:"troops"`
	MANA     int      `json:"mana"`
	MaxMANA  int      `json:"max_mana"`
}

// GameState represents the current state of the game
type GameState struct {
	Players     [2]*Player `json:"players"`
	CurrentTurn int        `json:"current_turn"` // 0 or 1, representing player index
	TimeLeft    int        `json:"time_left"`    // Time left in seconds
	GameMode    string     `json:"game_mode"`    // "simple" or "enhanced"
}

// CalculateDamage calculates the damage dealt by attacker to defender
func CalculateDamage(attackerATK int, defenderDEF int, critChance int, gameMode string) int {
	// Simple damage calculation according to the specified rules: DMG = ATK_A - DEF_B
	damage := attackerATK - defenderDEF

	if gameMode == "enhanced" {

		critRoll := utils.GetRandomNumber(1, 100)
		if critRoll <= critChance {

			damage = int(float64(damage) * 1.2)
		}
	}

	// If damage is negative, no damage is dealt
	if damage < 0 {
		damage = 0
	}

	return damage
}

// GetSymbol returns a visual symbol representing the troop type
func (t *Troop) GetSymbol() string {
	switch t.Name {
	case "Pawn":
		return "●"
	case "Bishop":
		return "♦"
	case "Rook":
		return "■"
	case "Knight":
		return "♘"
	case "Prince":
		return "★"
	case "Queen":
		return "♕"
	default:
		return "?"
	}
}

// GetSymbol returns a visual symbol representing the tower type
func (t *Tower) GetSymbol() string {
	switch t.Type {
	case "King Tower":
		return "◆"
	case "Guard Tower":
		return "▲"
	default:
		return "?"
	}
}

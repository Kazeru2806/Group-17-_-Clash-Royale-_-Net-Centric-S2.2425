# Text-Based Clash Royale (TCR)

A text-based implementation of Clash Royale using Golang and TCP networking.

## Simple TCR Rules

- **Two players connect** to the server using a username and password.
- **Each player has 3 towers:** 1 King Tower and 2 Guard Towers.
- **Attack order:** Players must destroy the 1st Guard Tower before attacking the 2nd Guard Tower or King Tower.
- **Each player has 3 troops**, randomly spawned from a defined list.
- **All towers and troops** have HP, ATK, and DEF values.
- **Damage Formula:** DMG = ATK_A - DEF_B (if ≥ 0); then HP_B = HP_B - DMG
- **Turn-based attacks:** Players take turns deploying troops to attack. If a troop destroys a tower in one turn, it can continue attacking the next tower.

## System Architecture

The system is built with a client-server architecture:

- **Server**: Manages game state, player data, and game logic
- **Client**: Connects to the server and provides a text-based interface for players

The system has two game modes:
1. **Simple Mode**: Turn-based gameplay with simple damage calculation
2. **Enhanced Mode**: Real-time gameplay with mana system, critical hits, and timed matches

## Components

- **Models**: Game entities (towers, troops, players)
- **Configs**: Configuration loaders for game data
- **Game**: Game logic and state management
- **Server**: TCP server implementation
- **Client**: TCP client implementation
- **Utils**: Utility functions

## Protocol Description

The application uses a simple text-based protocol over TCP:

### Client-to-Server Messages

- `LOGIN <username> <password>`: Log in with existing credentials
- `REGISTER <username> <password>`: Register a new account
- `PLAY [simple|enhanced]`: Queue for a game (default: simple)
- `DEPLOY <troop_index> <tower_position>`: Deploy a troop to attack an enemy tower
- `STATS`: Display player statistics and troops
- `HELP`: Display available commands

### Server-to-Client Messages

- Status messages for game events (damage dealt, tower destroyed, etc.)
- Error messages
- Game state updates

## Deployment & Execution

### Prerequisites

- Go 1.16 or higher

### Building

```bash
# Clone the repository
git clone https://github.com/user/tcr.git
cd tcr

# Build the server
go build -o bin/tcr-server ./cmd/server

# Build the client
go build -o bin/tcr-client ./cmd/client
```

### How to run

#### Server


# Open terminal

# Run with this command
go run cmd/server/main.go

#### Client

# Go to folder bin

# Double click tcr-client.exe

## Game Rules

### Simple Mode

- Turn-based gameplay
- Each player has 3 towers (1 King Tower, 2 Guard Towers)
- Players must destroy Guard Towers in order before attacking the King Tower
- Damage formula: DMG = ATK_A - DEF_B
- If a troop destroys a tower in one turn, it can continue attacking

### Enhanced Mode

- Real-time gameplay (3-minute matches)
- Includes critical hit chance: DMG = ATK_A or (ATK_A * 1.2 if CRIT) - DEF_B
- Mana system: starts at 5, regenerates 1/sec, max is 10
- Win conditions:
  - Destroy the King Tower first, or
  - Destroy more towers when time runs out
- EXP system:
  - Win: 30 EXP
  - Draw: 10 EXP each

## Towers and Troops

### Towers

| Type        | HP    | ATK  | DEF  | CRIT | EXP |
|-------------|-------|------|------|------|-----|
| King Tower  | 2000  | 500  | 300  | 10%  | 200 |
| Guard Tower | 1000  | 300  | 100  | 5%   | 100 |

### Troops

| Name   | HP  | ATK  | DEF  | MANA | EXP | Special                                          |
|--------|-----|------|------|------|-----|--------------------------------------------------|
| Pawn   | 50  | 150  | 100  | 3    | 5   |                                                  |
| Bishop | 100 | 200  | 150  | 4    | 10  |                                                  |
| Rook   | 250 | 200  | 200  | 5    | 25  |                                                  |
| Knight | 200 | 300  | 150  | 5    | 25  |                                                  |
| Prince | 500 | 400  | 300  | 6    | 50  |                                                  |
| Queen  | 200 | 0    | 100  | 5    | 30  | Heals the friendly tower with lowest HP by 300   | 
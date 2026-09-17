# System Architecture and Sequence Diagrams

## System Architecture

The Text-Based Clash Royale (TCR) game is built with a client-server architecture using TCP for network communication.

### Components

1. **Client**
   - Handles user input and displays game information
   - Communicates with the server using text-based commands
   - Simple interface for testing/playing the game

2. **Server**
   - Manages game state
   - Handles player authentication and data persistence
   - Executes game logic
   - Broadcasts game events to connected clients

3. **Game Manager**
   - Manages active games
   - Processes player actions
   - Applies game rules
   - Handles game mode-specific logic

4. **Config Manager**
   - Loads configuration data from JSON files
   - Manages player data persistence
   - Handles level bonuses and stat calculations

## Sequence Diagrams

### Player Registration and Login

```
Client                 Server                  ConfigManager
  |                      |                           |
  |--- REGISTER user --->|                           |
  |                      |--- Create player -------->|
  |                      |<-- Player created --------|
  |                      |--- Save player data ----->|
  |                      |<-- Data saved ------------|
  |<-- Success message --|                           |
  |                      |                           |
  |--- LOGIN user ------>|                           |
  |                      |--- Load player data ----->|
  |                      |<-- Player data ------------|
  |                      |--- Validate password ---->|
  |                      |<-- Valid/Invalid ----------|
  |<-- Success/Error ----|                           |
```

### Game Matchmaking

```
Client1                Server                Client2
  |                      |                      |
  |---- PLAY simple ---->|                      |
  |<--- Waiting ---------|                      |
  |                      |<---- PLAY simple ----|
  |                      |--- Create game ----->|
  |<--- Game started ----|--- Game started ---->|
```

### Simple Mode Gameplay

```
Client1                Server                Client2
  |                      |                      |
  |--- DEPLOY 1 2 ------>|                      |
  |                      |--- Process turn ---->|
  |                      |<-- Turn processed ---|
  |<--- Damage message --|--- Damage message -->|
  |                      |                      |
  |                      |<--- DEPLOY 0 1 ------|
  |                      |--- Process turn ---->|
  |                      |<-- Turn processed ---|
  |<--- Damage message --|--- Damage message -->|
```

### Enhanced Mode Gameplay

```
Client1                Server                Client2
  |                      |                      |
  |--- PLAY enhanced --->|                      |
  |<--- Waiting ---------|                      |
  |                      |<--- PLAY enhanced ---|
  |                      |--- Create game ----->|
  |<--- Game started ----|--- Game started ---->|
  |                      |                      |
  |--- DEPLOY 1 2 ------>|                      |
  |                      |--- Process deploy -->|
  |                      |<-- Deploy processed -|
  |<--- Damage message --|--- Damage message -->|
  |                      |                      |
  |                      |<--- DEPLOY 0 1 ------|
  |                      |--- Process deploy -->|
  |                      |<-- Deploy processed -|
  |<--- Damage message --|--- Damage message -->|
  |                      |                      |
  |                      |--- Timer updated --->|
  |<--- Time's up! ------|--- Time's up! ------>|
  |<--- Results ---------|--- Results --------->|
```

## Data Flow

### Player Data Storage

Player data is stored in JSON files under the `data/players` directory. Each player has their own file named `username.json` containing:

- Authentication data (username, password)
- Experience points and level
- Tower levels
- Troop levels

### Game State Management

Game states are stored in memory during active games and include:

- Player references
- Tower states (HP, ATK, DEF)
- Current turn (simple mode)
- Time left (enhanced mode)
- Game mode

### Network Protocol

The application uses a simple text-based protocol over TCP:

#### Client-to-Server Commands:

- `LOGIN <username> <password>`
- `REGISTER <username> <password>`
- `PLAY [simple|enhanced]`
- `DEPLOY <troop_index> <tower_position>`
- `STATS`
- `HELP`

#### Server-to-Client Messages:

- Status messages
- Error messages
- Game state updates

## Deployment Architecture

The system can be deployed in various configurations:

1. **Local Development**
   - Server and clients on the same machine
   - Useful for testing and development

2. **Local Network**
   - Server on one machine in the network
   - Clients connect from other machines on the same network

3. **Internet Deployment**
   - Server deployed on a cloud instance with a public IP
   - Clients connect from anywhere on the internet 
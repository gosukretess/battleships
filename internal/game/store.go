package game

import (
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(path string) *Store {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("Cannot open db", err)
	}

	createGameTable := `
    CREATE TABLE IF NOT EXISTS games (
        id TEXT PRIMARY KEY,
        userid1 TEXT,
        userid2 TEXT,
		created TEXT,
		nextuser TEXT
    );`
	if _, err := db.Exec(createGameTable); err != nil {
		log.Fatal("cannot create game table:", err)
	}

	createShipTable := `
    CREATE TABLE IF NOT EXISTS ships (
        gameid TEXT,
        userid TEXT,
        x INTEGER,
		y INTEGER
    );`
	if _, err := db.Exec(createShipTable); err != nil {
		log.Fatal("cannot create ship table:", err)
	}

	createMovesTable := `
    CREATE TABLE IF NOT EXISTS moves (
        gameid TEXT,
        userid TEXT,
        x INTEGER,
		y INTEGER,
		hit BOOLEAN
    );`
	if _, err := db.Exec(createMovesTable); err != nil {
		log.Fatal("cannot create moves table:", err)
	}

	return &Store{db: db}
}

func (s *Store) CreateGame(userId1, userId2 string) (GameDto, error) {
	id := uuid.New().String()
	currentTime := time.Now().UTC().Format(time.RFC3339)

	gameDto := GameDto{
		Id:       id,
		UserId1:  userId1,
		UserId2:  userId2,
		Created:  currentTime,
		NextUser: userId1,
	}

	_, err := s.db.Exec("INSERT INTO games(id, userid1, userid2, created, nextuser) VALUES (?, ?, ?, ?, ?)", gameDto.Id, gameDto.UserId1, gameDto.UserId2, gameDto.Created, gameDto.NextUser)
	return gameDto, err
}

func (s *Store) AddShip(gameId, userId string, x, y int) {
	s.db.Exec("INSERT INTO ships(gameid, userid, x, y) VALUES (?, ?, ?, ?)", gameId, userId, x, y)
}

func (s *Store) GetGames() ([]GameDto, error) {
	rows, err := s.db.Query("SELECT id, userid1, userid2, created, nextuser FROM games")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []GameDto
	for rows.Next() {
		var u GameDto
		if err := rows.Scan(&u.Id, &u.UserId1, &u.UserId2, &u.Created, &u.NextUser); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) GetShips(gameId, userId string) ([]ShipDto, error) {
	rows, err := s.db.Query("SELECT gameid, userid, x, y FROM ships WHERE gameid = ? AND userid = ?",
		gameId, userId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ships []ShipDto
	for rows.Next() {
		var ship ShipDto
		if err := rows.Scan(&ship.GameId, &ship.UserId, &ship.X, &ship.Y); err != nil {
			return nil, err
		}
		ships = append(ships, ship)
	}
	return ships, nil
}

func (s *Store) GetMoves(gameId, userId string) ([]MoveDto, error) {
	rows, err := s.db.Query("SELECT gameid, userid, x, y, hit FROM moves WHERE gameid = ? AND userid = ?",
		gameId, userId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moves []MoveDto
	for rows.Next() {
		var rec MoveDto
		if err := rows.Scan(&rec.GameId, &rec.UserId, &rec.X, &rec.Y, &rec.Hit); err != nil {
			return nil, err
		}
		moves = append(moves, rec)
	}
	return moves, nil
}

func (s *Store) AreCoordsTaken(gameId, userId string, x, y int) (bool, error) {
	query := "SELECT 1 FROM moves WHERE gameid = ? AND userid = ? AND x = ? AND y = ? LIMIT 1"
	var exists int
	err := s.db.QueryRow(query, gameId, userId, x, y).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, err
	}
	return true, nil
}

func (s *Store) Move(gameId, userId string, x, y int) (bool, error) {
	query := "SELECT 1 FROM ships WHERE gameid = ? AND userid <> ? AND x = ? AND y = ? LIMIT 1"
	var exists int
	err := s.db.QueryRow(query, gameId, userId, x, y).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	hit := exists == 1
	insert := `INSERT INTO moves (gameid, userid, x, y, hit) VALUES (?, ?, ?, ?, ?)`
	_, err = s.db.Exec(insert, gameId, userId, x, y, hit)
	if err != nil {
		return hit, err
	}

	updateQuery := `
		UPDATE games 
		SET nextuser = CASE 
			WHEN userid1 = ? THEN userid2 
			WHEN userid2 = ? THEN userid1 
			ELSE nextuser 
		END
		WHERE id = ?`
	_, err = s.db.Exec(updateQuery, userId, userId, gameId)
	if err != nil {
		return hit, err
	}

	return hit, nil
}

type ShipDto struct {
	GameId string
	UserId string
	X      int
	Y      int
}

type MoveDto struct {
	GameId string
	UserId string
	X      int
	Y      int
	Hit    bool
}

type GameDto struct {
	Id       string
	UserId1  string
	UserId2  string
	Created  string
	NextUser string
}

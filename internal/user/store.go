package user

import (
	"database/sql"
	"log"

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

	createTable := `
    CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY,
        name TEXT,
        email TEXT
    );`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatal("cannot create table:", err)
	}

	return &Store{db: db}
}

func (s *Store) CreateUser(id, name, email string) error {
	_, err := s.db.Exec("INSERT INTO users(id, name, email) VALUES (?, ?, ?)", id, name, email)
	return err
}

func (s *Store) GetUser(id string) (string, string, error) {
	row := s.db.QueryRow("SELECT name, email FROM users WHERE id = ?", id)
	var name, email string
	err := row.Scan(&name, &email)
	return name, email, err
}

func (s *Store) GetUsers() ([]UserDto, error) {
	rows, err := s.db.Query("SELECT id, name, email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserDto
	for rows.Next() {
		var u UserDto
		if err := rows.Scan(&u.Id, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

type UserDto struct {
	Id    string
	Name  string
	Email string
}

package migration

import (
	"context"
	"database/sql"
)

func CreateDb(dbName, dbPath string, ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "data/forum.db")
	if err != nil {
		return nil, err
	}
	// db.SetMaxIdleConns(100)
	if err = CreateAllTables(ctx, db); err != nil {
		return nil, err
	}
	return db, nil
}

func CreateAllTables(ctx context.Context, db *sql.DB) error {
	// Begin transaction
	trans, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			trans.Rollback() // Rollback on panic
			panic(p)
		} else if err != nil {
			trans.Rollback() // Rollback on error
		} else {
			err = trans.Commit() // Commit on success
		}
	}()

	// Create users table
	if _, err = trans.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            firstName TEXT,
			secondName TEXT,
			usernames TEXT UNIQUE,
			email TEXT UNIQUE,
			password TEXT,
			role TEXT
        );
    `); err != nil {
		return err
	}

	// Create sessions table
	if _, err = trans.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS sessions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER UNIQUE,
			token TEXT UNIQUE,
			exp_time DATE,
			FOREIGN KEY (user_id) REFERENCES users (id)
        );
    `); err != nil {
		return err
	}

	// Create posts table
	if _, err = trans.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			title TEXT, 
			content TEXT,
			created_time DATE,
			likes_counter INTEGER, 
			dislikes_counter INTEGER,
			image_path TEXT,
			is_approved INTEGER,
			reports INTEGER,
			report_category TEXT,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)
	`); err != nil {
		return err
	}

	// Create post_category table
	if _, err = trans.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS post_category(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER,
			category_name TEXT,
			FOREIGN KEY (post_id) REFERENCES posts (id)
		)
	`); err != nil {
		return err
	}

	// Create categories table
	if _, err = trans.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS categories(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category_name TEXT
		)
	`); err != nil {
		return err
	}

	// Create comments table
	if _, err = trans.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER,
			user_id INTEGER, 
			content TEXT,
			created_time DATE,
			likes_counter INTEGER, 
			dislikes_counter INTEGER,
			is_approved INTEGER,
			reports INTEGER,
			FOREIGN KEY (post_id) REFERENCES posts (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		)
	`); err != nil {
		return err
	}

	// Drop and recreate post_votes table with unique constraint
	if _, err = trans.ExecContext(ctx, `DROP TABLE IF EXISTS post_votes`); err != nil {
		return err
	}
	if _, err = trans.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS post_votes(
			post_votes_id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER,
			user_id INTEGER,
			reaction INTEGER,
			is_seen BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts (id),
			FOREIGN KEY (user_id) REFERENCES users (id),
			UNIQUE(post_id, user_id)
		)
	`); err != nil {
		return err
	}

	// Create comment_votes table
	if _, err = trans.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS comment_votes(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			comment_id INTEGER,
			user_id INTEGER,
			reaction INTEGER,
			FOREIGN KEY (comment_id) REFERENCES comments (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		)
	`); err != nil {
		return err
	}

	return nil
}

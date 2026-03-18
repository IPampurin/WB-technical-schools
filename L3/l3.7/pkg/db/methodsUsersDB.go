package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/auth"
	"github.com/IPampurin/WarehouseControlAntiPattern/pkg/domain"
	"github.com/jackc/pgx/v5"
)

// getUserIDFromContext извлекает id пользователя из контекста
func getUserIDFromContext(ctx context.Context) int {

	if userID, ok := ctx.Value(auth.UserIDKey).(int); ok {
		return userID
	}

	return 0
}

// setUserIDInTransaction устанавливает переменную сессии app.current_user_id
func setUserIDInTransaction(ctx context.Context, tx pgx.Tx, userID int) error {

	if userID == 0 {
		_, err := tx.Exec(ctx, "SET LOCAL app.current_user_id = NULL")
		return err
	}

	query := fmt.Sprintf("SET LOCAL app.current_user_id = %d", userID)
	_, err := tx.Exec(ctx, query)

	return err
}

// CreateUser добавляет нового пользователя
func (d *DataBase) CreateUser(ctx context.Context, user *domain.User) error {

	dbUser := domainUserToDBUser(user)

	query := `   INSERT INTO users (name, email, role, created_at) 
	             VALUES ($1, $2, $3, NOW())
			  RETURNING id, created_at`

	err := d.Pool.QueryRow(ctx, query, dbUser.Name, dbUser.Email, dbUser.Role).
		Scan(&dbUser.ID, &dbUser.CreatedAt)
	if err != nil {
		return fmt.Errorf("CreateUser: %w", err)
	}

	user.ID = dbUser.ID
	user.CreatedAt = dbUser.CreatedAt

	return nil
}

// GetUserByID возвращает пользователя по id
func (d *DataBase) GetUserByID(ctx context.Context, id int) (*domain.User, error) {

	dbUser := &User{}

	query := `SELECT id, name, email, role, created_at
	            FROM users
			   WHERE id = $1`

	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&dbUser.ID, &dbUser.Name, &dbUser.Email, &dbUser.Role, &dbUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetUserByID: %w", err)
	}

	return dbUserToDomainUser(dbUser), nil
}

// GetUserByEmail возвращает пользователя по email
func (d *DataBase) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {

	dbUser := &User{}

	query := `SELECT id, name, email, role, created_at
	            FROM users
			   WHERE email = $1`

	err := d.Pool.QueryRow(ctx, query, email).Scan(
		&dbUser.ID, &dbUser.Name, &dbUser.Email, &dbUser.Role, &dbUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetUserByEmail: %w", err)
	}

	return dbUserToDomainUser(dbUser), nil
}

// UpdateUser обновляет данные пользователя
func (d *DataBase) UpdateUser(ctx context.Context, user *domain.User) error {

	dbUser := domainUserToDBUser(user)

	query := `UPDATE users
	             SET name = $1, role = $2
			   WHERE id = $3`

	cmdTag, err := d.Pool.Exec(ctx, query, dbUser.Name, dbUser.Role, dbUser.ID)
	if err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("UpdateUser: пользователь не найден")
	}

	return nil
}

// DeleteUser удаляет пользователя по id
func (d *DataBase) DeleteUser(ctx context.Context, id int) error {

	query := `DELETE
	            FROM users
			   WHERE id = $1`

	cmdTag, err := d.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteUser: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("DeleteUser: пользователь не найден")
	}

	return nil
}

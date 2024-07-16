package database

import (
	"backend/lib"
	"fmt"
	"os"

	"github.com/google/uuid"
	surrealdb "github.com/surrealdb/surrealdb.go"
)

type Cache struct {
	Db *surrealdb.DB
}

func NewCache() (Cache, error) {
	address := os.Getenv("SURREALDB_ADDRESS")
	db, err := surrealdb.New(address)
	if err != nil {
		return Cache{}, err
	}

	return Cache{
		Db: db,
	}, nil
}

func (cache *Cache) Connect(password string) error {
	user := os.Getenv("SURREALDB_USER")
	namespace := os.Getenv("SURREALDB_NAMESPACE")
	database := os.Getenv("SURREALDB_DATABASE")

	if _, err := cache.Db.Signin(map[string]interface{}{
		"user": user,
		"pass": password,
	}); err != nil {
		return err
	}
	if _, err := cache.Db.Use(namespace, database); err != nil {
		return fmt.Errorf("failed to select namespace and database: %w", err)
	}
	return nil
}

func (cache *Cache) NewEngineUser(id string, password string) error {
	user := map[string]string{
		"user": id,
		"pass": password,
	}
	if _, err := cache.Db.Query("DEFINE USER $user SET PASSWORD $pass", user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (cache *Cache) AddEngine(engine lib.Engine) error {
	if engine.Id == "" {
		engine.Id = uuid.New().String()
	}
	_, err := cache.Db.Create("engine", engine)
	if err != nil {
		return fmt.Errorf("failed to add engine: %w", err)
	}
	return nil
}

func (cache *Cache) GetEngine(id string) (lib.Engine, error) {
	var engine lib.Engine
	resp, err := cache.Db.Select(fmt.Sprintf("engine:%s", id))
	if err != nil {
		return engine, fmt.Errorf("failed to get engine: %w", err)
	}
	if resp == nil {
		return engine, fmt.Errorf("engine with ID %s does not exist", id)
	}

	if err := surrealdb.Unmarshal(resp, &engine); err != nil {
		return engine, fmt.Errorf("failed to unmarshal engine data: %w", err)
	}

	return engine, nil
}

func (cache *Cache) UpdateEngine(engine lib.Engine) error {
	if engine.Id == "" {
		return fmt.Errorf("engine ID cannot be empty")
	}
	_, err := cache.Db.Update(fmt.Sprintf("engine:%s", engine.Id), engine)
	if err != nil {
		return fmt.Errorf("failed to update engine: %w", err)
	}
	return nil
}

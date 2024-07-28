package vault

import (
	"context"
	"fmt"
	"os"

	v "github.com/hashicorp/vault/api"
)

type Vault = v.Client

type VaultManager struct {
	Engines  *Vault
	Api      *Vault
	Services *Vault
}

func NewVaultManager() (VaultManager, error) {
	config := v.Config{
		Address: os.Getenv("VAULT_ADDR"),
	}

	engines, err := v.NewClient(&config)
	if err != nil {
		return VaultManager{}, fmt.Errorf("failed to create Vault client: %w", err)
	}

	api, err := v.NewClient(&config)
	if err != nil {
		return VaultManager{}, fmt.Errorf("failed to create Vault client: %w", err)
	}

	services, err := v.NewClient(&config)
	if err != nil {
		return VaultManager{}, fmt.Errorf("failed to create Vault client: %w", err)
	}

	vault_manager := VaultManager{
		Engines:  engines,
		Api:      api,
		Services: services,
	}
	return vault_manager, nil
}

func (manager *VaultManager) Health() bool {
	engine_health, err := manager.Engines.Sys().Health()
	if err != nil {
		return false
	}
	api_health, err := manager.Engines.Sys().Health()
	if err != nil {
		return false
	}
	services_health, err := manager.Engines.Sys().Health()
	if err != nil {
		return false
	}

	return (err == nil) &&
		(engine_health.Initialized && engine_health.Sealed) &&
		(api_health.Initialized && api_health.Sealed) &&
		(services_health.Initialized && services_health.Sealed)
}

func (manager *VaultManager) StoreEngineAESKey(id string, key string) error {
	secret := map[string]interface{}{
		"key": key,
	}
	kvv2 := manager.Engines.KVv2("engines")

	// Write the secret
	_, err := kvv2.Put(context.Background(), fmt.Sprintf("aes/%s", id), secret)
	if err != nil {
		return fmt.Errorf("failed to store key in Vault: %w", err)
	}

	if err != nil {
		return fmt.Errorf("failed to store key in Vault: %w", err)
	}
	return err
}

func (manager *VaultManager) GetEngineAESKey(id string) (string, error) {
	kvv2 := manager.Engines.KVv2("engines")
	path := fmt.Sprintf("aes/%s", id)

	secret, err := kvv2.Get(context.Background(), path)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve key from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no secret found at path: %s", path)
	}

	key, ok := secret.Data["key"].(string)
	if !ok {
		return "", fmt.Errorf("key not found or invalid in secret data at path: %s", path)
	}
	return key, nil
}

func (manager *VaultManager) GetCachePwd() (string, error) {
	secret, err := manager.Services.Logical().Read("services/data/cache/mcs_pwd")
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no secret found at path: services/data/cache")
	}
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid secret data format at path: services/data/cache")
	}
	key, ok := data["value"].(string)
	if !ok {
		return "", fmt.Errorf("key not found or invalid in secret data at path: services/data/cache")
	}
	return key, nil
}

func (manager *VaultManager) GetApiKey(name string) (string, error) {
	path := fmt.Sprintf("api/data/%s", name)
	secret, err := manager.Api.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no secret found at path: %s", path)
	}
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid secret data format at path: %s", path)
	}

	key, ok := data["value"].(string)
	if !ok {
		return "", fmt.Errorf("key not found or invalid in secret data at path: %s", path)
	}
	return key, nil
}

func (manager *VaultManager) GenPwd() (string, error) {
	secret, err := manager.Services.Logical().Read("password/generate")
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	password, ok := secret.Data["password"].(string)
	if !ok {
		return "", fmt.Errorf("password format is incorrect")
	}

	return password, nil
}

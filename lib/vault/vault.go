package vault

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	v "github.com/hashicorp/vault/api"
)

type Vault = v.Client

type VaultManager struct {
	NexusPool *Vault
	Api       *Vault
	Services  *Vault

	OpenNexusAESKey  string
	OpenNexusHMACKey string
	OpenAPIKey       map[string]string
}

func NewVaultManager() (VaultManager, error) {
	config := v.Config{
		Address: os.Getenv("VAULT_ADDR"),
	}

	nexuspool, err := v.NewClient(&config)
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

	var open_api_key = make(map[string]string, 8)

	vault_manager := VaultManager{
		NexusPool:       nexuspool,
		Api:             api,
		Services:        services,
		OpenNexusAESKey: "",
		OpenAPIKey:      open_api_key,
	}
	return vault_manager, nil
}

func (manager *VaultManager) Health() bool {
	nexuspool_health, err := manager.NexusPool.Sys().Health()
	if err != nil {
		return false
	}
	api_health, err := manager.Api.Sys().Health()
	if err != nil {
		return false
	}
	services_health, err := manager.Services.Sys().Health()
	if err != nil {
		return false
	}

	return (err == nil) &&
		(nexuspool_health.Initialized && nexuspool_health.Sealed) &&
		(api_health.Initialized && api_health.Sealed) &&
		(services_health.Initialized && services_health.Sealed)
}

func (manager *VaultManager) StoreNexusPoolAESKey(id string, key string) error {
	secret := map[string]interface{}{
		"key": key,
	}
	kvv2 := manager.NexusPool.KVv2("nexuspool")

	// Write the secret
	_, err := kvv2.Put(context.Background(), "aes", secret)
	if err != nil {
		return fmt.Errorf("failed to store key in Vault: %w", err)
	}
	manager.OpenNexusAESKey = key

	return err
}

func (manager *VaultManager) StoreNexusPoolHMACKey(id string, key string) error {
	secret := map[string]interface{}{
		"key": key,
	}
	kvv2 := manager.NexusPool.KVv2("nexuspool")

	// Write the secret
	_, err := kvv2.Put(context.Background(), "hmac", secret)
	if err != nil {
		return fmt.Errorf("failed to store key in Vault: %w", err)
	}
	manager.OpenNexusHMACKey = key

	return err
}

func (manager *VaultManager) GetNexusPoolAESKey(id string) (string, error) {
	kvv2 := manager.NexusPool.KVv2("nexuspool")
	path := "aes"

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

func (manager *VaultManager) GetDbPwd() (string, error) {
	secret, err := manager.Services.Logical().Read("services/data/db/mcs_pwd")
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no secret found at path: services/data/db")
	}
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid secret data format at path: services/data/db")
	}
	key, ok := data["value"].(string)
	if !ok {
		return "", fmt.Errorf("key not found or invalid in secret data at path: services/data/db")
	}
	return key, nil
}

func (manager *VaultManager) LoadApiKeys(names ...string) error {

	for _, name := range names {
		key, err := manager.getApiKey(name)
		if err != nil {
			return fmt.Errorf("failed to load API key for '%s': %w", name, err)
		}
		manager.OpenAPIKey[name] = key
	}
	slog.Info("All api keys were loaded")
	return nil
}

func (manager *VaultManager) getApiKey(name string) (string, error) {
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

func (manager *VaultManager) GetApiKey(name string) (string, error) {
	key, ok := manager.OpenAPIKey[name]
	if !ok {
		return "", fmt.Errorf("unknown api key")
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

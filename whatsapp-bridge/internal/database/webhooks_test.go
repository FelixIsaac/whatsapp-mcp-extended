package database

import (
	"database/sql"
	"os"
	"testing"
	"whatsapp-bridge/internal/types"
)

func TestUpdateWebhookConfig(t *testing.T) {
	// Create a temporary database for testing
	tempDB := "test_webhooks.db"
	defer os.Remove(tempDB)

	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create tables
	err = createTables(db)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	store := &MessageStore{db: db}

	// Create initial webhook config
	config := &types.WebhookConfig{
		Name:        "Test Webhook",
		WebhookURL:  "https://example.com/webhook",
		SecretToken: "secret123",
		Enabled:     true,
		Triggers: []types.WebhookTrigger{
			{
				TriggerType:  "keyword",
				TriggerValue: "test",
				MatchType:    "contains",
				Enabled:      true,
			},
		},
	}

	// Store the initial config
	err = store.StoreWebhookConfig(config)
	if err != nil {
		t.Fatalf("Failed to store initial webhook config: %v", err)
	}

	initialID := config.ID
	if initialID == 0 {
		t.Fatal("Config ID should be set after storing")
	}

	// Update the config with new triggers
	config.Name = "Updated Test Webhook"
	config.WebhookURL = "https://example.com/updated"
	config.SecretToken = "newsecret456"
	config.Triggers = []types.WebhookTrigger{
		{
			TriggerType:  "keyword",
			TriggerValue: "urgent",
			MatchType:    "contains",
			Enabled:      true,
		},
		{
			TriggerType:  "sender",
			TriggerValue: "123456@s.whatsapp.net",
			MatchType:    "exact",
			Enabled:      true,
		},
	}

	// Update the config
	err = store.UpdateWebhookConfig(config)
	if err != nil {
		t.Fatalf("Failed to update webhook config: %v", err)
	}

	// Retrieve the updated config
	updatedConfig, err := store.GetWebhookConfig(initialID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated webhook config: %v", err)
	}

	// Verify the main config was updated
	if updatedConfig.Name != "Updated Test Webhook" {
		t.Errorf("Expected name 'Updated Test Webhook', got '%s'", updatedConfig.Name)
	}
	if updatedConfig.WebhookURL != "https://example.com/updated" {
		t.Errorf("Expected URL 'https://example.com/updated', got '%s'", updatedConfig.WebhookURL)
	}
	if updatedConfig.SecretToken != "newsecret456" {
		t.Errorf("Expected secret 'newsecret456', got '%s'", updatedConfig.SecretToken)
	}

	// Verify the triggers were updated
	if len(updatedConfig.Triggers) != 2 {
		t.Errorf("Expected 2 triggers, got %d", len(updatedConfig.Triggers))
	}

	// Check first trigger
	if updatedConfig.Triggers[0].TriggerValue != "urgent" {
		t.Errorf("Expected first trigger value 'urgent', got '%s'", updatedConfig.Triggers[0].TriggerValue)
	}

	// Check second trigger
	if updatedConfig.Triggers[1].TriggerType != "sender" {
		t.Errorf("Expected second trigger type 'sender', got '%s'", updatedConfig.Triggers[1].TriggerType)
	}
	if updatedConfig.Triggers[1].TriggerValue != "123456@s.whatsapp.net" {
		t.Errorf("Expected second trigger value '123456@s.whatsapp.net', got '%s'", updatedConfig.Triggers[1].TriggerValue)
	}

	// Verify all triggers have proper IDs
	for i, trigger := range updatedConfig.Triggers {
		if trigger.ID == 0 {
			t.Errorf("Trigger %d should have a valid ID", i)
		}
		if trigger.WebhookConfigID != initialID {
			t.Errorf("Trigger %d should have webhook_config_id %d, got %d", i, initialID, trigger.WebhookConfigID)
		}
	}

	t.Log("✓ Webhook update test passed successfully")
}

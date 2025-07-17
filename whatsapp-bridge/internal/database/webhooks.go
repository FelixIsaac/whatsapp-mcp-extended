package database

import (
	"fmt"
	"whatsapp-bridge/internal/types"
)

// StoreWebhookConfig stores a webhook configuration in the database
func (store *MessageStore) StoreWebhookConfig(config *types.WebhookConfig) error {
	result, err := store.db.Exec(
		`INSERT INTO webhook_configs (name, webhook_url, secret_token, enabled) 
		 VALUES (?, ?, ?, ?)`,
		config.Name, config.WebhookURL, config.SecretToken, config.Enabled,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	config.ID = int(id)

	// Store triggers
	for i := range config.Triggers {
		config.Triggers[i].WebhookConfigID = config.ID
		err = store.StoreWebhookTrigger(&config.Triggers[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// GetWebhookConfig retrieves a webhook configuration by ID
func (store *MessageStore) GetWebhookConfig(id int) (*types.WebhookConfig, error) {
	config := &types.WebhookConfig{}
	err := store.db.QueryRow(
		`SELECT id, name, webhook_url, secret_token, enabled, created_at, updated_at 
		 FROM webhook_configs WHERE id = ?`, id,
	).Scan(&config.ID, &config.Name, &config.WebhookURL, &config.SecretToken,
		&config.Enabled, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Load triggers
	config.Triggers, err = store.GetWebhookTriggers(id)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetAllWebhookConfigs retrieves all webhook configurations
func (store *MessageStore) GetAllWebhookConfigs() ([]*types.WebhookConfig, error) {
	rows, err := store.db.Query(
		`SELECT id, name, webhook_url, secret_token, enabled, created_at, updated_at 
		 FROM webhook_configs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*types.WebhookConfig
	for rows.Next() {
		config := &types.WebhookConfig{}
		err := rows.Scan(&config.ID, &config.Name, &config.WebhookURL, &config.SecretToken,
			&config.Enabled, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Load triggers for each config
		config.Triggers, err = store.GetWebhookTriggers(config.ID)
		if err != nil {
			return nil, err
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// UpdateWebhookConfig updates a webhook configuration
func (store *MessageStore) UpdateWebhookConfig(config *types.WebhookConfig) error {
	_, err := store.db.Exec(
		`UPDATE webhook_configs SET name = ?, webhook_url = ?, secret_token = ?, 
		 enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		config.Name, config.WebhookURL, config.SecretToken, config.Enabled, config.ID,
	)
	return err
}

// DeleteWebhookConfig deletes a webhook configuration and its triggers and logs
func (store *MessageStore) DeleteWebhookConfig(id int) error {
	// First check if the webhook exists
	var count int
	err := store.db.QueryRow("SELECT COUNT(*) FROM webhook_configs WHERE id = ?", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("webhook with ID %d not found", id)
	}

	// Delete webhook logs first (foreign key constraint)
	_, err = store.db.Exec("DELETE FROM webhook_logs WHERE webhook_config_id = ?", id)
	if err != nil {
		return err
	}

	// Delete triggers second (foreign key constraint)
	_, err = store.db.Exec("DELETE FROM webhook_triggers WHERE webhook_config_id = ?", id)
	if err != nil {
		return err
	}

	// Delete config last
	_, err = store.db.Exec("DELETE FROM webhook_configs WHERE id = ?", id)
	return err
}

// StoreWebhookTrigger stores a webhook trigger
func (store *MessageStore) StoreWebhookTrigger(trigger *types.WebhookTrigger) error {
	result, err := store.db.Exec(
		`INSERT INTO webhook_triggers (webhook_config_id, trigger_type, trigger_value, match_type, enabled) 
		 VALUES (?, ?, ?, ?, ?)`,
		trigger.WebhookConfigID, trigger.TriggerType, trigger.TriggerValue, trigger.MatchType, trigger.Enabled,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	trigger.ID = int(id)

	return nil
}

// GetWebhookTriggers retrieves all triggers for a webhook config
func (store *MessageStore) GetWebhookTriggers(webhookConfigID int) ([]types.WebhookTrigger, error) {
	rows, err := store.db.Query(
		`SELECT id, webhook_config_id, trigger_type, trigger_value, match_type, enabled 
		 FROM webhook_triggers WHERE webhook_config_id = ?`, webhookConfigID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []types.WebhookTrigger
	for rows.Next() {
		trigger := types.WebhookTrigger{}
		err := rows.Scan(&trigger.ID, &trigger.WebhookConfigID, &trigger.TriggerType,
			&trigger.TriggerValue, &trigger.MatchType, &trigger.Enabled)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, trigger)
	}

	return triggers, nil
}

// DeleteWebhookTrigger deletes a webhook trigger
func (store *MessageStore) DeleteWebhookTrigger(id int) error {
	_, err := store.db.Exec("DELETE FROM webhook_triggers WHERE id = ?", id)
	return err
}

// StoreWebhookLog stores a webhook delivery log
func (store *MessageStore) StoreWebhookLog(log *types.WebhookLog) error {
	_, err := store.db.Exec(
		`INSERT INTO webhook_logs (webhook_config_id, message_id, chat_jid, trigger_type, trigger_value, 
		 payload, response_status, response_body, attempt_count, delivered_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.WebhookConfigID, log.MessageID, log.ChatJID, log.TriggerType, log.TriggerValue,
		log.Payload, log.ResponseStatus, log.ResponseBody, log.AttemptCount, log.DeliveredAt,
	)
	return err
}

// GetWebhookLogs retrieves webhook logs with optional filtering
func (store *MessageStore) GetWebhookLogs(webhookConfigID int, limit int) ([]*types.WebhookLog, error) {
	query := `SELECT id, webhook_config_id, message_id, chat_jid, trigger_type, trigger_value, 
		 payload, response_status, response_body, attempt_count, delivered_at, created_at 
		 FROM webhook_logs`

	var args []interface{}
	if webhookConfigID > 0 {
		query += " WHERE webhook_config_id = ?"
		args = append(args, webhookConfigID)
	}

	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*types.WebhookLog
	for rows.Next() {
		log := &types.WebhookLog{}
		err := rows.Scan(&log.ID, &log.WebhookConfigID, &log.MessageID, &log.ChatJID,
			&log.TriggerType, &log.TriggerValue, &log.Payload, &log.ResponseStatus,
			&log.ResponseBody, &log.AttemptCount, &log.DeliveredAt, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

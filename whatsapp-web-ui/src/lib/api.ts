const getApiBaseUrl = () => {
  if (typeof window === "undefined") return "http://localhost:8180/api";
  return `${window.location.protocol}//${window.location.hostname}:8180/api`;
};

// Pairing types
export interface PairResponse {
  success: boolean;
  code?: string;
  expires_in?: number;
  error?: string;
}

export interface PairingStatusResponse {
  success: boolean;
  in_progress: boolean;
  code?: string;
  expires_in?: number;
  complete: boolean;
  error?: string;
}

export interface ConnectionStatusResponse {
  success: boolean;
  connected: boolean;
  linked: boolean;
  jid?: string;
  uptime?: string;
  last_connected?: string;
  disconnected_for?: string;
  auto_reconnect_errors?: number;
}

export interface SyncStatusResponse {
  success: boolean;
  syncing: boolean;
  last_sync?: string;
  sync_progress: number;
  message_count: number;
  conversation_count: number;
  error?: string;
  recommendations?: string[];
}

// Webhook types
export interface WebhookTrigger {
  trigger_type: "all" | "chat_jid" | "sender" | "keyword" | "media_type";
  trigger_value: string;
  match_type: "exact" | "contains" | "regex";
  enabled: boolean;
}

export interface Webhook {
  id: string;
  name: string;
  webhook_url: string;
  secret_token?: string;
  enabled: boolean;
  triggers: WebhookTrigger[];
  created_at: string;
  updated_at?: string;
}

export interface WebhookLog {
  id: string;
  webhook_id: string;
  message_id: string;
  chat_jid: string;
  trigger_type: string;
  trigger_value: string;
  payload: string;
  response_status?: number;
  response_body?: string;
  delivered_at?: string;
  created_at: string;
  attempt_count: number;
}

export interface WebhooksResponse {
  success: boolean;
  data?: Webhook[];
  error?: string;
}

export interface WebhookResponse {
  success: boolean;
  data?: Webhook;
  error?: string;
}

export interface WebhookLogsResponse {
  success: boolean;
  data?: WebhookLog[];
  error?: string;
}

export class WhatsAppAPI {
  private baseUrl: string;
  private apiKey: string;

  constructor(apiKey: string) {
    this.baseUrl = getApiBaseUrl();
    this.apiKey = apiKey;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        "X-API-Key": this.apiKey,
        ...options.headers,
      },
    });

    const data = await response.json();

    if (!response.ok) {
      throw new APIError(response.status, data.error || "Request failed");
    }

    return data;
  }

  // Pairing methods
  async pair(phoneNumber: string): Promise<PairResponse> {
    return this.request<PairResponse>("/pair", {
      method: "POST",
      body: JSON.stringify({ phone_number: phoneNumber }),
    });
  }

  async getPairingStatus(): Promise<PairingStatusResponse> {
    return this.request<PairingStatusResponse>("/pairing");
  }

  async getConnectionStatus(): Promise<ConnectionStatusResponse> {
    return this.request<ConnectionStatusResponse>("/connection");
  }

  async getSyncStatus(): Promise<SyncStatusResponse> {
    return this.request<SyncStatusResponse>("/sync-status");
  }

  async reconnect(): Promise<void> {
    await this.request("/reconnect", { method: "POST" });
  }

  // Webhook methods
  async getWebhooks(): Promise<Webhook[]> {
    const response = await this.request<WebhooksResponse>("/webhooks");
    return response.data || [];
  }

  async createWebhook(webhook: Omit<Webhook, "id" | "created_at" | "updated_at">): Promise<Webhook> {
    const response = await this.request<WebhookResponse>("/webhooks", {
      method: "POST",
      body: JSON.stringify(webhook),
    });
    return response.data!;
  }

  async updateWebhook(id: string, webhook: Partial<Webhook>): Promise<Webhook> {
    const response = await this.request<WebhookResponse>(`/webhooks/${id}`, {
      method: "PUT",
      body: JSON.stringify(webhook),
    });
    return response.data!;
  }

  async deleteWebhook(id: string): Promise<void> {
    await this.request(`/webhooks/${id}`, { method: "DELETE" });
  }

  async toggleWebhook(id: string, enabled: boolean): Promise<void> {
    await this.request(`/webhooks/${id}/enable`, {
      method: "POST",
      body: JSON.stringify({ enabled }),
    });
  }

  async testWebhook(id: string): Promise<void> {
    await this.request(`/webhooks/${id}/test`, { method: "POST" });
  }

  async getWebhookLogs(id: string): Promise<WebhookLog[]> {
    const response = await this.request<WebhookLogsResponse>(`/webhooks/${id}/logs`);
    return response.data || [];
  }
}

export class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "APIError";
  }
}

export const getErrorMessage = (error: unknown): { title: string; description: string; action?: string } => {
  if (error instanceof APIError) {
    switch (error.status) {
      case 401:
        return {
          title: "Unauthorized",
          description: "Your API key is invalid or expired.",
          action: "Check your API key in Settings",
        };
      case 404:
        return {
          title: "Bridge Not Found",
          description: "Cannot connect to the WhatsApp bridge.",
          action: "Verify the bridge is running",
        };
      case 429:
        return {
          title: "Rate Limited",
          description: "Too many requests. Please wait.",
          action: "Will retry automatically...",
        };
      default:
        return {
          title: `Error ${error.status}`,
          description: error.message,
        };
    }
  }

  if (error instanceof TypeError && error.message.includes("fetch")) {
    return {
      title: "Network Error",
      description: "Cannot reach the WhatsApp bridge.",
      action: "Check your connection",
    };
  }

  return {
    title: "Unknown Error",
    description: error instanceof Error ? error.message : "An unexpected error occurred",
  };
};

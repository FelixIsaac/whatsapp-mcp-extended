const getApiBaseUrl = () => {
  if (typeof window === "undefined") return "http://localhost:8180/api";
  return `${window.location.protocol}//${window.location.hostname}:8180/api`;
};

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
}

export interface SyncStatusResponse {
  success: boolean;
  syncing: boolean;
  lastSync?: string;
  syncProgress: number;
  messageCount: number;
  conversationCount: number;
  error?: string;
  recommendations?: string[];
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

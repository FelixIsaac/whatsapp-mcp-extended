"use client";

import { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { CheckCircle, RefreshCw, Settings, Plus, MessageSquare, Clock, AlertTriangle, Loader2, XCircle, WifiOff, Zap } from "lucide-react";
import { WhatsAppAPI, SyncStatusResponse, ConnectionStatusResponse } from "@/lib/api";
import { useSettings, usePairing } from "@/lib/store";
import { cn } from "@/lib/utils";

interface DashboardProps {
  onOpenSettings: () => void;
}

export function Dashboard({ onOpenSettings }: DashboardProps) {
  const { apiKey } = useSettings();
  const { jid, reset } = usePairing();
  const [syncStatus, setSyncStatus] = useState<SyncStatusResponse | null>(null);
  const [connStatus, setConnStatus] = useState<ConnectionStatusResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [reconnecting, setReconnecting] = useState(false);

  const fetchStatus = useCallback(async () => {
    try {
      const api = new WhatsAppAPI(apiKey);
      const [sync, conn] = await Promise.all([
        api.getSyncStatus(),
        api.getConnectionStatus(),
      ]);
      setSyncStatus(sync);
      setConnStatus(conn);
    } catch (error) {
      console.error("Failed to fetch status:", error);
    } finally {
      setLoading(false);
    }
  }, [apiKey]);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  const handleRefresh = () => {
    setLoading(true);
    fetchStatus();
  };

  const handleReconnect = async () => {
    setReconnecting(true);
    try {
      const api = new WhatsAppAPI(apiKey);
      await api.reconnect();
    } catch (error) {
      console.error("Failed to reconnect:", error);
    } finally {
      setTimeout(() => {
        setReconnecting(false);
        fetchStatus();
      }, 3000);
    }
  };

  const isConnected = connStatus?.connected ?? true;
  const isDisconnected = connStatus !== null && !connStatus.connected;
  const hasReconnectErrors = (connStatus?.auto_reconnect_errors ?? 0) > 0;

  return (
    <Card className="w-full max-w-lg mx-auto">
      <CardHeader className="text-center">
        <CardTitle className="flex items-center justify-center gap-2">
          {isConnected ? (
            <>
              <CheckCircle className="h-6 w-6 text-green-500" />
              Connected
            </>
          ) : hasReconnectErrors ? (
            <>
              <Loader2 className="h-6 w-6 text-yellow-500 animate-spin" />
              Reconnecting...
            </>
          ) : (
            <>
              <XCircle className="h-6 w-6 text-red-500" />
              Disconnected
            </>
          )}
        </CardTitle>
        <CardDescription className="font-mono text-xs break-all">{jid}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Connection details when disconnected */}
        {isDisconnected && (
          <Card className={cn(
            "border-red-500/50 bg-red-500/10",
            hasReconnectErrors && "border-yellow-500/50 bg-yellow-500/10"
          )}>
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                <WifiOff className={cn("h-4 w-4", hasReconnectErrors ? "text-yellow-500" : "text-red-500")} />
                <span className="text-sm font-medium">Connection Lost</span>
              </div>
              <div className="text-sm text-muted-foreground space-y-1">
                {connStatus?.disconnected_for && (
                  <p>Disconnected for: <span className="font-mono font-medium text-foreground">{connStatus.disconnected_for}</span></p>
                )}
                {connStatus?.last_connected && (
                  <p>Last connected: <span className="font-mono text-foreground">{new Date(connStatus.last_connected).toLocaleString()}</span></p>
                )}
                {hasReconnectErrors && (
                  <p>Reconnect attempts: <span className="font-mono text-foreground">{connStatus?.auto_reconnect_errors}</span></p>
                )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Uptime when connected */}
        {isConnected && connStatus?.uptime && (
          <div className="text-center text-sm text-muted-foreground">
            Uptime: <span className="font-mono">{connStatus.uptime}</span>
            {connStatus.last_connected && (
              <> &middot; Connected since: <span className="font-mono">{new Date(connStatus.last_connected).toLocaleTimeString()}</span></>
            )}
          </div>
        )}

        <div className="grid grid-cols-2 gap-4">
          <Card className="bg-muted/50">
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                {syncStatus?.syncing ? (
                  <Loader2 className="h-4 w-4 animate-spin text-yellow-500" />
                ) : (
                  <CheckCircle className="h-4 w-4 text-green-500" />
                )}
                <span className="text-sm font-medium">Sync Status</span>
              </div>
              <div className="text-2xl font-bold">
                {syncStatus?.syncing ? "Syncing" : "Synced"}
              </div>
              {syncStatus && (
                <Progress value={syncStatus.sync_progress} className="mt-2 h-2" />
              )}
            </CardContent>
          </Card>

          <Card className="bg-muted/50">
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Last Sync</span>
              </div>
              <div className="text-lg font-medium">
                {syncStatus?.last_sync
                  ? new Date(syncStatus.last_sync).toLocaleString()
                  : "In progress..."}
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Card className="bg-muted/50">
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                <MessageSquare className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Messages</span>
              </div>
              <div className="text-2xl font-bold">
                {syncStatus?.message_count?.toLocaleString() || "0"}
              </div>
            </CardContent>
          </Card>

          <Card className="bg-muted/50">
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                <MessageSquare className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Chats</span>
              </div>
              <div className="text-2xl font-bold">
                {syncStatus?.conversation_count?.toLocaleString() || "0"}
              </div>
            </CardContent>
          </Card>
        </div>

        {syncStatus?.recommendations && syncStatus.recommendations.length > 0 && (
          <Card className="border-yellow-500/50 bg-yellow-500/10">
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                <AlertTriangle className="h-4 w-4 text-yellow-500" />
                <span className="text-sm font-medium">Recommendations</span>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                {syncStatus.recommendations.map((rec, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <span className="text-yellow-500">•</span>
                    {rec}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        )}

        {syncStatus?.error && (
          <Card className="border-destructive/50 bg-destructive/10">
            <CardContent className="pt-4">
              <div className="flex items-center gap-2 mb-2">
                <AlertTriangle className="h-4 w-4 text-destructive" />
                <span className="text-sm font-medium">Error</span>
              </div>
              <p className="text-sm text-muted-foreground">{syncStatus.error}</p>
            </CardContent>
          </Card>
        )}

        <Separator />

        <div className="flex gap-2 justify-center flex-wrap">
          {isDisconnected && (
            <Button variant="destructive" size="sm" onClick={handleReconnect} disabled={reconnecting}>
              <Zap className={"h-4 w-4 mr-2 " + (reconnecting ? "animate-pulse" : "")} />
              {reconnecting ? "Reconnecting..." : "Force Reconnect"}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={handleRefresh} disabled={loading}>
            <RefreshCw className={"h-4 w-4 mr-2 " + (loading ? "animate-spin" : "")} />
            Refresh
          </Button>
          <Button variant="outline" size="sm" onClick={onOpenSettings}>
            <Settings className="h-4 w-4 mr-2" />
            Settings
          </Button>
          <Button variant="outline" size="sm" onClick={reset}>
            <Plus className="h-4 w-4 mr-2" />
            New Device
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

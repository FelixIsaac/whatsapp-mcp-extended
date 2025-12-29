"use client";

import { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { CheckCircle, RefreshCw, Settings, Plus, MessageSquare, Clock, AlertTriangle, Loader2 } from "lucide-react";
import { WhatsAppAPI, SyncStatusResponse } from "@/lib/api";
import { useSettings, usePairing } from "@/lib/store";

interface DashboardProps {
  onOpenSettings: () => void;
}

export function Dashboard({ onOpenSettings }: DashboardProps) {
  const { apiKey } = useSettings();
  const { jid, reset } = usePairing();
  const [syncStatus, setSyncStatus] = useState<SyncStatusResponse | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchSyncStatus = useCallback(async () => {
    try {
      const api = new WhatsAppAPI(apiKey);
      const status = await api.getSyncStatus();
      setSyncStatus(status);
    } catch (error) {
      console.error("Failed to fetch sync status:", error);
    } finally {
      setLoading(false);
    }
  }, [apiKey]);

  useEffect(() => {
    fetchSyncStatus();
    const interval = setInterval(fetchSyncStatus, 3000);
    return () => clearInterval(interval);
  }, [fetchSyncStatus]);

  const handleRefresh = () => {
    setLoading(true);
    fetchSyncStatus();
  };

  return (
    <Card className="w-full max-w-lg mx-auto">
      <CardHeader className="text-center">
        <CardTitle className="flex items-center justify-center gap-2">
          <CheckCircle className="h-6 w-6 text-green-500" />
          Connected
        </CardTitle>
        <CardDescription className="font-mono text-xs break-all">{jid}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
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
                <Progress value={syncStatus.syncProgress} className="mt-2 h-2" />
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
                {syncStatus?.lastSync || "In progress..."}
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
                {syncStatus?.messageCount?.toLocaleString() || "0"}
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
                {syncStatus?.conversationCount?.toLocaleString() || "0"}
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

        <div className="flex gap-2 justify-center">
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

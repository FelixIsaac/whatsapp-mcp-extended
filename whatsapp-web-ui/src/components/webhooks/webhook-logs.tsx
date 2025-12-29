"use client";

import { useState, useEffect } from "react";
import { WebhookLog, WhatsAppAPI, getErrorMessage } from "@/lib/api";
import { useSettings } from "@/lib/store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { RefreshCw, CheckCircle, XCircle, Clock, Loader2 } from "lucide-react";
import { toast } from "sonner";

interface WebhookLogsProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  webhookId: string | null;
  webhookName: string;
}

export function WebhookLogs({ open, onOpenChange, webhookId, webhookName }: WebhookLogsProps) {
  const { apiKey } = useSettings();
  const [logs, setLogs] = useState<WebhookLog[]>([]);
  const [loading, setLoading] = useState(false);

  const loadLogs = async () => {
    if (!webhookId) return;

    setLoading(true);
    try {
      const api = new WhatsAppAPI(apiKey);
      const data = await api.getWebhookLogs(webhookId);
      setLogs(data.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()));
    } catch (error) {
      const { title, description } = getErrorMessage(error);
      toast.error(title, { description });
      setLogs([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open && webhookId) {
      loadLogs();
    }
  }, [open, webhookId]);

  const getStatusInfo = (log: WebhookLog) => {
    if (log.response_status) {
      if (log.response_status >= 200 && log.response_status < 300) {
        return { status: "success", icon: CheckCircle, label: "SUCCESS" };
      }
      return { status: "error", icon: XCircle, label: "ERROR" };
    }
    if (log.delivered_at) {
      return { status: "success", icon: CheckCircle, label: "DELIVERED" };
    }
    return { status: "pending", icon: Clock, label: "PENDING" };
  };

  const formatDate = (dateString: string) => {
    try {
      const date = new Date(dateString);
      return date.toLocaleDateString() + " " + date.toLocaleTimeString();
    } catch {
      return "Invalid date";
    }
  };

  const parsePayload = (payload: string) => {
    try {
      return JSON.parse(payload);
    } catch {
      return null;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[85vh]">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle>Webhook Logs</DialogTitle>
              <DialogDescription>{webhookName}</DialogDescription>
            </div>
            <Button variant="outline" size="sm" onClick={loadLogs} disabled={loading}>
              <RefreshCw className={"h-4 w-4 mr-1" + (loading ? " animate-spin" : "")} />
              Refresh
            </Button>
          </div>
        </DialogHeader>

        <div className="flex items-center justify-between text-sm text-muted-foreground border-b pb-2">
          <span>{logs.length} log {logs.length === 1 ? "entry" : "entries"}</span>
        </div>

        {loading && logs.length === 0 ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <Clock className="h-12 w-12 mb-4" />
            <p>No logs yet</p>
            <p className="text-sm">Logs will appear here when messages trigger this webhook</p>
          </div>
        ) : (
          <ScrollArea className="h-[500px] pr-4">
            <div className="space-y-3">
              {logs.map((log) => {
                const statusInfo = getStatusInfo(log);
                const StatusIcon = statusInfo.icon;
                const payload = parsePayload(log.payload);
                const chatName = payload?.message?.chat_name;
                const processingTime = payload?.metadata?.processing_time_ms || 0;

                const borderClass = statusInfo.status === "error"
                  ? "border-destructive/50 bg-destructive/5"
                  : statusInfo.status === "success"
                    ? "border-green-500/50 bg-green-500/5"
                    : "border-yellow-500/50 bg-yellow-500/5";

                const iconClass = statusInfo.status === "error"
                  ? "text-destructive"
                  : statusInfo.status === "success"
                    ? "text-green-500"
                    : "text-yellow-500";

                return (
                  <div key={log.id} className={"border rounded-lg p-4 space-y-2 " + borderClass}>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <StatusIcon className={"h-4 w-4 " + iconClass} />
                        <Badge variant={statusInfo.status === "error" ? "destructive" : "secondary"}>
                          {statusInfo.label}
                        </Badge>
                        <span className="text-sm text-muted-foreground">
                          Attempt #{log.attempt_count || 1}
                        </span>
                      </div>
                      <span className="text-sm text-muted-foreground">{formatDate(log.created_at)}</span>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-sm">
                      <div>
                        <span className="text-muted-foreground">Trigger: </span>
                        <span>{log.trigger_type}: {log.trigger_value || "all"}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Message ID: </span>
                        <span className="font-mono text-xs">{log.message_id || "N/A"}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Chat: </span>
                        <span>{chatName ? chatName + " (" + log.chat_jid + ")" : log.chat_jid || "N/A"}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Processing: </span>
                        <span>{processingTime}ms</span>
                      </div>
                    </div>

                    {log.response_status && (
                      <div className="text-sm">
                        <span className="text-muted-foreground">Response: </span>
                        <span>HTTP {log.response_status}</span>
                        {log.response_body && (
                          <span className="text-muted-foreground ml-2">
                            - {log.response_body.substring(0, 100)}{log.response_body.length > 100 ? "..." : ""}
                          </span>
                        )}
                      </div>
                    )}

                    {statusInfo.status === "error" && log.response_body && (
                      <div className="text-sm p-2 bg-destructive/10 rounded text-destructive">
                        {log.response_body}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </ScrollArea>
        )}
      </DialogContent>
    </Dialog>
  );
}

"use client";

import { useState, useEffect } from "react";
import { Webhook, WhatsAppAPI, getErrorMessage } from "@/lib/api";
import { useSettings } from "@/lib/store";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { WebhookCard } from "./webhook-card";
import { WebhookForm, WebhookFormData } from "./webhook-form";
import { WebhookLogs } from "./webhook-logs";
import { Plus, RefreshCw, Loader2, Webhook as WebhookIcon } from "lucide-react";
import { toast } from "sonner";

export function WebhookList() {
  const { apiKey } = useSettings();
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<Webhook | null>(null);
  const [logsOpen, setLogsOpen] = useState(false);
  const [logsWebhook, setLogsWebhook] = useState<{ id: string; name: string } | null>(null);
  const [deleteDialog, setDeleteDialog] = useState<{ id: string; name: string } | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const loadWebhooks = async () => {
    setLoading(true);
    try {
      const api = new WhatsAppAPI(apiKey);
      const data = await api.getWebhooks();
      setWebhooks(data);
    } catch (error) {
      const { title, description } = getErrorMessage(error);
      toast.error(title, { description });
      setWebhooks([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadWebhooks();
  }, [apiKey]);

  const handleCreate = () => {
    setEditingWebhook(null);
    setFormOpen(true);
  };

  const handleEdit = (webhook: Webhook) => {
    setEditingWebhook(webhook);
    setFormOpen(true);
  };

  const handleFormSubmit = async (data: WebhookFormData) => {
    setSubmitting(true);
    try {
      const api = new WhatsAppAPI(apiKey);
      if (editingWebhook) {
        await api.updateWebhook(editingWebhook.id, data);
        toast.success("Webhook updated");
      } else {
        await api.createWebhook(data);
        toast.success("Webhook created");
      }
      setFormOpen(false);
      loadWebhooks();
    } catch (error) {
      const { title, description } = getErrorMessage(error);
      toast.error(title, { description });
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggle = async (id: string, enabled: boolean) => {
    try {
      const api = new WhatsAppAPI(apiKey);
      await api.toggleWebhook(id, enabled);
      toast.success(enabled ? "Webhook enabled" : "Webhook disabled");
      loadWebhooks();
    } catch (error) {
      const { title, description } = getErrorMessage(error);
      toast.error(title, { description });
    }
  };

  const handleTest = async (id: string) => {
    try {
      const api = new WhatsAppAPI(apiKey);
      await api.testWebhook(id);
      toast.success("Test sent successfully");
    } catch (error) {
      const { title, description } = getErrorMessage(error);
      toast.error(title, { description });
    }
  };

  const handleLogs = (id: string) => {
    const webhook = webhooks.find((w) => w.id === id);
    setLogsWebhook({ id, name: webhook?.name || "Webhook" });
    setLogsOpen(true);
  };

  const handleDelete = async () => {
    if (!deleteDialog) return;

    try {
      const api = new WhatsAppAPI(apiKey);
      await api.deleteWebhook(deleteDialog.id);
      toast.success("Webhook deleted");
      setDeleteDialog(null);
      loadWebhooks();
    } catch (error) {
      const { title, description } = getErrorMessage(error);
      toast.error(title, { description });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Webhooks</h2>
          <p className="text-muted-foreground">
            Manage webhook endpoints for receiving WhatsApp messages
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadWebhooks} disabled={loading}>
            <RefreshCw className={"h-4 w-4 mr-2" + (loading ? " animate-spin" : "")} />
            Refresh
          </Button>
          <Button onClick={handleCreate}>
            <Plus className="h-4 w-4 mr-2" />
            Add Webhook
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : webhooks.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground border rounded-lg bg-muted/30">
          <WebhookIcon className="h-16 w-16 mb-4" />
          <h3 className="text-lg font-medium mb-2">No webhooks configured</h3>
          <p className="text-sm mb-4">Create your first webhook to start receiving messages</p>
          <Button onClick={handleCreate}>
            <Plus className="h-4 w-4 mr-2" />
            Create Webhook
          </Button>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {webhooks.map((webhook) => (
            <WebhookCard
              key={webhook.id}
              webhook={webhook}
              onTest={handleTest}
              onLogs={handleLogs}
              onEdit={handleEdit}
              onToggle={handleToggle}
              onDelete={(id, name) => setDeleteDialog({ id, name })}
            />
          ))}
        </div>
      )}

      <WebhookForm
        open={formOpen}
        onOpenChange={setFormOpen}
        webhook={editingWebhook}
        onSubmit={handleFormSubmit}
        isLoading={submitting}
      />

      <WebhookLogs
        open={logsOpen}
        onOpenChange={setLogsOpen}
        webhookId={logsWebhook?.id || null}
        webhookName={logsWebhook?.name || ""}
      />

      <AlertDialog open={!!deleteDialog} onOpenChange={() => setDeleteDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Webhook</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deleteDialog?.name}"? This will also delete all webhook logs and cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

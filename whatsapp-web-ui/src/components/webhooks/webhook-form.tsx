"use client";

import { useState, useEffect } from "react";
import { Webhook, WebhookTrigger } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { TriggerBuilder } from "./trigger-builder";

interface WebhookFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  webhook?: Webhook | null;
  onSubmit: (data: WebhookFormData) => void;
  isLoading?: boolean;
}

export interface WebhookFormData {
  name: string;
  webhook_url: string;
  secret_token: string;
  enabled: boolean;
  triggers: WebhookTrigger[];
}

const defaultTrigger: WebhookTrigger = {
  trigger_type: "all",
  trigger_value: "",
  match_type: "exact",
  enabled: true,
};

export function WebhookForm({ open, onOpenChange, webhook, onSubmit, isLoading }: WebhookFormProps) {
  const [formData, setFormData] = useState<WebhookFormData>({
    name: "",
    webhook_url: "",
    secret_token: "",
    enabled: true,
    triggers: [defaultTrigger],
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (webhook) {
      setFormData({
        name: webhook.name,
        webhook_url: webhook.webhook_url,
        secret_token: webhook.secret_token || "",
        enabled: webhook.enabled,
        triggers: webhook.triggers?.length ? webhook.triggers : [defaultTrigger],
      });
    } else {
      setFormData({
        name: "",
        webhook_url: "",
        secret_token: "",
        enabled: true,
        triggers: [defaultTrigger],
      });
    }
    setErrors({});
  }, [webhook, open]);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = "Name is required";
    }

    if (!formData.webhook_url.trim()) {
      newErrors.webhook_url = "URL is required";
    } else {
      try {
        new URL(formData.webhook_url);
      } catch {
        newErrors.webhook_url = "Invalid URL format";
      }
    }

    const validTriggers = formData.triggers.filter(
      (t) => t.trigger_type === "all" || t.trigger_value.trim()
    );
    if (validTriggers.length === 0) {
      newErrors.triggers = "At least one valid trigger is required";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      onSubmit(formData);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{webhook ? "Edit Webhook" : "Create Webhook"}</DialogTitle>
          <DialogDescription>
            {webhook ? "Update the webhook configuration" : "Configure a new webhook endpoint"}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="My Webhook"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            />
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="webhook_url">Webhook URL</Label>
            <Input
              id="webhook_url"
              type="url"
              placeholder="https://example.com/webhook"
              value={formData.webhook_url}
              onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
            />
            {errors.webhook_url && <p className="text-sm text-destructive">{errors.webhook_url}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="secret_token">Secret Token (optional)</Label>
            <Input
              id="secret_token"
              type="password"
              placeholder="Optional secret for verification"
              value={formData.secret_token}
              onChange={(e) => setFormData({ ...formData, secret_token: e.target.value })}
            />
          </div>

          <div className="flex items-center gap-2">
            <Switch
              id="enabled"
              checked={formData.enabled}
              onCheckedChange={(checked) => setFormData({ ...formData, enabled: checked })}
            />
            <Label htmlFor="enabled">Enabled</Label>
          </div>

          <TriggerBuilder
            triggers={formData.triggers}
            onChange={(triggers) => setFormData({ ...formData, triggers })}
          />
          {errors.triggers && <p className="text-sm text-destructive">{errors.triggers}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Saving..." : webhook ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

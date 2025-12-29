"use client";

import { WebhookTrigger } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Trash2, Plus } from "lucide-react";

interface TriggerBuilderProps {
  triggers: WebhookTrigger[];
  onChange: (triggers: WebhookTrigger[]) => void;
}

const triggerTypes = [
  { value: "all", label: "All Messages" },
  { value: "chat_jid", label: "Specific Chat" },
  { value: "sender", label: "Specific Sender" },
  { value: "keyword", label: "Keyword" },
  { value: "media_type", label: "Media Type" },
] as const;

const matchTypes = [
  { value: "exact", label: "Exact" },
  { value: "contains", label: "Contains" },
  { value: "regex", label: "Regex" },
] as const;

export function TriggerBuilder({ triggers, onChange }: TriggerBuilderProps) {
  const addTrigger = () => {
    onChange([
      ...triggers,
      { trigger_type: "all", trigger_value: "", match_type: "exact", enabled: true },
    ]);
  };

  const removeTrigger = (index: number) => {
    if (triggers.length > 1) {
      onChange(triggers.filter((_, i) => i !== index));
    }
  };

  const updateTrigger = (index: number, field: keyof WebhookTrigger, value: string | boolean) => {
    const updated = [...triggers];
    updated[index] = { ...updated[index], [field]: value };
    // Clear value when switching to "all"
    if (field === "trigger_type" && value === "all") {
      updated[index].trigger_value = "";
    }
    onChange(updated);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label>Triggers</Label>
        <Button type="button" variant="outline" size="sm" onClick={addTrigger}>
          <Plus className="h-3 w-3 mr-1" />
          Add Trigger
        </Button>
      </div>

      {triggers.map((trigger, index) => (
        <div key={index} className="flex gap-2 items-start p-3 border rounded-lg bg-muted/30">
          <div className="flex-1 grid grid-cols-3 gap-2">
            <Select
              value={trigger.trigger_type}
              onValueChange={(v) => updateTrigger(index, "trigger_type", v)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                {triggerTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    {type.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Input
              placeholder={trigger.trigger_type === "all" ? "No value needed" : "Value"}
              value={trigger.trigger_value}
              onChange={(e) => updateTrigger(index, "trigger_value", e.target.value)}
              disabled={trigger.trigger_type === "all"}
            />

            <Select
              value={trigger.match_type}
              onValueChange={(v) => updateTrigger(index, "match_type", v)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Match" />
              </SelectTrigger>
              <SelectContent>
                {matchTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    {type.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="shrink-0"
            onClick={() => removeTrigger(index)}
            disabled={triggers.length <= 1}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ))}
    </div>
  );
}

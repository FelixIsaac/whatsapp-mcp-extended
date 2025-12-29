"use client";

import { useEffect } from "react";
import { WebhookList } from "@/components/webhooks";
import { useSettings } from "@/lib/store";

export default function WebhooksPage() {
  const { darkMode } = useSettings();

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  }, [darkMode]);

  return (
    <div className="p-8">
      <div className="max-w-6xl mx-auto">
        <WebhookList />
      </div>
    </div>
  );
}

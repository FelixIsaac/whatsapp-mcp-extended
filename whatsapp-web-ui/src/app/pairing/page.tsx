"use client";

import { useState, useEffect } from "react";
import { PhoneInput, CodeDisplay, Dashboard, SettingsDialog } from "@/components/pairing";
import { usePairing, useSettings } from "@/lib/store";
import { WhatsAppAPI } from "@/lib/api";

export default function PairingPage() {
  const { step, setStep, setJid } = usePairing();
  const { apiKey, darkMode } = useSettings();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  }, [darkMode]);

  useEffect(() => {
    const checkExistingConnection = async () => {
      try {
        const api = new WhatsAppAPI(apiKey);
        const status = await api.getConnectionStatus();
        if (status.success && status.linked && status.jid) {
          setJid(status.jid);
          setStep("dashboard");
        }
      } catch (error) {
        console.log("No existing connection");
      } finally {
        setInitialized(true);
      }
    };

    checkExistingConnection();
  }, [apiKey, setJid, setStep]);

  if (!initialized) {
    return (
      <div className="flex items-center justify-center min-h-[80vh]">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    );
  }

  return (
    <div className="p-8">
      <div className="max-w-lg mx-auto">
        <h1 className="text-2xl font-bold mb-6 text-center">Device Pairing</h1>
        
        {step === "phone" && <PhoneInput />}
        {step === "code" && <CodeDisplay />}
        {step === "dashboard" && <Dashboard onOpenSettings={() => setSettingsOpen(true)} />}

        <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      </div>
    </div>
  );
}

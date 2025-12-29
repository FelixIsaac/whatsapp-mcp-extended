"use client";

import { useState, useEffect } from "react";
import { PhoneInput, CodeDisplay, Dashboard, SettingsDialog } from "@/components/pairing";
import { usePairing, useSettings } from "@/lib/store";
import { WhatsAppAPI } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Settings } from "lucide-react";

export default function Home() {
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
      <main className="min-h-screen bg-gradient-to-br from-purple-600 via-purple-500 to-indigo-600 flex items-center justify-center p-4">
        <div className="text-white text-lg">Loading...</div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-purple-600 via-purple-500 to-indigo-600 flex flex-col items-center justify-center p-4">
      <div className="absolute top-4 right-4">
        <Button
          variant="ghost"
          size="icon"
          className="text-white/80 hover:text-white hover:bg-white/10"
          onClick={() => setSettingsOpen(true)}
        >
          <Settings className="h-5 w-5" />
        </Button>
      </div>

      {step === "phone" && <PhoneInput />}
      {step === "code" && <CodeDisplay />}
      {step === "dashboard" && <Dashboard onOpenSettings={() => setSettingsOpen(true)} />}

      <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />

      <footer className="absolute bottom-4 text-center text-white/60 text-sm">
        WhatsApp MCP Extended - Phone Pairing Interface
      </footer>
    </main>
  );
}

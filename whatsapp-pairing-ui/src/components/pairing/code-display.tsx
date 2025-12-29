"use client";

import { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Clock, Loader2, CheckCircle, AlertCircle, Smartphone, Copy } from "lucide-react";
import { WhatsAppAPI, getErrorMessage } from "@/lib/api";
import { useSettings, usePairing } from "@/lib/store";
import { toast } from "sonner";

type PairingStatus = "waiting" | "success" | "error" | "expired";

interface StatusInfo {
  icon: typeof Loader2;
  text: string;
  color: string;
  animate: boolean;
}

const statusConfigs: Record<PairingStatus, StatusInfo> = {
  waiting: { icon: Loader2, text: "Waiting for phone...", color: "text-muted-foreground", animate: true },
  success: { icon: CheckCircle, text: "Pairing successful!", color: "text-green-500", animate: false },
  error: { icon: AlertCircle, text: "Pairing failed", color: "text-destructive", animate: false },
  expired: { icon: AlertCircle, text: "Code expired", color: "text-destructive", animate: false },
};

export function CodeDisplay() {
  const { apiKey } = useSettings();
  const { pairingCode, expiresIn, setStep, setJid } = usePairing();
  const [countdown, setCountdown] = useState(expiresIn);
  const [status, setStatus] = useState<PairingStatus>("waiting");

  const copyCode = () => {
    navigator.clipboard.writeText(pairingCode);
    toast.success("Code copied to clipboard");
  };

  const checkStatus = useCallback(async () => {
    try {
      const api = new WhatsAppAPI(apiKey);
      const result = await api.getPairingStatus();

      if (result.complete) {
        setStatus("success");
        const connStatus = await api.getConnectionStatus();
        if (connStatus.jid) {
          setJid(connStatus.jid);
        }
        toast.success("Pairing successful!", {
          description: "Your device is now linked",
        });
        setTimeout(() => setStep("dashboard"), 1500);
      } else if (result.error) {
        setStatus("error");
        toast.error("Pairing failed", { description: result.error });
      }
    } catch (error) {
      const msg = getErrorMessage(error);
      console.error("Polling error:", msg);
    }
  }, [apiKey, setJid, setStep]);

  useEffect(() => {
    if (status !== "waiting") return;

    const countdownInterval = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          setStatus("expired");
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    const pollInterval = setInterval(checkStatus, 2000);

    return () => {
      clearInterval(countdownInterval);
      clearInterval(pollInterval);
    };
  }, [status, checkStatus]);

  const currentStatus = statusConfigs[status];
  const StatusIcon = currentStatus.icon;

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader className="text-center">
        <CardTitle>Enter This Code</CardTitle>
        <CardDescription>On your phones WhatsApp app</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="relative">
          <div
            className="text-4xl font-mono font-bold text-center py-8 px-4 rounded-lg bg-gradient-to-br from-purple-500 to-purple-700 text-white tracking-widest cursor-pointer hover:opacity-90 transition-opacity"
            onClick={copyCode}
            title="Click to copy"
          >
            {pairingCode}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="absolute top-2 right-2 text-white/70 hover:text-white hover:bg-white/20"
            onClick={copyCode}
          >
            <Copy className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex items-center justify-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <span className={countdown <= 30 ? "text-destructive font-medium" : "text-muted-foreground"}>
            {countdown > 0 ? countdown + "s remaining" : "Expired"}
          </span>
        </div>

        <div className={"flex items-center justify-center gap-2 " + currentStatus.color}>
          <StatusIcon className={"h-5 w-5 " + (currentStatus.animate ? "animate-spin" : "")} />
          <span>{currentStatus.text}</span>
        </div>

        <div className="border rounded-lg p-4">
          <div className="flex items-center gap-2 mb-4">
            <Smartphone className="h-5 w-5 text-green-500" />
            <span className="font-medium">On Your Phone:</span>
          </div>

          <Tabs defaultValue="android" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="android">Android</TabsTrigger>
              <TabsTrigger value="ios">iOS</TabsTrigger>
            </TabsList>
            <TabsContent value="android" className="mt-4">
              <ol className="list-decimal list-inside space-y-2 text-sm text-muted-foreground">
                <li>Open <strong className="text-foreground">WhatsApp</strong></li>
                <li>Tap <strong className="text-foreground">Settings</strong> (or menu)</li>
                <li>Tap <strong className="text-foreground">Linked devices</strong></li>
                <li>Tap <strong className="text-foreground">Link a device</strong></li>
                <li>Tap <strong className="text-foreground">Link with phone number</strong></li>
                <li>Enter phone number, tap <strong className="text-foreground">Next</strong></li>
                <li><Badge variant="secondary" className="font-mono">{pairingCode}</Badge> Enter code</li>
              </ol>
            </TabsContent>
            <TabsContent value="ios" className="mt-4">
              <ol className="list-decimal list-inside space-y-2 text-sm text-muted-foreground">
                <li>Open <strong className="text-foreground">WhatsApp</strong></li>
                <li>Tap <strong className="text-foreground">Settings</strong></li>
                <li>Tap <strong className="text-foreground">Linked devices</strong></li>
                <li>Tap <strong className="text-foreground">Link a device</strong></li>
                <li>Tap <strong className="text-foreground">Link with phone number</strong></li>
                <li>Enter phone number, tap <strong className="text-foreground">Next</strong></li>
                <li><Badge variant="secondary" className="font-mono">{pairingCode}</Badge> Enter code</li>
              </ol>
            </TabsContent>
          </Tabs>
        </div>

        {status === "expired" && (
          <Button variant="outline" className="w-full" onClick={() => setStep("phone")}>
            Generate New Code
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

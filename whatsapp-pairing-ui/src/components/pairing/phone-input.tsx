"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Loader2, Key } from "lucide-react";
import { WhatsAppAPI, getErrorMessage } from "@/lib/api";
import { useSettings, usePairing } from "@/lib/store";
import { toast } from "sonner";

export function PhoneInput() {
  const [phone, setPhone] = useState("");
  const [loading, setLoading] = useState(false);
  const { apiKey } = useSettings();
  const { setStep, setPhoneNumber, setPairingCode } = usePairing();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const cleanedPhone = phone.replace(/[^0-9]/g, "");
    if (!cleanedPhone || cleanedPhone.length < 10) {
      toast.error("Invalid phone number", {
        description: "Enter country code + number (e.g., 6596149012)",
      });
      return;
    }

    setLoading(true);
    try {
      const api = new WhatsAppAPI(apiKey);
      const result = await api.pair(cleanedPhone);

      if (result.success && result.code) {
        setPhoneNumber(cleanedPhone);
        setPairingCode(result.code, result.expires_in || 160);
        setStep("code");
        toast.success("Pairing code generated!", {
          description: "Enter the code on your phone",
        });
      } else {
        throw new Error(result.error || "Failed to generate code");
      }
    } catch (error) {
      const msg = getErrorMessage(error);
      toast.error(msg.title, {
        description: msg.description,
        action: msg.action ? { label: "Settings", onClick: () => {} } : undefined,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader className="text-center">
        <CardTitle className="flex items-center justify-center gap-2 text-2xl">
          <svg viewBox="0 0 24 24" className="w-8 h-8 text-[#25D366]" fill="currentColor">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z"/>
          </svg>
          WhatsApp Pairing
        </CardTitle>
        <CardDescription>
          Enter your phone number to link this device
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="phone">Phone Number</Label>
            <Input
              id="phone"
              type="tel"
              placeholder="6596149012"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              disabled={loading}
              aria-describedby="phone-help"
            />
            <p id="phone-help" className="text-sm text-muted-foreground">
              Format: country code + number, no leading zeros
            </p>
          </div>
          <Button type="submit" className="w-full bg-[#25D366] hover:bg-[#128C7E]" disabled={loading}>
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Key className="mr-2 h-4 w-4" />
                Generate Code
              </>
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

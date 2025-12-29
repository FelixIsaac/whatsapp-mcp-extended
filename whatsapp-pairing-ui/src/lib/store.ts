"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SettingsState {
  apiKey: string;
  darkMode: boolean;
  setApiKey: (key: string) => void;
  setDarkMode: (dark: boolean) => void;
}

export const useSettings = create<SettingsState>()(
  persist(
    (set) => ({
      apiKey: "test_key_for_build_verification_only",
      darkMode: false,
      setApiKey: (apiKey) => set({ apiKey }),
      setDarkMode: (darkMode) => set({ darkMode }),
    }),
    {
      name: "whatsapp-pairing-settings",
    }
  )
);

type PairingStep = "phone" | "code" | "dashboard";

interface PairingState {
  step: PairingStep;
  phoneNumber: string;
  pairingCode: string;
  expiresIn: number;
  jid: string;
  setStep: (step: PairingStep) => void;
  setPhoneNumber: (phone: string) => void;
  setPairingCode: (code: string, expiresIn: number) => void;
  setJid: (jid: string) => void;
  reset: () => void;
}

export const usePairing = create<PairingState>((set) => ({
  step: "phone",
  phoneNumber: "",
  pairingCode: "",
  expiresIn: 0,
  jid: "",
  setStep: (step) => set({ step }),
  setPhoneNumber: (phoneNumber) => set({ phoneNumber }),
  setPairingCode: (pairingCode, expiresIn) => set({ pairingCode, expiresIn }),
  setJid: (jid) => set({ jid }),
  reset: () =>
    set({
      step: "phone",
      phoneNumber: "",
      pairingCode: "",
      expiresIn: 0,
      jid: "",
    }),
}));

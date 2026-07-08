"use client";

import { Suspense } from "react";
import ChatView from "./ChatView";

export default function ChatsPage() {
  return (
    <Suspense fallback={<div className="p-8 text-muted-foreground">Loading…</div>}>
      <ChatView />
    </Suspense>
  );
}

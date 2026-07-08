"use client";

import { useEffect, useRef, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { WhatsAppAPI, Chat, Message } from "@/lib/api";
import { useSettings } from "@/lib/store";
import { Users, User, RefreshCw, ArrowLeft, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

function timeAgo(iso: string): string {
  const d = new Date(iso);
  const diff = Date.now() - d.getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return d.toLocaleDateString([], { month: "short", day: "numeric" });
}

function timeHM(iso: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function ChatList({ onSelect }: { onSelect: (jid: string) => void }) {
  const { apiKey } = useSettings();
  const [chats, setChats] = useState<Chat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const api = new WhatsAppAPI(apiKey);
      const data = await api.getChats();
      data.sort((a, b) => {
        const ta = a.last_message_time ? new Date(a.last_message_time).getTime() : 0;
        const tb = b.last_message_time ? new Date(b.last_message_time).getTime() : 0;
        return tb - ta;
      });
      setChats(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load chats");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [apiKey]);

  const filtered = chats.filter(c =>
    (c.name || c.jid).toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="flex flex-col h-screen">
      <div className="p-4 border-b bg-card flex items-center gap-3">
        <h1 className="text-xl font-bold flex-1">Chats</h1>
        <Button variant="ghost" size="icon" onClick={load} disabled={loading}>
          <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
        </Button>
      </div>
      <div className="p-3 border-b">
        <Input
          placeholder="Search chats…"
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="h-8"
        />
      </div>
      <div className="flex-1 overflow-y-auto">
        {error && <div className="p-4 text-sm text-destructive">{error}</div>}
        {!error && !loading && filtered.length === 0 && (
          <div className="p-8 text-center text-muted-foreground text-sm">
            {search ? "No chats match." : "No chats found. Make sure WhatsApp is connected."}
          </div>
        )}
        {filtered.map(chat => {
          const name = chat.name || chat.jid;
          const preview = chat.last_message
            ? (chat.last_is_from_me ? "You: " : "") + chat.last_message
            : "";
          return (
            <button
              key={chat.jid}
              onClick={() => onSelect(chat.jid)}
              className="w-full flex items-center gap-3 px-4 py-3 border-b hover:bg-muted/50 transition-colors text-left"
            >
              <div className="flex-shrink-0 w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                {chat.is_group
                  ? <Users className="h-5 w-5 text-muted-foreground" />
                  : <User className="h-5 w-5 text-muted-foreground" />
                }
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium text-sm truncate">{name}</span>
                  {chat.last_message_time && (
                    <span className="text-xs text-muted-foreground whitespace-nowrap">{timeAgo(chat.last_message_time)}</span>
                  )}
                </div>
                {preview && (
                  <p className="text-xs text-muted-foreground truncate mt-0.5">{preview}</p>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ChatDetail({ jid, onBack }: { jid: string; onBack: () => void }) {
  const { apiKey } = useSettings();
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [text, setText] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  const load = async () => {
    try {
      const api = new WhatsAppAPI(apiKey);
      const data = await api.getMessages(jid, 100);
      setMessages(data);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load messages");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [jid, apiKey]);
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const send = async () => {
    if (!text.trim() || sending) return;
    setSending(true);
    try {
      const api = new WhatsAppAPI(apiKey);
      await api.sendMessage(jid, text.trim());
      setText("");
      await load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to send");
    } finally {
      setSending(false);
    }
  };

  const displayName = jid.endsWith("@g.us")
    ? `Group ${jid.split("@")[0]}`
    : jid.split("@")[0];

  return (
    <div className="flex flex-col h-screen">
      <div className="p-4 border-b bg-card flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1 min-w-0">
          <h1 className="font-bold truncate">{displayName}</h1>
          <p className="text-xs text-muted-foreground truncate">{jid}</p>
        </div>
        <Button variant="ghost" size="sm" onClick={load}>Refresh</Button>
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-2">
        {loading && <p className="text-center text-sm text-muted-foreground">Loading…</p>}
        {messages.map(msg => {
          const senderLabel = !msg.is_from_me && msg.sender_name ? msg.sender_name : null;
          return (
            <div key={msg.id} className={cn("flex", msg.is_from_me ? "justify-end" : "justify-start")}>
              <div className={cn(
                "max-w-[70%] rounded-2xl px-3 py-2 text-sm",
                msg.is_from_me
                  ? "bg-green-600 text-white rounded-br-sm"
                  : "bg-card border rounded-bl-sm"
              )}>
                {senderLabel && (
                  <p className="text-xs font-semibold text-green-400 mb-0.5">{senderLabel}</p>
                )}
                {msg.media_type && msg.media_type !== "text" && (
                  <p className="text-xs opacity-70 italic mb-0.5">[{msg.media_type}]</p>
                )}
                <p className="whitespace-pre-wrap break-words">{msg.content}</p>
                {msg.timestamp && (
                  <p className={cn("text-xs mt-1 text-right", msg.is_from_me ? "text-green-200" : "text-muted-foreground")}>
                    {timeHM(msg.timestamp)}
                  </p>
                )}
              </div>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>
      <div className="p-3 border-t bg-card flex gap-2">
        <Input
          value={text}
          onChange={e => setText(e.target.value)}
          onKeyDown={e => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); } }}
          placeholder="Type a message…"
          disabled={sending}
          className="flex-1"
        />
        <Button
          onClick={send}
          disabled={sending || !text.trim()}
          size="icon"
          className="bg-green-600 hover:bg-green-700"
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

export default function ChatView() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const jid = searchParams.get("jid");

  const selectChat = (selectedJid: string) => {
    router.push(`/chats?jid=${encodeURIComponent(selectedJid)}`);
  };

  const goBack = () => {
    router.push("/chats");
  };

  if (jid) {
    return <ChatDetail jid={jid} onBack={goBack} />;
  }

  return <ChatList onSelect={selectChat} />;
}

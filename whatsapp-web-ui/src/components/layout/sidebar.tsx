"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Link2, MessageSquare, Settings, Webhook } from "lucide-react";

const navItems = [
  {
    title: "Pairing",
    href: "/pairing",
    icon: Link2,
    description: "Link WhatsApp device",
  },
  {
    title: "Webhooks",
    href: "/webhooks",
    icon: Webhook,
    description: "Manage webhook endpoints",
  },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-card border-r min-h-screen p-4 flex flex-col">
      <div className="flex items-center gap-2 px-2 mb-8">
        <MessageSquare className="h-8 w-8 text-green-500" />
        <div>
          <h1 className="font-bold text-lg">WhatsApp MCP</h1>
          <p className="text-xs text-muted-foreground">Extended</p>
        </div>
      </div>

      <nav className="flex-1 space-y-2">
        {navItems.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
                isActive
                  ? "bg-primary text-primary-foreground"
                  : "hover:bg-muted text-muted-foreground hover:text-foreground"
              )}
            >
              <item.icon className="h-5 w-5" />
              <div>
                <div className="font-medium text-sm">{item.title}</div>
                <div className={cn("text-xs", isActive ? "text-primary-foreground/80" : "text-muted-foreground")}>
                  {item.description}
                </div>
              </div>
            </Link>
          );
        })}
      </nav>

      <div className="border-t pt-4 mt-4">
        <Link
          href="/settings"
          className={cn(
            "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
            pathname === "/settings"
              ? "bg-primary text-primary-foreground"
              : "hover:bg-muted text-muted-foreground hover:text-foreground"
          )}
        >
          <Settings className="h-5 w-5" />
          <span className="font-medium text-sm">Settings</span>
        </Link>
      </div>
    </aside>
  );
}

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Link2, Webhook, ArrowRight } from "lucide-react";
import Link from "next/link";

export default function Home() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl font-bold mb-2">WhatsApp MCP Extended</h1>
        <p className="text-muted-foreground mb-8">
          Manage your WhatsApp bridge, device pairing, and webhook configurations
        </p>

        <div className="grid md:grid-cols-2 gap-6">
          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-500/10 rounded-lg">
                  <Link2 className="h-6 w-6 text-green-500" />
                </div>
                <div>
                  <CardTitle>Device Pairing</CardTitle>
                  <CardDescription>Link your WhatsApp device</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                Connect a new WhatsApp device using phone number pairing. 
                Get an 8-digit code to enter on your phone.
              </p>
              <Link href="/pairing">
                <Button className="w-full bg-green-600 hover:bg-green-700">
                  Go to Pairing
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </Link>
            </CardContent>
          </Card>

          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-purple-500/10 rounded-lg">
                  <Webhook className="h-6 w-6 text-purple-500" />
                </div>
                <div>
                  <CardTitle>Webhook Manager</CardTitle>
                  <CardDescription>Configure webhook endpoints</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                Create and manage webhooks to receive WhatsApp messages. 
                Set up triggers, view logs, and test your endpoints.
              </p>
              <Link href="/webhooks">
                <Button variant="outline" className="w-full">
                  Manage Webhooks
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

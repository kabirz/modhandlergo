import React, { useState, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useWailsEvent } from "@/hooks/useEvents";
import { Radio } from "lucide-react";
import { LoRaDataService, LoRaConfigService } from "../../bindings/github.com/kabirz/modhandlergo/service";

export function LoRaDataPage() {
  const [ip, setIp] = useState("192.168.1.100");
  const [port, setPort] = useState("8080");
  const [connected, setConnected] = useState(false);
  const [scanner, setScanner] = useState<any>(null);
  const [logs, setLogs] = useState<string[]>([]);

  const addLog = useCallback((msg: string) => {
    setLogs((prev) => [...prev.slice(-200), msg]);
  }, []);

  useWailsEvent<number>("lora:connstate", (state) => {
    setConnected(state === 2);
    const labels = ["已断开", "连接中...", "已连接"];
    addLog(`连接状态: ${labels[state] || "未知"}`);
  });

  useWailsEvent<any>("lora:frame", (data) => {
    if (data?.payload?.[0] === 0x01) {
      setScanner(data);
    }
  });

  useWailsEvent<string>("lora:log", (msg) => {
    addLog(msg);
  });

  const handleConnect = async () => {
    try {
      if (connected) {
        await LoRaDataService.Disconnect();
      } else {
        await LoRaDataService.Connect(ip, parseInt(port));
      }
    } catch (err: any) {
      addLog(`错误: ${err.message || err}`);
    }
  };

  return (
    <div className="space-y-4">
      {/* Connection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Radio className="h-4 w-4" /> TCP 连接
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Input value={ip} onChange={(e) => setIp(e.target.value)} placeholder="网关 IP" className="w-48" />
            <Input value={port} onChange={(e) => setPort(e.target.value)} placeholder="端口" className="w-24" />
            <Button onClick={handleConnect} variant={connected ? "destructive" : "default"}>
              {connected ? "断开" : "连接"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Telemetry */}
      <Card>
        <CardHeader>
          <CardTitle>遥测数据</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-4">
            <DataField label="X 坐标" value={scanner ? "—" : "—"} />
            <DataField label="Y 坐标" value={scanner ? "—" : "—"} />
            <DataField label="激光测距" value={scanner ? "—" : "—"} />
            <DataField label="超欠挖" value={scanner ? "—" : "—"} />
          </div>
        </CardContent>
      </Card>

      {/* Log */}
      <Card className="flex-1">
        <CardHeader>
          <CardTitle>日志</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-48 overflow-y-auto bg-terminal-bg rounded-md p-3 font-mono text-xs text-terminal-fg">
            {logs.length === 0 ? (
              <p className="text-muted-foreground">等待日志...</p>
            ) : (
              logs.map((log, i) => <div key={i}>{log}</div>)
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function DataField({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center p-3 rounded-md bg-muted/50">
      <p className="text-xs text-muted-foreground mb-1">{label}</p>
      <p className="text-lg font-mono font-semibold">{value}</p>
    </div>
  );
}

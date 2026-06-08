import { useState, useEffect, useRef, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { useI18n } from "@/lib/i18n";
import { Play, Square, Trash2, Send, Radio, Wifi } from "lucide-react";
import { GatewaySimService } from "../../bindings/github.com/kabirz/modhandlergo/service";
import type { GatewaySimConfig } from "../../bindings/github.com/kabirz/modhandlergo/service/models";

export function GatewaySimPage() {
  const { t } = useI18n();
  const [running, setRunning] = useState(false);
  const [tcpPort, setTcpPort] = useState("1234");
  const [udpPort, setUdpPort] = useState("1566");
  const [nid, setNid] = useState("00000001");
  const [gwid, setGwid] = useState("00000005");
  const [autoTelemetry, setAutoTelemetry] = useState(false);
  const [clientConnected, setClientConnected] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);
  const logRef = useRef<HTMLDivElement>(null);

  const addLog = useCallback((line: string) => {
    setLogs((prev) => [...prev.slice(-500), line]);
  }, []);

  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logs]);

  useEffect(() => {
    GatewaySimService.IsRunning().then(setRunning).catch(() => {});
  }, []);

  useWailsEvent<boolean>("gateway:sim:status", (isRunning) => {
    setRunning(isRunning);
    if (!isRunning) {
      addLog("── Gateway simulator stopped ──");
      setAutoTelemetry(false);
      setClientConnected(false);
    }
  });

  useWailsEvent<boolean>("gateway:sim:client", (connected) => {
    setClientConnected(connected);
    if (!connected) setAutoTelemetry(false);
  });

  useWailsEvent<string>("gateway:sim:log", (line) => {
    addLog(line);
  });

  const handleStart = async () => {
    const config: GatewaySimConfig = {
      tcpPort: parseInt(tcpPort) || 1234,
      udpPort: parseInt(udpPort) || 1566,
      nid,
      gwid,
    };
    try {
      await GatewaySimService.Start(config);
      addLog("── Gateway simulator started ──");
    } catch (err: any) {
      addLog(`Error: ${err}`);
    }
  };

  const handleStop = async () => {
    try {
      await GatewaySimService.Stop();
    } catch (err: any) {
      addLog(`Error: ${err}`);
    }
  };

  const sendCmd = async (cmd: string) => {
    try {
      await GatewaySimService.SendCommand(cmd);
    } catch (err: any) {
      addLog(`Error: ${err}`);
    }
  };

  const toggleAutoTelemetry = async () => {
    const newState = !autoTelemetry;
    await sendCmd(`auto ${newState ? "on" : "off"}`);
    setAutoTelemetry(newState);
  };

  const disabled = !running;
  const cmdDisabled = !running || !clientConnected;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-[400px_1fr] gap-3">
        {/* Config */}
        <Card>
          <CardHeader className="pb-1 pt-2">
            <CardTitle className="text-sm">{t("gw.config")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1.5">
            <div className="flex items-center gap-2">
              <label className="text-[10px] text-muted-foreground w-14 shrink-0">{t("gw.tcpPort")}:</label>
              <Input value={tcpPort} onChange={(e) => setTcpPort(e.target.value)} className="h-6 text-xs w-20 font-mono" disabled={running} />
              <label className="text-[10px] text-muted-foreground w-14 shrink-0 ml-2">{t("gw.udpPort")}:</label>
              <Input value={udpPort} onChange={(e) => setUdpPort(e.target.value)} className="h-6 text-xs w-20 font-mono" disabled={running} />
            </div>
            <div className="flex items-center gap-2">
              <label className="text-[10px] text-muted-foreground w-14 shrink-0">NID:</label>
              <Input value={nid} onChange={(e) => setNid(e.target.value.replace(/[^0-9a-fA-F]/g, "").slice(0, 8))} className="h-6 text-xs w-24 font-mono" disabled={running} maxLength={8} />
              <label className="text-[10px] text-muted-foreground w-14 shrink-0 ml-2">GWID:</label>
              <Input value={gwid} onChange={(e) => setGwid(e.target.value.replace(/[^0-9a-fA-F]/g, "").slice(0, 8))} className="h-6 text-xs w-24 font-mono" disabled={running} maxLength={8} />
            </div>
            <div className="flex items-center justify-center pt-2">
              {running ? (
                <Button onClick={handleStop} variant="destructive" size="sm" className="h-7 text-xs px-6">
                  <Square className="h-3.5 w-3.5 mr-1" /> {t("gw.stop")}
                </Button>
              ) : (
                <Button onClick={handleStart} size="sm" className="h-7 text-xs px-6">
                  <Play className="h-3.5 w-3.5 mr-1" /> {t("gw.start")}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Quick Commands */}
        <Card>
          <CardHeader className="pb-1 pt-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm">{t("gw.commands")}</CardTitle>
              <div className="flex items-center gap-2">
                <div className="flex items-center gap-1">
                  <span className={`h-2 w-2 rounded-full ${running ? "bg-green-500" : "bg-muted-foreground"}`} />
                  <span className="text-[10px] text-muted-foreground">{running ? t("gw.running") : t("gw.stopped")}</span>
                </div>
                {running && (
                  <div className="flex items-center gap-1">
                    <span className={`h-2 w-2 rounded-full ${clientConnected ? "bg-green-500" : "bg-muted-foreground"}`} />
                    <span className="text-[10px] text-muted-foreground">{clientConnected ? t("gw.connected") : t("gw.waiting")}</span>
                  </div>
                )}
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex items-center gap-2">
              <Button onClick={() => sendCmd("telemetry")} size="sm" variant="outline" className="h-7 text-[10px] px-2" disabled={cmdDisabled}>
                <Radio className="h-3 w-3 mr-1" /> {t("gw.telemetry")}
              </Button>
              <Button onClick={() => sendCmd("rssi")} size="sm" variant="outline" className="h-7 text-[10px] px-2" disabled={cmdDisabled}>
                <Wifi className="h-3 w-3 mr-1" /> RSSI
              </Button>
              <Button onClick={toggleAutoTelemetry} size="sm" variant={autoTelemetry ? "destructive" : "outline"} className="h-7 text-[10px] px-2" disabled={cmdDisabled}>
                <Send className="h-3 w-3 mr-1" /> {autoTelemetry ? t("gw.stopAuto") : t("gw.autoTelemetry")}
              </Button>
              <Button onClick={() => sendCmd("stats")} size="sm" variant="outline" className="h-7 text-[10px] px-2" disabled={cmdDisabled}>
                {t("gw.stats")}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Log Output */}
      <Card>
        <CardHeader className="pb-1 pt-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm">{t("gw.log")}</CardTitle>
            <Button variant="ghost" size="icon" onClick={() => setLogs([])} title="Clear">
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div ref={logRef} className="overflow-y-auto bg-terminal-bg rounded-md font-mono text-[11px] h-[350px] p-2 whitespace-pre-wrap terminal-selectable">
            {logs.length === 0 ? (
              <span className="text-muted-foreground">{t("gw.noLog")}</span>
            ) : (
              logs.join("\n")
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

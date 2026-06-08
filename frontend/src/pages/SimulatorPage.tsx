import { useState, useEffect, useRef, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { useI18n } from "@/lib/i18n";
import { Play, Square, Trash2 } from "lucide-react";
import { SimulatorService } from "../../bindings/github.com/kabirz/modhandlergo/service";
import type { SimulatorConfig } from "../../bindings/github.com/kabirz/modhandlergo/service/models";

interface SimulatorPageProps {
  canConnected: boolean;
}

export function SimulatorPage({ canConnected }: SimulatorPageProps) {
  const { t } = useI18n();
  const [running, setRunning] = useState(false);
  const [version, setVersion] = useState("0x00010203");
  const [noHeartbeat, setNoHeartbeat] = useState(false);
  const [noHandler, setNoHandler] = useState(false);
  const [handlerInterval, setHandlerInterval] = useState("0.10");
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
    SimulatorService.IsRunning().then(setRunning).catch(() => {});
  }, []);

  useEffect(() => {
    if (!canConnected && running) {
      SimulatorService.Stop().catch(() => {});
    }
  }, [canConnected, running]);

  useWailsEvent<boolean>("simulator:status", (isRunning) => {
    setRunning(isRunning);
    if (!isRunning) addLog("── Simulator stopped ──");
  });

  useWailsEvent<string>("simulator:log", (line) => {
    addLog(line);
  });

  const handleStart = async () => {
    const config: SimulatorConfig = {
      channel: "",
      version,
      noHeartbeat,
      noHandler,
      handlerInterval: parseFloat(handlerInterval) || 0.1,
    };
    try {
      await SimulatorService.Start(config);
      addLog("── Simulator started ──");
    } catch (err: any) {
      addLog(`Error: ${err}`);
    }
  };

  const handleStop = async () => {
    try {
      await SimulatorService.Stop();
    } catch (err: any) {
      addLog(`Error: ${err}`);
    }
  };

  const disabled = running || !canConnected;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-[420px_1fr] gap-3">
        <Card>
          <CardHeader className="pb-1 pt-2">
            <CardTitle className="text-sm">{t("sim.config")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1.5">
            <div className="flex items-center gap-2">
              <label className="text-[10px] text-muted-foreground w-20 shrink-0">{t("sim.version")}:</label>
              <Input value={version} onChange={(e) => setVersion(e.target.value)} className="h-6 text-xs flex-1 font-mono" disabled={disabled} />
            </div>
            <div className="flex items-center gap-2">
              <label className="text-[10px] text-muted-foreground w-20 shrink-0">{t("sim.handlerInterval")}:</label>
              <Input value={handlerInterval} onChange={(e) => setHandlerInterval(e.target.value)} className="h-6 text-xs w-20 font-mono" disabled={disabled || noHandler} />
            </div>
            <div className="flex items-center gap-4 pt-1">
              <label className={`flex items-center gap-1 text-[10px] ${disabled ? "text-muted-foreground" : "cursor-pointer"}`}>
                <input type="checkbox" checked={noHeartbeat} onChange={(e) => setNoHeartbeat(e.target.checked)} disabled={disabled} />
                {t("sim.disableHeartbeat")}
              </label>
              <label className={`flex items-center gap-1 text-[10px] ${disabled ? "text-muted-foreground" : "cursor-pointer"}`}>
                <input type="checkbox" checked={noHandler} onChange={(e) => setNoHandler(e.target.checked)} disabled={disabled} />
                {t("sim.disableHandler")}
              </label>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-1 pt-2">
            <CardTitle className="text-sm">{t("sim.control")}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col items-center justify-center gap-3 h-full min-h-[120px]">
            {!canConnected && (
              <span className="text-xs text-muted-foreground">{t("sim.connectFirst")}</span>
            )}
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${running ? "bg-green-500" : "bg-muted-foreground"}`} />
              <span className="text-xs">{running ? t("sim.running") : t("sim.stopped")}</span>
            </div>
            {running ? (
              <Button onClick={handleStop} variant="destructive" size="sm" className="h-7 text-xs px-4">
                <Square className="h-3.5 w-3.5 mr-1" /> {t("sim.stop")}
              </Button>
            ) : (
              <Button onClick={handleStart} size="sm" className="h-7 text-xs px-4" disabled={!canConnected}>
                <Play className="h-3.5 w-3.5 mr-1" /> {t("sim.start")}
              </Button>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-1 pt-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm">{t("sim.log")}</CardTitle>
            <Button variant="ghost" size="icon" onClick={() => setLogs([])} title="Clear">
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div ref={logRef} className="overflow-y-auto bg-terminal-bg rounded-md font-mono text-[11px] h-[300px] p-2 whitespace-pre-wrap terminal-selectable">
            {logs.length === 0 ? (
              <span className="text-muted-foreground">{t("sim.noLog")}</span>
            ) : (
              logs.join("\n")
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

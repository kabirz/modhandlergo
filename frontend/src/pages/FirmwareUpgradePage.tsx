import React, { useState, useCallback, useRef, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Progress } from "@/components/ui/progress";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { useWailsEvent } from "@/hooks/useEvents";
import { Upload, Trash2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { msTimestamp } from "@/lib/utils";
import { CANUpgradeService } from "../../bindings/github.com/kabirz/modhandlergo/service";

const baudRates = ["10K", "20K", "50K", "100K", "125K", "250K", "500K", "1M"];
const serialBaudRates = ["9600", "19200", "38400", "57600", "115200", "230400", "460800", "921600"];

interface SerialPortInfo {
  portName: string;
  friendlyName: string;
}

let logIdCounter = 0;

const LogLine = React.memo(function LogLine({ text }: { text: string }) {
  return <div>{text}</div>;
});

export function FirmwareUpgradePage() {
  const { t } = useI18n();
  const [channel, setChannel] = useState<"can" | "uart">("can");
  const [baudIndex, setBaudIndex] = useState(5);
  const [serialBaud, setSerialBaud] = useState("115200");
  const [canDevices, setCanDevices] = useState<number[]>([]);
  const [selectedDevice, setSelectedDevice] = useState("");
  const [serialPorts, setSerialPorts] = useState<SerialPortInfo[]>([]);
  const [selectedPort, setSelectedPort] = useState("");
  const [firmwarePath, setFirmwarePath] = useState("");
  const [connected, setConnected] = useState(false);
  const [progress, setProgress] = useState(0);
  const [version, setVersion] = useState("");
  const [logs, setLogs] = useState<{ id: number; text: string }[]>([]);
  const logRef = useRef<HTMLDivElement>(null);
  const channelRef = useRef(channel);
  channelRef.current = channel;

  // Dialog state
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogConfig, setDialogConfig] = useState<{ type: "alert" | "confirm"; title: string; message: string; onConfirm?: () => void }>({ type: "alert", title: "", message: "" });

  const addLog = useCallback((msg: string) => {
    setLogs((prev) => [...prev.slice(-500), { id: logIdCounter++, text: msg }]);
  }, []);

  // Auto-scroll log
  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [logs]);

  useWailsEvent<string>("can:log", (msg) => { if (channelRef.current === "can") addLog(msg); });
  useWailsEvent<number>("can:progress", (pct) => setProgress(pct));
  useWailsEvent<string>("uart:log", (msg) => { if (channelRef.current === "uart") addLog(msg); });
  useWailsEvent<number>("uart:progress", (pct) => setProgress(pct));

  const detectCanDevices = () => {
    CANUpgradeService.DetectCANDevices().then((devices) => {
      setCanDevices(devices);
      if (devices.length > 0) setSelectedDevice(devices[0].toString());
      addLog(`[${msTimestamp()}] Found ${devices.length} available CAN device(s)`);
    }).catch(() => {});
  };

  // Auto-detect CAN devices on mount
  useEffect(() => {
    detectCanDevices();
  }, []);

  // Refresh serial ports when switching to UART or after disconnect
  useEffect(() => {
    if (channel === "uart" && !connected) {
      CANUpgradeService.DetectSerialPorts().then((ports) => {
        setSerialPorts(ports);
        if (ports.length > 0 && !selectedPort) setSelectedPort(ports[0].portName);
      }).catch(() => {});
    }
  }, [channel, connected]);

  const handleConnect = async () => {
    try {
      if (connected) {
        if (channel === "can") {
          await CANUpgradeService.DisconnectCAN();
        } else {
          await CANUpgradeService.DisconnectUART();
        }
        setConnected(false);
        addLog(`[${msTimestamp()}] Disconnected`);
      } else {
        if (channel === "can") {
          await CANUpgradeService.ConnectCAN(parseInt(selectedDevice) || 0, baudIndex);
        } else {
          await CANUpgradeService.ConnectUART(selectedPort, parseInt(serialBaud));
        }
        setConnected(true);
        addLog(`[${msTimestamp()}] Connected successfully`);
      }
    } catch (err: any) {
      addLog(`[${msTimestamp()}] Error: ${err.message || err}`);
    }
  };

  const handleUpgrade = async () => {
    if (!firmwarePath) {
      addLog(`[${msTimestamp()}] ${t("fw.pleaseSelect")}`);
      return;
    }
    try {
      setProgress(0);
      addLog(`[${msTimestamp()}] Starting firmware upgrade: ${firmwarePath}`);
      if (channel === "can") {
        await CANUpgradeService.CANFirmwareUpgrade(firmwarePath, false);
      } else {
        await CANUpgradeService.UARTFirmwareUpgrade(firmwarePath, false);
      }
      setDialogConfig({ type: "alert", title: t("fw.startUpgrade"), message: t("fw.upgradeDone") });
      setDialogOpen(true);
    } catch (err: any) {
      addLog(`[${msTimestamp()}] Error: ${err.message || err}`);
    }
  };

  const handleQueryVersion = async () => {
    try {
      let ver: string;
      if (channel === "can") {
        ver = await CANUpgradeService.CANGetFirmwareVersion();
      } else {
        ver = await CANUpgradeService.UARTGetFirmwareVersion();
      }
      setVersion(ver);
      addLog(`[${msTimestamp()}] Firmware version: ${ver}`);
    } catch (err: any) {
      addLog(`[${msTimestamp()}] Error: ${err.message || err}`);
    }
  };

  const handleReboot = () => {
    setDialogConfig({
      type: "confirm",
      title: t("fw.reboot"),
      message: t("fw.rebootConfirm"),
      onConfirm: async () => {
        try {
          setProgress(0);
          if (channel === "can") {
            await CANUpgradeService.CANBoardReboot();
          } else {
            await CANUpgradeService.UARTBoardReboot();
          }
          addLog(`[${msTimestamp()}] Reboot command sent`);
        } catch (err: any) {
          addLog(`[${msTimestamp()}] Error: ${err.message || err}`);
        }
      },
    });
    setDialogOpen(true);
  };

  return (
    <div className="space-y-4">
      {/* Channel Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Upload className="h-4 w-4" /> {t("fw.channel")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            {/* Channel toggle switch */}
            <div className="relative flex items-center bg-muted rounded-md p-0.5 h-7 w-28">
              <div className={`absolute top-0.5 bottom-0.5 w-1/2 bg-primary rounded-[4px] transition-transform duration-200 ${channel === "uart" ? "translate-x-full" : "translate-x-0"}`} />
              <button
                onClick={() => {
                  setChannel("can");
                  detectCanDevices();
                }}
                className={`relative z-10 flex-1 text-xs font-medium text-center rounded transition-colors cursor-pointer ${channel === "can" ? "text-primary-foreground" : "text-muted-foreground"}`}
              >CAN</button>
              <button
                onClick={() => {
                  setChannel("uart");
                  // Auto-detect serial ports when switching to UART
                  CANUpgradeService.DetectSerialPorts().then((ports) => {
                    setSerialPorts(ports);
                    if (ports.length > 0 && !selectedPort) setSelectedPort(ports[0].portName);
                  }).catch(() => {});
                }}
                className={`relative z-10 flex-1 text-xs font-medium text-center rounded transition-colors cursor-pointer ${channel === "uart" ? "text-primary-foreground" : "text-muted-foreground"}`}
              >UART</button>
            </div>

            {channel === "can" ? (
              <>
                <Select value={selectedDevice} onChange={(e) => setSelectedDevice(e.target.value)} className="w-44">
                  {canDevices.map((d) => (
                    <option key={d} value={d.toString()}>Channel 0x{d.toString(16)}</option>
                  ))}
                </Select>
                <span className="text-sm text-muted-foreground shrink-0">{t("fw.baud")}:</span>
                <Select value={baudIndex.toString()} onChange={(e) => setBaudIndex(Number(e.target.value))} className="w-24">
                  {baudRates.map((br, i) => (
                    <option key={i} value={i.toString()}>{br}</option>
                  ))}
                </Select>
              </>
            ) : (
              <>
                <Select value={selectedPort} onChange={(e) => setSelectedPort(e.target.value)} className="w-44">
                  {serialPorts.map((p: SerialPortInfo) => (
                    <option key={p.portName} value={p.portName}>{p.friendlyName || p.portName}</option>
                  ))}
                </Select>
                <span className="text-sm text-muted-foreground shrink-0">{t("fw.baud")}:</span>
                <Select value={serialBaud} onChange={(e) => setSerialBaud(e.target.value)} className="w-24">
                  {serialBaudRates.map((br) => (
                    <option key={br} value={br}>{br}</option>
                  ))}
                </Select>
              </>
            )}

            <Button onClick={handleConnect} variant={connected ? "destructive" : "default"}>
              {connected ? t("lora.disconnect") : t("lora.conn")}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Firmware File + Progress & Controls */}
      <Card>
        <CardHeader>
          <CardTitle>{t("fw.firmwareFile")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <Input value={firmwarePath} onChange={(e) => setFirmwarePath(e.target.value)} placeholder={t("fw.selectFile")} className="flex-1" />
            <Button variant="outline" onClick={async () => {
              try {
                const path = await CANUpgradeService.OpenFirmwareFile();
                if (path) setFirmwarePath(path);
              } catch (err: any) {
                addLog(`[${msTimestamp()}] Error: ${err.message || err}`);
              }
            }}>{t("fw.browse")}</Button>
          </div>

          {/* Progress + Start Upgrade */}
          <div className="flex items-center gap-3">
            <Progress value={progress} className="flex-1" />
            <span className="text-sm text-muted-foreground font-mono w-10 text-right">{progress}%</span>
            <Button onClick={handleUpgrade} disabled={!connected || !firmwarePath}>{t("fw.startUpgrade")}</Button>
          </div>

          {/* Version + Query + Reboot */}
          <div className="flex items-center gap-3">
            <span className="text-sm text-muted-foreground shrink-0">{t("fw.curVersion")}: <span className="font-mono text-foreground">{version || "—"}</span></span>
            <div className="flex-1" />
            <Button variant="outline" size="sm" onClick={handleQueryVersion} disabled={!connected}>{t("fw.queryVer")}</Button>
            <Button variant="outline" size="sm" onClick={handleReboot} disabled={!connected}>{t("fw.reboot")}</Button>
          </div>
        </CardContent>
      </Card>

      {/* Log */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>{t("fw.log")}</CardTitle>
            <Button variant="ghost" size="icon" onClick={() => setLogs([])} title="Clear log">
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div ref={logRef} className="overflow-y-auto bg-terminal-bg rounded-md p-3 font-mono text-xs text-terminal-fg terminal-selectable terminal-xl">
            {logs.length === 0 ? (
              <p className="text-muted-foreground">{t("fw.waitOp")}</p>
            ) : (
              logs.map((log) => <LogLine key={log.id} text={log.text} />)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{dialogConfig.title}</DialogTitle>
            <DialogDescription>{dialogConfig.message}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            {dialogConfig.type === "confirm" && (
              <Button variant="outline" onClick={() => setDialogOpen(false)}>{t("fw.cancel")}</Button>
            )}
            <Button onClick={() => {
              setDialogOpen(false);
              if (dialogConfig.onConfirm) dialogConfig.onConfirm();
            }}>
              {dialogConfig.type === "confirm" ? t("fw.reboot") : t("fw.ok")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

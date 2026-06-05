import React, { useState, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Progress } from "@/components/ui/progress";
import { useWailsEvent } from "@/hooks/useEvents";
import { Upload, Trash2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { CANUpgradeService } from "../../bindings/github.com/kabirz/modhandlergo/service";

const baudRates = ["10K", "20K", "50K", "100K", "125K", "250K", "500K", "1M"];
const serialBaudRates = ["9600", "19200", "38400", "57600", "115200", "230400", "460800", "921600"];

export function FirmwareUpgradePage() {
  const { t } = useI18n();
  const [channel, setChannel] = useState<"can" | "uart">("can");
  const [baudIndex, setBaudIndex] = useState(6);
  const [serialBaud, setSerialBaud] = useState("115200");
  const [canDevices, setCanDevices] = useState<number[]>([]);
  const [selectedDevice, setSelectedDevice] = useState("");
  const [serialPorts, setSerialPorts] = useState<any[]>([]);
  const [selectedPort, setSelectedPort] = useState("");
  const [firmwarePath, setFirmwarePath] = useState("");
  const [connected, setConnected] = useState(false);
  const [progress, setProgress] = useState(0);
  const [version, setVersion] = useState("");
  const [logs, setLogs] = useState<string[]>([]);

  const addLog = useCallback((msg: string) => {
    setLogs((prev) => [...prev.slice(-200), msg]);
  }, []);

  useWailsEvent<string>("can:log", (msg) => addLog(msg));
  useWailsEvent<number>("can:progress", (pct) => setProgress(pct));
  useWailsEvent<string>("uart:log", (msg) => addLog(msg));
  useWailsEvent<number>("uart:progress", (pct) => setProgress(pct));

  const handleDetectDevices = async () => {
    try {
      const devices = await CANUpgradeService.DetectCANDevices();
      setCanDevices(devices);
      // Auto-select first device
      if (devices.length > 0) {
        setSelectedDevice(devices[0].toString());
      }
      addLog(`Detected ${devices.length} CAN device(s)`);
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
  };

  const handleDetectPorts = async () => {
    try {
      const ports = await CANUpgradeService.DetectSerialPorts();
      setSerialPorts(ports);
      // Auto-select first port
      if (ports.length > 0) {
        setSelectedPort(ports[0].portName);
      }
      addLog(`Detected ${ports.length} serial port(s)`);
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
  };

  const handleConnect = async () => {
    try {
      if (connected) {
        if (channel === "can") {
          await CANUpgradeService.DisconnectCAN();
        } else {
          await CANUpgradeService.DisconnectUART();
        }
        setConnected(false);
        addLog("Disconnected");
      } else {
        if (channel === "can") {
          await CANUpgradeService.ConnectCAN(parseInt(selectedDevice) || 0, baudIndex);
        } else {
          await CANUpgradeService.ConnectUART(selectedPort, parseInt(serialBaud));
        }
        setConnected(true);
        addLog("Connected successfully");
      }
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
  };

  const handleUpgrade = async () => {
    if (!firmwarePath) {
      addLog("Please select a firmware file first");
      return;
    }
    try {
      setProgress(0);
      addLog(`Starting firmware upgrade: ${firmwarePath}`);
      if (channel === "can") {
        await CANUpgradeService.CANFirmwareUpgrade(firmwarePath, false);
      } else {
        await CANUpgradeService.UARTFirmwareUpgrade(firmwarePath, false);
      }
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
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
      addLog(`Firmware version: ${ver}`);
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
  };

  const handleReboot = async () => {
    try {
      if (channel === "can") {
        await CANUpgradeService.CANBoardReboot();
      } else {
        await CANUpgradeService.UARTBoardReboot();
      }
      addLog("Reboot command sent");
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
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
            <Select value={channel} onChange={(e) => setChannel(e.target.value as "can" | "uart")} className="w-32">
              <option value="can">CAN</option>
              <option value="uart">UART</option>
            </Select>

            {channel === "can" ? (
              <>
                <Select value={selectedDevice} onChange={(e) => setSelectedDevice(e.target.value)} className="w-48">
                  <option value="">{t("fw.selectDev")}</option>
                  {canDevices.map((d, i) => (
                    <option key={i} value={d.toString()}>Channel 0x{d.toString(16)}</option>
                  ))}
                </Select>
                <Select value={baudIndex.toString()} onChange={(e) => setBaudIndex(Number(e.target.value))} className="w-24">
                  {baudRates.map((br, i) => (
                    <option key={i} value={i.toString()}>{br}</option>
                  ))}
                </Select>
                <Button variant="outline" size="sm" onClick={handleDetectDevices}>{t("fw.detectDev")}</Button>
              </>
            ) : (
              <>
                <Select value={selectedPort} onChange={(e) => setSelectedPort(e.target.value)} className="w-36">
                  <option value="">{t("fw.selectPort")}</option>
                  {serialPorts.map((p: any, i: number) => (
                    <option key={i} value={p.portName}>{p.friendlyName || p.portName}</option>
                  ))}
                </Select>
                <Select value={serialBaud} onChange={(e) => setSerialBaud(e.target.value)} className="w-32">
                  {serialBaudRates.map((br) => (
                    <option key={br} value={br}>{br}</option>
                  ))}
                </Select>
                <Button variant="outline" size="sm" onClick={handleDetectPorts}>{t("fw.detectPort")}</Button>
              </>
            )}

            <Button onClick={handleConnect} variant={connected ? "destructive" : "default"}>
              {connected ? t("lora.disconnect") : t("lora.conn")}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Firmware File */}
      <Card>
        <CardHeader>
          <CardTitle>{t("fw.firmwareFile")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Input value={firmwarePath} onChange={(e) => setFirmwarePath(e.target.value)} placeholder="Select firmware file (.bin)" className="flex-1" />
            <Button variant="outline" onClick={async () => {
              try {
                const path = await CANUpgradeService.OpenFirmwareFile();
                if (path) setFirmwarePath(path);
              } catch (err: any) {
                addLog(`Error: ${err.message || err}`);
              }
            }}>{t("fw.browse")}</Button>
          </div>
        </CardContent>
      </Card>

      {/* Progress & Controls */}
      <Card>
        <CardHeader>
          <CardTitle>{t("fw.upgrade")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Progress value={progress} />
          <p className="text-sm text-muted-foreground text-center">{progress}%</p>

          <div className="flex items-center gap-3 justify-center">
            <Button onClick={handleUpgrade} disabled={!connected || !firmwarePath}>{t("fw.startUpgrade")}</Button>
            <Button variant="outline" onClick={handleQueryVersion} disabled={!connected}>{t("fw.queryVer")}</Button>
            <Button variant="outline" onClick={handleReboot} disabled={!connected}>{t("fw.reboot")}</Button>
          </div>

          {version && <p className="text-center text-sm">{t("fw.curVersion")}: {version}</p>}
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
          <div className="overflow-y-auto bg-terminal-bg rounded-md p-3 font-mono text-xs text-terminal-fg terminal-selectable">
            {logs.length === 0 ? (
              <p className="text-muted-foreground">{t("fw.waitOp")}</p>
            ) : (
              logs.map((log, i) => <div key={i}>{log}</div>)
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

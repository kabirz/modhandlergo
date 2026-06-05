import React, { useState, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Progress } from "@/components/ui/progress";
import { useWailsEvent } from "@/hooks/useEvents";
import { Upload } from "lucide-react";

const baudRates = ["10K", "20K", "50K", "100K", "125K", "250K", "500K", "1M"];
const serialBaudRates = ["9600", "19200", "38400", "57600", "115200", "230400", "460800", "921600"];

export function FirmwareUpgradePage() {
  const [channel, setChannel] = useState<"can" | "uart">("can");
  const [baudIndex, setBaudIndex] = useState(6); // 500K
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

  const handleDetectDevices = () => {
    setCanDevices([0x51]); // placeholder
    addLog("正在检测 CAN 设备...");
  };

  const handleDetectPorts = () => {
    setSerialPorts([{ portName: "COM3", friendlyName: "COM3" }]); // placeholder
    addLog("正在枚举串口...");
  };

  const handleConnect = () => {
    setConnected(!connected);
    addLog(connected ? "已断开" : "连接中...");
  };

  const handleUpgrade = () => {
    if (!firmwarePath) {
      addLog("请先选择固件文件");
      return;
    }
    setProgress(0);
    addLog(`开始固件升级: ${firmwarePath}`);
  };

  const handleQueryVersion = () => {
    setVersion("v1.0.0 (查询中...)");
  };

  const handleReboot = () => {
    addLog("发送重启命令...");
  };

  return (
    <div className="space-y-4">
      {/* Channel Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Upload className="h-4 w-4" /> 通道选择
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
                  <option value="">选择设备</option>
                  {canDevices.map((d, i) => (
                    <option key={i} value={d.toString()}>
                      Channel 0x{d.toString(16)}
                    </option>
                  ))}
                </Select>
                <Select value={baudIndex.toString()} onChange={(e) => setBaudIndex(Number(e.target.value))} className="w-24">
                  {baudRates.map((br, i) => (
                    <option key={i} value={i.toString()}>{br}</option>
                  ))}
                </Select>
                <Button variant="outline" size="sm" onClick={handleDetectDevices}>检测设备</Button>
              </>
            ) : (
              <>
                <Select value={selectedPort} onChange={(e) => setSelectedPort(e.target.value)} className="w-36">
                  <option value="">选择串口</option>
                  {serialPorts.map((p, i) => (
                    <option key={i} value={p.portName}>{p.friendlyName}</option>
                  ))}
                </Select>
                <Select value={serialBaud} onChange={(e) => setSerialBaud(e.target.value)} className="w-32">
                  {serialBaudRates.map((br) => (
                    <option key={br} value={br}>{br}</option>
                  ))}
                </Select>
                <Button variant="outline" size="sm" onClick={handleDetectPorts}>检测串口</Button>
              </>
            )}

            <Button onClick={handleConnect} variant={connected ? "destructive" : "default"}>
              {connected ? "断开" : "连接"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Firmware File */}
      <Card>
        <CardHeader>
          <CardTitle>固件文件</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Input
              value={firmwarePath}
              onChange={(e) => setFirmwarePath(e.target.value)}
              placeholder="选择固件文件 (.bin)"
              className="flex-1"
            />
            <Button variant="outline" onClick={() => setFirmwarePath("firmware.bin")}>
              浏览
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Progress & Controls */}
      <Card>
        <CardHeader>
          <CardTitle>升级控制</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Progress value={progress} />
          <p className="text-sm text-muted-foreground text-center">{progress}%</p>

          <div className="flex items-center gap-3 justify-center">
            <Button onClick={handleUpgrade} disabled={!connected || !firmwarePath}>
              开始升级
            </Button>
            <Button variant="outline" onClick={handleQueryVersion} disabled={!connected}>
              查询版本
            </Button>
            <Button variant="outline" onClick={handleReboot} disabled={!connected}>
              重启板卡
            </Button>
          </div>

          {version && (
            <p className="text-center text-sm">当前版本: {version}</p>
          )}
        </CardContent>
      </Card>

      {/* Log */}
      <Card>
        <CardHeader>
          <CardTitle>日志</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-48 overflow-y-auto bg-black/30 rounded-md p-3 font-mono text-xs text-green-400">
            {logs.length === 0 ? (
              <p className="text-muted-foreground">等待操作...</p>
            ) : (
              logs.map((log, i) => <div key={i}>{log}</div>)
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

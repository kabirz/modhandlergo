import React, { useState, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useWailsEvent } from "@/hooks/useEvents";
import { Settings } from "lucide-react";
import { LoRaConfigService } from "../../bindings/github.com/kabirz/modhandlergo/service";

export function LoRaConfigPage() {
  const [transport, setTransport] = useState("udp");
  const [gatewayIP, setGatewayIP] = useState("192.168.1.100");
  const [serialPort, setSerialPort] = useState("");
  const [devices, setDevices] = useState<any[]>([]);
  const [atCmd, setAtCmd] = useState("");
  const [atResponse, setAtResponse] = useState("");
  const [logs, setLogs] = useState<string[]>([]);

  const addLog = useCallback((msg: string) => {
    setLogs((prev) => [...prev.slice(-200), msg]);
  }, []);

  useWailsEvent<any>("lora:device", (data) => {
    setDevices((prev) => [...prev, data]);
    addLog(`发现设备: ${data?.name} (${data?.ip})`);
  });

  useWailsEvent<string>("lora:atresponse", (resp) => {
    setAtResponse((prev) => prev + resp);
  });

  useWailsEvent<string>("lora:log", (msg) => {
    addLog(msg);
  });

  useWailsEvent<any>("lora:netparams", (params) => {
    addLog(`网络参数: IP=${params?.ip} 掩码=${params?.mask} 网关=${params?.gateway}`);
  });

  const handleSearch = async () => {
    try {
      setDevices([]);
      await LoRaConfigService.SearchDevices();
      addLog("开始搜索 LoRa 设备...");
    } catch (err: any) {
      addLog(`错误: ${err.message || err}`);
    }
  };

  const handleSendAT = async () => {
    if (!atCmd.trim()) return;
    try {
      addLog(`发送: ${atCmd}`);
      setAtResponse("");
      await LoRaConfigService.SendAT(atCmd, gatewayIP);
    } catch (err: any) {
      addLog(`错误: ${err.message || err}`);
    }
  };

  const handleReboot = async () => {
    try {
      addLog("发送重启命令...");
      await LoRaConfigService.Reboot(gatewayIP);
    } catch (err: any) {
      addLog(`错误: ${err.message || err}`);
    }
  };

  return (
    <div className="space-y-4">
      {/* Transport Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-4 w-4" /> 传输方式
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Select value={transport} onChange={(e) => setTransport(e.target.value)} className="w-32">
              <option value="udp">UDP 网络</option>
              <option value="serial">串口直连</option>
            </Select>
            {transport === "udp" ? (
              <Input value={gatewayIP} onChange={(e) => setGatewayIP(e.target.value)} placeholder="网关 IP" className="w-48" />
            ) : (
              <Input value={serialPort} onChange={(e) => setSerialPort(e.target.value)} placeholder="串口号 (如 COM3)" className="w-48" />
            )}
          </div>
        </CardContent>
      </Card>

      {/* Device Search */}
      <Card>
        <CardHeader>
          <CardTitle>设备搜索</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3 mb-3">
            <Button onClick={handleSearch}>搜索设备</Button>
            <Button variant="outline" onClick={handleReboot}>重启网关</Button>
          </div>
          {devices.length > 0 ? (
            <div className="space-y-2">
              {devices.map((d, i) => (
                <div key={i} className="flex items-center justify-between p-2 rounded-md bg-muted/50 text-sm">
                  <span>{d.name || d.mac}</span>
                  <span className="text-muted-foreground">{d.ip}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">暂无设备</p>
          )}
        </CardContent>
      </Card>

      {/* Network Settings */}
      <Card>
        <CardHeader>
          <CardTitle>网络设置</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="text-xs text-muted-foreground">IP 地址</label>
              <Input placeholder="192.168.1.100" className="mt-1" />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">子网掩码</label>
              <Input placeholder="255.255.255.0" className="mt-1" />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">网关</label>
              <Input placeholder="192.168.1.1" className="mt-1" />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* LoRa Protocol Settings */}
      <Card>
        <CardHeader>
          <CardTitle>LoRa 协议参数</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-3">
            <div>
              <label className="text-xs text-muted-foreground">组网模式</label>
              <Select className="mt-1"><option>NWMODE</option></Select>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">透传模式</label>
              <Select className="mt-1"><option>TTMODE</option></Select>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">通道</label>
              <Select className="mt-1"><option>CH</option></Select>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">速率</label>
              <Select className="mt-1"><option>SPD</option></Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* AT Console */}
      <Card>
        <CardHeader>
          <CardTitle>AT 命令控制台</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3 mb-3">
            <Input
              value={atCmd}
              onChange={(e) => setAtCmd(e.target.value)}
              placeholder="输入 AT 命令"
              className="flex-1"
              onKeyDown={(e) => e.key === "Enter" && handleSendAT()}
            />
            <Button onClick={handleSendAT}>发送</Button>
          </div>
          <div className="h-36 overflow-y-auto bg-terminal-bg rounded-md p-3 font-mono text-xs text-terminal-fg">
            {atResponse || <span className="text-muted-foreground">等待 AT 响应...</span>}
          </div>
        </CardContent>
      </Card>

      {/* Log */}
      <Card>
        <CardHeader>
          <CardTitle>日志</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-32 overflow-y-auto bg-terminal-bg rounded-md p-3 font-mono text-xs text-terminal-fg">
            {logs.map((log, i) => <div key={i}>{log}</div>)}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

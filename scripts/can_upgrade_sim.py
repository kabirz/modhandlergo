#!/usr/bin/env python3
"""
CAN Device Simulator

模拟嵌入式设备的完整 CAN 通信协议，用于测试 ModHandlerGo 的全部 CAN 功能。

支持:
  - 固件升级 (0x101/0x102/0x103)
  - LoRa 远程配参 (0x105/0x106)
  - 心跳 (0x763)
  - 手柄状态 (0x1E3)

Usage:
  python can_upgrade_sim.py [--channel vcan0] [--version 0.1.2.3]
    [--no-heartbeat] [--no-handler]
    [--handler-interval 0.1]

依赖: pip install python-can tqdm
"""

import argparse
import math
import struct
import sys
import threading
import time

try:
    import can
except ImportError:
    print("Error: python-can not installed. Run: pip install python-can")
    sys.exit(1)

try:
    from tqdm import tqdm
except ImportError:
    print("Error: tqdm not installed. Run: pip install tqdm")
    sys.exit(1)


# ════════════════════════════════════════════════════════════════
# CAN Frame IDs (mod-can.h)
# ════════════════════════════════════════════════════════════════

PLATFORM_RX     = 0x101   # Platform → Device: 控制命令
PLATFORM_TX     = 0x102   # Device → Platform: 响应帧
FW_DATA_RX      = 0x103   # Platform → Device: 固件数据
COBID_HEATBEAT  = 0x763   # Device → Platform: 心跳
HANDLER_STATE   = 0x1E3   # Device → Platform: 手柄状态
LORA_CONFIG_RX  = 0x105   # Platform → Device: LoRa 配参命令
LORA_CONFIG_TX  = 0x106   # Device → Platform: LoRa 配参响应

FRAME_ID_NAMES = {
    PLATFORM_RX:     "PLATFORM_RX",
    PLATFORM_TX:     "PLATFORM_TX",
    FW_DATA_RX:      "FW_DATA_RX",
    COBID_HEATBEAT:  "HEARTBEAT",
    HANDLER_STATE:   "HANDLER_STATE",
    LORA_CONFIG_RX:  "LORA_CONFIG",
    LORA_CONFIG_TX:  "LORA_CONFIG_RESP",
}

# ════════════════════════════════════════════════════════════════
# Firmware Upgrade Commands (0x101)
# ════════════════════════════════════════════════════════════════

CMD_START_UPDATE = 0
CMD_CONFIRM      = 1
CMD_VERSION      = 2
CMD_REBOOT       = 3

CMD_NAMES = {
    CMD_START_UPDATE: "START_UPDATE",
    CMD_CONFIRM:      "CONFIRM",
    CMD_VERSION:      "VERSION",
    CMD_REBOOT:       "REBOOT",
}

# ════════════════════════════════════════════════════════════════
# Firmware Upgrade Responses (0x102)
# ════════════════════════════════════════════════════════════════

FW_CODE_OFFSET      = 0
FW_CODE_SUCCESS     = 1
FW_CODE_VERSION     = 2
FW_CODE_CONFIRM     = 3
FW_CODE_FLASH_ERROR = 4
FW_CODE_TRANSFER_ERR = 5

FW_CODE_NAMES = {
    FW_CODE_OFFSET:       "OFFSET",
    FW_CODE_SUCCESS:      "SUCCESS",
    FW_CODE_VERSION:      "VERSION",
    FW_CODE_CONFIRM:      "CONFIRM",
    FW_CODE_FLASH_ERROR:  "FLASH_ERROR",
    FW_CODE_TRANSFER_ERR: "TRANSFER_ERROR",
}

CONFIRM_MAGIC = 0x55AA55AA

# ════════════════════════════════════════════════════════════════
# LoRa Config Commands (0x105, mod-can.h lora_config_cmd)
# ════════════════════════════════════════════════════════════════

LORA_CMD_SET_MODE   = 0x01
LORA_CMD_QUERY_MODE = 0x02
LORA_CMD_SET_CH1    = 0x03
LORA_CMD_QUERY_CH1  = 0x04
LORA_CMD_SET_CH2    = 0x05
LORA_CMD_QUERY_CH2  = 0x06
LORA_CMD_QUERY_NID  = 0x07
LORA_CMD_SET_NID    = 0x08
LORA_CMD_QUERY_GWID = 0x09
LORA_CMD_SET_GWID   = 0x0A
LORA_CMD_QUERY_PNUM = 0x0B
LORA_CMD_SET_PNUM   = 0x0C
LORA_CMD_SET_TEST   = 0x0D
LORA_CMD_SET_POWER  = 0x0F

LORA_CMD_NAMES = {
    LORA_CMD_SET_MODE:   "SET_MODE",
    LORA_CMD_QUERY_MODE: "QUERY_MODE",
    LORA_CMD_SET_CH1:    "SET_CH1",
    LORA_CMD_QUERY_CH1:  "QUERY_CH1",
    LORA_CMD_SET_CH2:    "SET_CH2",
    LORA_CMD_QUERY_CH2:  "QUERY_CH2",
    LORA_CMD_QUERY_NID:  "QUERY_NID",
    LORA_CMD_SET_NID:    "SET_NID",
    LORA_CMD_QUERY_GWID: "QUERY_GWID",
    LORA_CMD_SET_GWID:   "SET_GWID",
    LORA_CMD_QUERY_PNUM: "QUERY_PNUM",
    LORA_CMD_SET_PNUM:   "SET_PNUM",
    LORA_CMD_SET_TEST:   "SET_TEST",
    LORA_CMD_SET_POWER:  "SET_POWER",
}

# ════════════════════════════════════════════════════════════════
# Helper
# ════════════════════════════════════════════════════════════════


def log(msg: str):
    ts = time.strftime("%H:%M:%S")
    print(f"  [{ts}] {msg}")


def format_version(v: int) -> str:
    return f"v{(v >> 24) & 0xFF}.{(v >> 16) & 0xFF}.{(v >> 8) & 0xFF}"


def send_can(bus: can.Bus, arb_id: int, data: bytes, delay: float = 0):
    if delay > 0:
        time.sleep(delay)
    msg = can.Message(arbitration_id=arb_id, data=data, is_extended_id=False)
    bus.send(msg)


# ════════════════════════════════════════════════════════════════
# LoRa Simulator State
# ════════════════════════════════════════════════════════════════


class LoraState:
    """模拟设备端 LoRa 模块的可配置参数 (参考 global_params)"""

    def __init__(self):
        self._lock = threading.Lock()
        self.prot = 1          # 网络协议类型 (1=LG210)
        self.mode = 2          # 工作模式 (2=NET)
        self.spd1 = 7          # 通道1 速率
        self.ch1 = 4700       # 通道1 频率 (470.0MHz, 单位100KHz)
        self.spd2 = 7          # 通道2 速率
        self.ch2 = 4800       # 通道2 频率 (480.0MHz, 单位100KHz)
        self.pnum = 0          # 通道选择
        self.nid = 0x01020304  # 节点 ID
        self.gwid = 0x11223344 # 网关 ID
        self.test_mode = False # 测试模式
        self.power_on = True   # LoRa 电源状态

    def handle(self, frame_data: bytes) -> tuple[int, bytes] | None:
        """处理 LoRa 配参命令，返回 (cmd, response_data) 或 None"""
        if len(frame_data) < 1:
            return None

        cmd = frame_data[0]

        with self._lock:
            if cmd == LORA_CMD_SET_MODE:
                self.prot = frame_data[1] >> 4
                self.mode = frame_data[1] & 0x0F
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = (self.prot << 4) | (self.mode & 0x0F)
                log(f"LoRa SET_MODE: prot={self.prot} mode={self.mode}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_QUERY_MODE:
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = (self.prot << 4) | (self.mode & 0x0F)
                log(f"LoRa QUERY_MODE: prot={self.prot} mode={self.mode}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_SET_CH1:
                self.spd1 = frame_data[1]
                self.ch1 = struct.unpack(">H", frame_data[2:4])[0]
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = self.spd1
                struct.pack_into(">H", resp, 2, self.ch1)
                log(f"LoRa SET_CH1: spd1={self.spd1} ch1={self.ch1}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_QUERY_CH1:
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = self.spd1
                struct.pack_into(">H", resp, 2, self.ch1)
                log(f"LoRa QUERY_CH1: spd1={self.spd1} ch1={self.ch1}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_SET_CH2:
                self.spd2 = frame_data[1]
                self.ch2 = struct.unpack(">H", frame_data[2:4])[0]
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = self.spd2
                struct.pack_into(">H", resp, 2, self.ch2)
                log(f"LoRa SET_CH2: spd2={self.spd2} ch2={self.ch2}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_QUERY_CH2:
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = self.spd2
                struct.pack_into(">H", resp, 2, self.ch2)
                log(f"LoRa QUERY_CH2: spd2={self.spd2} ch2={self.ch2}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_SET_PNUM:
                self.pnum = frame_data[1]
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = self.pnum
                log(f"LoRa SET_PNUM: pnum={self.pnum}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_QUERY_PNUM:
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = self.pnum
                log(f"LoRa QUERY_PNUM: pnum={self.pnum}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_QUERY_NID:
                resp = bytearray(8)
                resp[0] = cmd
                struct.pack_into(">I", resp, 4, self.nid)
                log(f"LoRa QUERY_NID: nid=0x{self.nid:08X}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_SET_GWID:
                self.gwid = struct.unpack(">I", frame_data[4:8])[0]
                resp = bytearray(8)
                resp[0] = cmd
                struct.pack_into(">I", resp, 4, self.gwid)
                log(f"LoRa SET_GWID: gwid=0x{self.gwid:08X}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_QUERY_GWID:
                resp = bytearray(8)
                resp[0] = cmd
                struct.pack_into(">I", resp, 4, self.gwid)
                log(f"LoRa QUERY_GWID: gwid=0x{self.gwid:08X}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_SET_TEST:
                self.test_mode = bool(frame_data[1])
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = 1 if self.test_mode else 0
                log(f"LoRa SET_TEST: test_mode={self.test_mode}")
                return cmd, bytes(resp)

            elif cmd == LORA_CMD_SET_POWER:
                self.power_on = bool(frame_data[1])
                resp = bytearray(8)
                resp[0] = cmd
                resp[1] = 1 if self.power_on else 0
                log(f"LoRa SET_POWER: power={'ON' if self.power_on else 'OFF'}")
                return cmd, bytes(resp)

            else:
                log(f"LoRa unknown cmd: 0x{cmd:02X}")
                return None


# ════════════════════════════════════════════════════════════════
# Periodic Threads
# ════════════════════════════════════════════════════════════════


def heartbeat_thread(bus: can.Bus, stop_event: threading.Event):
    """周期发送心跳帧 0x763, data[0]=5, dlc=1, 间隔 800ms"""
    while not stop_event.is_set():
        send_can(bus, COBID_HEATBEAT, bytes([5]))
        stop_event.wait(0.8)


def handler_state_thread(bus: can.Bus, stop_event: threading.Event, interval: float):
    """周期发送手柄状态帧 0x1E3 (大端序)
    data[0-1]: X 角度 (int16 BE)
    data[2-3]: Y 角度 (int16 BE)
    data[4]:   按键 (bit0=btnHandler, bit1=btnBox)
    data[5-7]: 0xFF
    """
    t = 0.0
    while not stop_event.is_set():
        x = int(math.sin(t * 0.5) * 90)   # -90 ~ 90
        y = int(math.cos(t * 0.3) * 60)   # -60 ~ 60
        btn = 0x01 | (0x02 if int(t) % 5 == 0 else 0x00)  # btnHandler=1, btnBox 间歇

        data = bytearray(8)
        struct.pack_into(">h", data, 0, x)
        struct.pack_into(">h", data, 2, y)
        data[4] = btn
        data[5] = 0xFF
        data[6] = 0xFF
        data[7] = 0xFF

        send_can(bus, HANDLER_STATE, bytes(data))
        t += interval
        stop_event.wait(interval)


# ════════════════════════════════════════════════════════════════
# Main Simulator
# ════════════════════════════════════════════════════════════════


def run_simulator(args):
    print("CAN Device Simulator")
    print(f"  Channel: {args.channel}")
    print(f"  Simulated version: {format_version(args.version)} (0x{args.version:08X})")
    print()

    if sys.platform == "win32":
        can_iface = "pcan"
    else:
        can_iface = "socketcan"

    try:
        bus = can.Bus(interface=can_iface, channel=args.channel, receive_own_messages=False)
    except OSError as e:
        print(f"Error: Cannot open {args.channel}: {e}")
        if sys.platform == "win32":
            print("Hint: Ensure PCAN drivers are installed and device is connected")
        else:
            print(f"Hint: sudo ip link add dev {args.channel} type vcan && sudo ip link set {args.channel} up")
        sys.exit(1)

    print(f"  Listening on {args.channel}...")

    lora = LoraState()
    stop_event = threading.Event()

    # ── 启动周期线程 ──
    threads: list[threading.Thread] = []

    if not args.no_heartbeat:
        t = threading.Thread(target=heartbeat_thread, args=(bus, stop_event), daemon=True, name="heartbeat")
        t.start()
        threads.append(t)
        print("  [+] Heartbeat (0x763) every 800ms")

    if not args.no_handler:
        t = threading.Thread(target=handler_state_thread, args=(bus, stop_event, args.handler_interval), daemon=True, name="handler")
        t.start()
        threads.append(t)
        print(f"  [+] Handler state (0x1E3) every {args.handler_interval}s")

    print("\n  Waiting for commands...\n")

    # ── 固件升级状态 ──
    total_size = 0
    offset = 0
    firmware_data = bytearray()
    upgrading = False
    pbar: tqdm | None = None

    try:
        while True:
            msg = bus.recv(timeout=1.0)
            if msg is None:
                continue

            arb_id = msg.arbitration_id
            data = msg.data
            name = FRAME_ID_NAMES.get(arb_id, f"0x{arb_id:03X}")

            # ── 固件数据 (0x103) ──
            if arb_id == FW_DATA_RX:
                if not upgrading:
                    continue
                size = len(data)
                firmware_data.extend(data)
                offset += size

                if offset >= total_size:
                    if pbar:
                        pbar.n = total_size
                        pbar.refresh()
                        pbar.close()
                        pbar = None
                    send_can(bus, PLATFORM_TX, struct.pack(">II", FW_CODE_SUCCESS, total_size), delay=0.005)
                    upgrading = False
                elif offset % 64 == 0:
                    if pbar:
                        pbar.n = offset
                        pbar.refresh()
                    send_can(bus, PLATFORM_TX, struct.pack(">II", FW_CODE_OFFSET, offset), delay=0.005)
                continue

            # ── LoRa 配参 (0x105) ──
            if arb_id == LORA_CONFIG_RX:
                padded = bytes(data).ljust(8, b'\x00')
                cmd = padded[0]
                cmd_name = LORA_CMD_NAMES.get(cmd, f"0x{cmd:02X}")
                log(f"RX 0x{LORA_CONFIG_RX:03X} [LoRa {cmd_name}] data={padded.hex(' ')}")
                result = lora.handle(padded)
                if result is not None:
                    _, resp_data = result
                    send_can(bus, LORA_CONFIG_TX, resp_data)
                continue

            # ── 控制命令 (0x101) ──
            if arb_id == PLATFORM_RX:
                if len(data) < 8:
                    log(f"RX 0x{PLATFORM_RX:03X} [too short: {len(data)} bytes]")
                    continue

                cmd, val = struct.unpack(">II", data[0:8])
                cmd_name = CMD_NAMES.get(cmd, f"UNKNOWN({cmd})")
                log(f"RX 0x{PLATFORM_RX:03X} [{cmd_name}] val=0x{val:08X} ({val})")

                if cmd == CMD_START_UPDATE:
                    total_size = val
                    offset = 0
                    firmware_data = bytearray()
                    upgrading = True
                    log(f"Starting firmware upgrade, size={total_size} bytes")
                    send_can(bus, PLATFORM_TX, struct.pack(">II", FW_CODE_OFFSET, 0), delay=0.005)
                    log("Flash erase complete")
                    pbar = tqdm(total=total_size, unit="B", unit_scale=True, desc="  Firmware", leave=True)

                elif cmd == CMD_CONFIRM:
                    if offset < total_size:
                        log(f"Confirm failed: offset={offset} < total={total_size}")
                        send_can(bus, PLATFORM_TX, struct.pack(">II", FW_CODE_TRANSFER_ERR, offset), delay=0.005)
                    else:
                        mode = "real" if val == 1 else "test"
                        log(f"Firmware confirmed ({mode} mode), {len(firmware_data)} bytes received")
                        if val == 1:
                            fname = f"firmware_received_{int(time.time())}.bin"
                            with open(fname, "wb") as f:
                                f.write(firmware_data)
                            log(f"Firmware saved to {fname}")
                        send_can(bus, PLATFORM_TX, struct.pack(">II", FW_CODE_CONFIRM, CONFIRM_MAGIC), delay=0.005)
                        upgrading = False

                elif cmd == CMD_VERSION:
                    send_can(bus, PLATFORM_TX, struct.pack(">II", FW_CODE_VERSION, args.version), delay=0.005)
                    log(f"Version: {format_version(args.version)}")

                elif cmd == CMD_REBOOT:
                    log("Reboot command received, resetting state")
                    upgrading = False
                    firmware_data = bytearray()
                    total_size = 0
                    offset = 0
                    if pbar:
                        pbar.close()
                        pbar = None

                else:
                    log(f"Unknown command: {cmd}")
                continue

            # ── 扫描仪/心跳/手柄帧 (仅日志) ──
            log(f"RX {name} (0x{arb_id:03X}) data={data.hex(' ')}")

    except KeyboardInterrupt:
        print("\n  Shutting down...")
    finally:
        stop_event.set()
        for t in threads:
            t.join(timeout=2.0)
        if pbar:
            pbar.close()
        bus.shutdown()


# ════════════════════════════════════════════════════════════════
# CLI
# ════════════════════════════════════════════════════════════════


def main():
    default_channel = "PCAN_USBBUS1" if sys.platform == "win32" else "vcan0"
    parser = argparse.ArgumentParser(description="CAN Device Simulator")
    parser.add_argument("--channel", default=default_channel, help=f"CAN channel (default: {default_channel})")
    parser.add_argument("--version", default="0x00010203",
                        help="Simulated firmware version as hex (default: 0x00010203)")
    parser.add_argument("--no-heartbeat", action="store_true", help="Disable heartbeat sending")
    parser.add_argument("--no-handler", action="store_true", help="Disable handler state sending")
    parser.add_argument("--handler-interval", type=float, default=0.1,
                        help="Handler state send interval in seconds (default: 0.1)")
    args = parser.parse_args()

    try:
        args.version = int(args.version, 0)
    except ValueError:
        print(f"Error: Invalid version format: {args.version}")
        sys.exit(1)

    run_simulator(args)


if __name__ == "__main__":
    main()

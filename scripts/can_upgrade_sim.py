#!/usr/bin/env python3
"""
CAN Firmware Upgrade Simulator

模拟嵌入式设备的 CAN 固件升级协议，用于测试 ModHandlerGo 的升级功能。

协议帧 ID:
  0x101  Platform → Device  控制命令 (StartUpdate/Confirm/Version/Reboot)
  0x102  Device → Platform  响应帧
  0x103  Platform → Device  固件数据

Usage:
  python can_upgrade_sim.py [--channel vcan0] [--version 0.1.2.3]

依赖: pip install python-can tqdm
"""

import argparse
import struct
import sys
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

# ── CAN Frame IDs ──

PLATFORM_RX = 0x101  # Platform → Device (control commands)
PLATFORM_TX = 0x102  # Device → Platform (response)
FW_DATA_RX = 0x103   # Platform → Device (firmware data)

# ── Command Codes (0x101 payload[0:4]) ──

CMD_START_UPDATE = 0
CMD_CONFIRM = 1
CMD_VERSION = 2
CMD_REBOOT = 3

CMD_NAMES = {
    CMD_START_UPDATE: "START_UPDATE",
    CMD_CONFIRM: "CONFIRM",
    CMD_VERSION: "VERSION",
    CMD_REBOOT: "REBOOT",
}

# ── Response Codes (0x102 payload[0:4]) ──

FW_CODE_OFFSET = 0
FW_CODE_SUCCESS = 1
FW_CODE_VERSION = 2
FW_CODE_CONFIRM = 3
FW_CODE_FLASH_ERROR = 4
FW_CODE_TRANSFER_ERR = 5

FW_CODE_NAMES = {
    FW_CODE_OFFSET: "OFFSET",
    FW_CODE_SUCCESS: "SUCCESS",
    FW_CODE_VERSION: "VERSION",
    FW_CODE_CONFIRM: "CONFIRM",
    FW_CODE_FLASH_ERROR: "FLASH_ERROR",
    FW_CODE_TRANSFER_ERR: "TRANSFER_ERROR",
}

CONFIRM_MAGIC = 0x55AA55AA


def send_response(bus: can.Bus, code: int, val: int):
    time.sleep(0.005)
    data = struct.pack(">II", code, val)
    msg = can.Message(arbitration_id=PLATFORM_TX, data=data, is_extended_id=False)
    bus.send(msg)


def parse_command(data: bytes) -> tuple[int, int]:
    if len(data) < 8:
        return 0xFFFFFFFF, 0
    code = struct.unpack(">I", data[0:4])[0]
    val = struct.unpack(">I", data[4:8])[0]
    return code, val


def log(msg: str):
    ts = time.strftime("%H:%M:%S")
    print(f"  [{ts}] {msg}")


def run_simulator(channel: str, version: int):
    print("CAN Firmware Upgrade Simulator")
    print(f"  Channel: {channel}")
    print(f"  Simulated version: {format_version(version)} (0x{version:08X})")
    print()

    try:
        bus = can.Bus(interface="socketcan", channel=channel, receive_own_messages=False)
    except OSError as e:
        print(f"Error: Cannot open {channel}: {e}")
        print(f"Hint: sudo ip link add dev {channel} type vcan && sudo ip link set {channel} up")
        sys.exit(1)

    print(f"  Listening on {channel}...")
    print("  Waiting for commands...\n")

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

            # Handle firmware data frames (0x103)
            if msg.arbitration_id == FW_DATA_RX:
                if not upgrading:
                    continue

                size = len(msg.data)
                firmware_data.extend(msg.data)
                offset += size

                if offset >= total_size:
                    if pbar:
                        pbar.n = total_size
                        pbar.refresh()
                        pbar.close()
                        pbar = None
                    send_response(bus, FW_CODE_SUCCESS, total_size)
                    upgrading = False
                elif offset % 64 == 0:
                    if pbar:
                        pbar.n = offset
                        pbar.refresh()
                    send_response(bus, FW_CODE_OFFSET, offset)

                continue

            # Handle control command frames (0x101)
            if msg.arbitration_id != PLATFORM_RX:
                continue

            cmd, val = parse_command(msg.data)
            cmd_name = CMD_NAMES.get(cmd, f"UNKNOWN({cmd})")
            log(f"RX 0x{PLATFORM_RX:03X} [{cmd_name}] val=0x{val:08X} ({val})")

            if cmd == CMD_START_UPDATE:
                total_size = val
                offset = 0
                firmware_data = bytearray()
                upgrading = True
                log(f"Starting firmware upgrade, size={total_size} bytes")
                send_response(bus, FW_CODE_OFFSET, 0)
                log("Flash erase complete")
                pbar = tqdm(total=total_size, unit="B", unit_scale=True, desc="  Firmware", leave=True)

            elif cmd == CMD_CONFIRM:
                if offset < total_size:
                    log(f"Confirm failed: offset={offset} < total={total_size}")
                    send_response(bus, FW_CODE_TRANSFER_ERR, offset)
                else:
                    mode = "real" if val == 1 else "test"
                    log(f"Firmware confirmed ({mode} mode), {len(firmware_data)} bytes received")
                    if val == 1:
                        fname = f"firmware_received_{int(time.time())}.bin"
                        with open(fname, "wb") as f:
                            f.write(firmware_data)
                        log(f"Firmware saved to {fname}")
                    send_response(bus, FW_CODE_CONFIRM, CONFIRM_MAGIC)
                    upgrading = False

            elif cmd == CMD_VERSION:
                send_response(bus, FW_CODE_VERSION, version)
                log(f"Version: {format_version(version)}")

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

    except KeyboardInterrupt:
        print("\n  Simulator stopped.")
    finally:
        if pbar:
            pbar.close()
        bus.shutdown()


def format_version(v: int) -> str:
    return f"v{(v >> 24) & 0xFF}.{(v >> 16) & 0xFF}.{(v >> 8) & 0xFF}"


def main():
    parser = argparse.ArgumentParser(description="CAN Firmware Upgrade Simulator")
    parser.add_argument("--channel", default="vcan0", help="CAN channel (default: vcan0)")
    parser.add_argument(
        "--version",
        default="0x00010203",
        help="Simulated firmware version as hex (default: 0x00010203 = v0.1.2.3)",
    )
    args = parser.parse_args()

    try:
        version = int(args.version, 0)
    except ValueError:
        print(f"Error: Invalid version format: {args.version}")
        sys.exit(1)

    run_simulator(args.channel, version)


if __name__ == "__main__":
    main()

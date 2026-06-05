import React from "react";
import { cn } from "@/lib/utils";
import {
  Radio,
  Settings,
  Upload,
  Terminal,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  id: string;
  label: string;
  icon: LucideIcon;
}

export const navItems: NavItem[] = [
  { id: "lora-data", label: "LoRa 数据", icon: Radio },
  { id: "lora-config", label: "LoRa 配置", icon: Settings },
  { id: "firmware", label: "固件升级", icon: Upload },
  { id: "can-command", label: "CAN 命令", icon: Terminal },
];

interface SidebarProps {
  activePage: string;
  onNavigate: (pageId: string) => void;
}

export function Sidebar({ activePage, onNavigate }: SidebarProps) {
  return (
    <aside className="w-[200px] min-w-[200px] h-screen bg-sidebar border-r border-sidebar-border flex flex-col">
      {/* Logo / Title */}
      <div className="px-4 py-5 border-b border-sidebar-border">
        <h1 className="text-sm font-bold text-sidebar-foreground tracking-wide">
          ModHandlerGo
        </h1>
        <p className="text-[11px] text-muted-foreground mt-1">
          激光测距系统配套工具
        </p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-2 px-2 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activePage === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onNavigate(item.id)}
              className={cn(
                "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors cursor-pointer",
                isActive
                  ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                  : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
              )}
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-sidebar-border">
        <p className="text-[10px] text-muted-foreground">v0.1.0 · Go + Wails v3</p>
      </div>
    </aside>
  );
}

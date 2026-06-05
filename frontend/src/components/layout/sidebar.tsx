import React from "react";
import { cn } from "@/lib/utils";
import {
  Radio,
  Settings,
  Upload,
  Terminal,
  Sun,
  Moon,
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

export const APP_VERSION = "0.1.0";

interface SidebarProps {
  activePage: string;
  onNavigate: (pageId: string) => void;
  darkMode: boolean;
  onToggleTheme: () => void;
}

export function Sidebar({ activePage, onNavigate, darkMode, onToggleTheme }: SidebarProps) {
  return (
    <aside className="w-[200px] min-w-[200px] h-screen bg-sidebar border-r border-sidebar-border flex flex-col transition-colors duration-300">
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

      {/* Footer: Theme + Version */}
      <div className="px-2 py-3 border-t border-sidebar-border space-y-2">
        {/* Theme Toggle */}
        <div className="flex items-center gap-1 px-2">
          <span className="text-[11px] text-muted-foreground shrink-0">主题：</span>
          <button
            onClick={() => { if (darkMode) onToggleTheme(); }}
            className={cn(
              "flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[11px] cursor-pointer transition-colors",
              !darkMode
                ? "bg-primary text-primary-foreground"
                : "bg-background text-foreground border border-input hover:bg-muted"
            )}
          >
            <Sun className="h-3 w-3" />
            亮色
          </button>
          <button
            onClick={() => { if (!darkMode) onToggleTheme(); }}
            className={cn(
              "flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[11px] cursor-pointer transition-colors",
              darkMode
                ? "bg-primary text-primary-foreground"
                : "bg-background text-foreground border border-input hover:bg-muted"
            )}
          >
            <Moon className="h-3 w-3" />
            暗色
          </button>
        </div>

        {/* Version */}
        <div className="px-2">
          <span className="text-[11px] text-muted-foreground">
            版本：v{APP_VERSION}
          </span>
        </div>
      </div>
    </aside>
  );
}

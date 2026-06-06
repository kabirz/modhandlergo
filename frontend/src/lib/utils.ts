import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format current time as HH:mm:ss.SSS (24h, Chinese locale) */
export function msTimestamp(): string {
  const now = new Date();
  return `${now.toLocaleTimeString("zh-CN", { hour12: false })}.${String(now.getMilliseconds()).padStart(3, "0")}`;
}

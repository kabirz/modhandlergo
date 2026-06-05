import { useEffect } from "react";
import { Events } from "@wailsio/runtime";

export function useWailsEvent<T = any>(
  eventName: string,
  callback: (data: T) => void
) {
  useEffect(() => {
    const cancel = Events.On(eventName, (ev: any) => {
      callback(ev.data as T);
    });
    return () => {
      if (typeof cancel === "function") cancel();
    };
  }, [eventName, callback]);
}

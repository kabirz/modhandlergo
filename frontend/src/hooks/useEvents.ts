import { useEffect, useRef } from "react";
import { Events } from "@wailsio/runtime";

export function useWailsEvent<T = any>(
  eventName: string,
  callback: (data: T) => void
) {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  useEffect(() => {
    const cancel = Events.On(eventName, (ev: any) => {
      callbackRef.current(ev.data as T);
    });
    return () => {
      if (typeof cancel === "function") cancel();
    };
  }, [eventName]);
}

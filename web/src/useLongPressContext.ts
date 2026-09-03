import { useEffect, useRef } from "react";
import type {
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
} from "react";

type Point = { x: number; y: number };

/**
 * Makes a context action equally reachable with a mouse, pen, touchscreen, or
 * keyboard. The timer does not prevent normal vertical scrolling; moving more
 * than the tolerance cancels the gesture.
 */
export function useLongPressContext<T>(
  onOpen: (value: T, point: Point) => void,
  delay = 560,
) {
  const timer = useRef<number | null>(null);
  const start = useRef<Point | null>(null);
  const activePointer = useRef<number | null>(null);
  const suppressClick = useRef(false);

  const cancel = () => {
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = null;
    start.current = null;
    activePointer.current = null;
  };

  useEffect(() => cancel, []);

  return (value: T) => ({
    onPointerDown: (event: ReactPointerEvent<HTMLElement>) => {
      if (event.pointerType !== "touch" && event.pointerType !== "pen") return;
      cancel();
      const point = { x: event.clientX, y: event.clientY };
      start.current = point;
      activePointer.current = event.pointerId;
      timer.current = window.setTimeout(() => {
        suppressClick.current = true;
        onOpen(value, point);
        timer.current = null;
        navigator.vibrate?.(12);
      }, delay);
    },
    onPointerMove: (event: ReactPointerEvent<HTMLElement>) => {
      if (event.pointerId !== activePointer.current || !start.current) return;
      if (Math.hypot(event.clientX - start.current.x, event.clientY - start.current.y) > 12)
        cancel();
    },
    onPointerUp: cancel,
    onPointerCancel: cancel,
    onClickCapture: (event: ReactMouseEvent<HTMLElement>) => {
      if (!suppressClick.current) return;
      suppressClick.current = false;
      event.preventDefault();
      event.stopPropagation();
    },
    onKeyDown: (event: React.KeyboardEvent<HTMLElement>) => {
      if (event.key !== "ContextMenu" && !(event.shiftKey && event.key === "F10")) return;
      event.preventDefault();
      const rect = event.currentTarget.getBoundingClientRect();
      onOpen(value, { x: rect.left + Math.min(rect.width / 2, 36), y: rect.top + rect.height / 2 });
    },
  });
}

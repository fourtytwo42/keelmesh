import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

type Help = { text: string; x: number; y: number } | null;

function helpText(target: Element | null) {
  const element = target?.closest<HTMLElement>("[data-help],[title],button,[aria-label],input,select,textarea");
  if (!element || element.closest(".hover-help")) return "";
  return (
    element.dataset.help ||
    element.getAttribute("title") ||
    element.getAttribute("aria-label") ||
    (element instanceof HTMLButtonElement ? element.innerText.trim() : "")
  );
}

export function HoverHelp() {
  const [help, setHelp] = useState<Help>(null);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => {
    const show = (event: Event) => {
      window.clearTimeout(timer.current);
      const text = helpText(event.target as Element);
      if (!text) return setHelp(null);
      const pointer = event as PointerEvent;
      const target = event.target as HTMLElement;
      const rect = target.getBoundingClientRect();
      const x = "clientX" in pointer && pointer.clientX ? pointer.clientX : rect.left + rect.width / 2;
      const y = "clientY" in pointer && pointer.clientY ? pointer.clientY : rect.bottom;
      timer.current = window.setTimeout(() => setHelp({ text, x, y }), 420);
    };
    const hide = () => { window.clearTimeout(timer.current); setHelp(null); };
    document.addEventListener("pointerover", show, true);
    document.addEventListener("pointerout", hide, true);
    document.addEventListener("focusin", show, true);
    document.addEventListener("focusout", hide, true);
    document.addEventListener("pointerdown", hide, true);
    return () => {
      window.clearTimeout(timer.current);
      document.removeEventListener("pointerover", show, true);
      document.removeEventListener("pointerout", hide, true);
      document.removeEventListener("focusin", show, true);
      document.removeEventListener("focusout", hide, true);
      document.removeEventListener("pointerdown", hide, true);
    };
  }, []);
  if (!help) return null;
  return createPortal(
    <div className="hover-help" role="tooltip" style={{ left: help.x, top: help.y }}>{help.text}</div>,
    document.body,
  );
}

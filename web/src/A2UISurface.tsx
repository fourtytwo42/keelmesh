import { useEffect, useMemo, useState } from "react";
import { MessageProcessor } from "@a2ui/web_core/v0_9";
import { A2uiSurface, MarkdownContext, basicCatalog } from "@a2ui/react/v0_9";
import { renderMarkdown } from "@a2ui/markdown-it";
import type { WorkspaceSurfaceV1 } from "./types";

export function KeelMeshA2UISurface({ surface }: { surface: WorkspaceSurfaceV1 }) {
  const processor = useMemo(() => new MessageProcessor([basicCatalog]), [surface.id, surface.sequence]);
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    // @a2ui/react 0.11 currently exposes its renderer through the v0_9 entry
    // point. KeelMesh receives the v1.0 ordered envelope and adapts only the
    // protocol/catalog identifiers; component data is never interpreted as
    // HTML or executable UI.
    const renderMessages = surface.messages.map((message) => {
      const normalized: Record<string, unknown> = { ...message, version: "v0.9" };
      if (normalized.createSurface && typeof normalized.createSurface === "object")
        normalized.createSurface = { ...(normalized.createSurface as Record<string, unknown>), catalogId: basicCatalog.id };
      return normalized;
    });
    processor.processMessages(renderMessages as unknown as Parameters<typeof processor.processMessages>[0]);
    setRevision((value) => value + 1);
  }, [processor, surface.messages, surface.sequence]);
  const model = processor.model.surfacesMap.get(surface.id);
  return (
    <div className="km-a2ui-surface" data-surface-id={surface.id} data-sequence={surface.sequence} data-revision={revision}>
      {model ? <MarkdownContext.Provider value={renderMarkdown}><A2uiSurface surface={model} /></MarkdownContext.Provider> : <span>Validating command surface…</span>}
    </div>
  );
}

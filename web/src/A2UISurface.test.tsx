import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { KeelMeshA2UISurface } from "./A2UISurface";

describe("KeelMeshA2UISurface", () => {
  it("renders an ordered trusted surface without evaluating markup", async () => {
    render(<KeelMeshA2UISurface surface={{
      id: "surface-test", role: "primary", title: "Operational Brief", sequence: 3,
      messages: [
        { version: "v1.0", createSurface: { surfaceId: "surface-test", catalogId: "https://keelmesh.local/catalogs/keelmesh-operations-v1" } },
        { version: "v1.0", updateComponents: { surfaceId: "surface-test", components: [
          { id: "root", component: "Column", children: ["title", "summary"] },
          { id: "title", component: "Text", text: { path: "/title" } },
          { id: "summary", component: "Text", text: { path: "/summary" } },
        ] } },
        { version: "v1.0", updateDataModel: { surfaceId: "surface-test", path: "/", value: { title: "Yellow Group", summary: "Six vessels ready." } } },
      ],
    }} />);
    expect(await screen.findByText("Yellow Group")).toBeTruthy();
    expect(screen.getByText("Six vessels ready.")).toBeTruthy();
  });

  it("keeps the rendered surface mounted while live bindings advance", async () => {
    const surface = {
      id: "surface-live",
      role: "primary" as const,
      title: "Vessel status",
      sequence: 1,
      messages: [
        { version: "v1.0", createSurface: { surfaceId: "surface-live", catalogId: "https://keelmesh.local/catalogs/keelmesh-operations-v1" } },
        { version: "v1.0", updateComponents: { surfaceId: "surface-live", components: [
          { id: "root", component: "Column", children: ["status"] },
          { id: "status", component: "Text", text: { path: "/status" } },
        ] } },
        { version: "v1.0", updateDataModel: { surfaceId: "surface-live", path: "/", value: { status: "Holding" } } },
      ],
    };
    const view = render(<KeelMeshA2UISurface surface={surface} />);
    await screen.findByText("Holding");
    const mountedSurface = view.container.querySelector(".km-a2ui-surface");

    view.rerender(<KeelMeshA2UISurface surface={{
      ...surface,
      sequence: 2,
      messages: surface.messages.map((message, index) => index === 2
        ? { version: "v1.0", updateDataModel: { surfaceId: "surface-live", path: "/", value: { status: "Underway" } } }
        : message),
    }} />);

    expect(view.container.querySelector(".km-a2ui-surface")).toBe(mountedSurface);
    expect(screen.queryByText("Validating command surface…")).toBeNull();
    expect(await screen.findByText("Underway")).toBeTruthy();
  });
});

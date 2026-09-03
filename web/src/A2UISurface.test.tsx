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
});

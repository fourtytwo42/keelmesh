import { Component, StrictMode, type ErrorInfo, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import "./app.css";

class StartupBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("KeelMesh workspace startup failed", error, info.componentStack);
  }

  render() {
    if (this.state.failed) {
      return (
        <main className="startup-failure" role="alert">
          <div className="startup-failure__mark">KM</div>
          <p className="startup-failure__eyebrow">WORKSPACE RECOVERY</p>
          <h1>KeelMesh could not finish loading.</h1>
          <p>The mission authority remains separate. Reload to fetch the current workspace bundle and snapshot.</p>
          <button type="button" onClick={() => window.location.reload()}>Reload workspace</button>
        </main>
      );
    }
    return this.props.children;
  }
}

function renderBootstrapFailure(error: unknown) {
  console.error("KeelMesh workspace bootstrap failed", error);
  const root = document.getElementById("root");
  if (!root) return;
  root.innerHTML = `
    <main class="startup-failure" role="alert">
      <div class="startup-failure__mark">KM</div>
      <p class="startup-failure__eyebrow">WORKSPACE RECOVERY</p>
      <h1>KeelMesh could not load this browser session.</h1>
      <p>The mission authority remains separate. Reload to fetch a compatible workspace bundle and current snapshot.</p>
      <button type="button" data-reload-workspace>Reload workspace</button>
    </main>`;
  root.querySelector("[data-reload-workspace]")?.addEventListener("click", () => window.location.reload());
}

async function bootstrap() {
  try {
    // Keep the application in a separate chunk so import/initialization
    // failures in an embedded WebView produce a recovery surface rather
    // than an unexplained dark screen.
    const { default: App } = await import("./App");
    const root = document.getElementById("root");
    if (!root) throw new Error("Missing application root");
    createRoot(root).render(<StrictMode><StartupBoundary><App /></StartupBoundary></StrictMode>);
  } catch (error) {
    renderBootstrapFailure(error);
  }
}

void bootstrap();

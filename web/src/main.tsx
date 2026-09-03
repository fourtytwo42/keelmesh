import { Component, StrictMode, type ErrorInfo, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
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

createRoot(document.getElementById("root")!).render(<StrictMode><StartupBoundary><App /></StartupBoundary></StrictMode>);

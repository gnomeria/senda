import { render } from "solid-js/web";
import App from "./App";
import "./styles.css";

// In a plain browser there's no Wails runtime (window.go), so install a mock
// backend for dev. The dynamic import + import.meta.env.DEV guard keeps the
// mock out of production builds.
async function boot() {
  if (import.meta.env.DEV && !(window as any).go) {
    const { installDevMock } = await import("./lib/devMock");
    installDevMock();
  }
  render(() => <App />, document.getElementById("root")!);
}

boot();

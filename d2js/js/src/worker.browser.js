let currentPort;
let d2;

function loadScript(content) {
  const func = new Function(content);
  func.call(globalThis);
}

// Load ELK library for WASM environment
function loadELK() {
  if (typeof globalThis.ELK === "undefined") {
    try {
      console.log("Loading ELK library in browser...");
      // Load the ELK library from the files
      // In a real browser implementation, these would be loaded via fetch
      // For now, we'll throw an error if ELK is not available
      throw new Error("ELK library not available in browser environment");
    } catch (err) {
      console.error("Failed to load ELK library in browser:", err);
      throw new Error("ELK library is required but not available in browser environment");
    }
  }
}

export function setupMessageHandler(isNode, port, initWasm) {
  currentPort = port;

  const handleMessage = async (e) => {
    const { type, data } = e;

    switch (type) {
      case "init":
        try {
          if (isNode) {
            loadScript(data.wasmExecContent);
          }
          loadELK(); // Load ELK library before initializing WASM
          d2 = await initWasm(data.wasm);
          currentPort.postMessage({ type: "ready" });
        } catch (err) {
          currentPort.postMessage({ type: "error", error: err.message });
        }
        break;

      case "compile":
        try {
          const result = await d2.compile(JSON.stringify(data));
          const response = JSON.parse(result);
          if (response.error) throw new Error(response.error.message);
          currentPort.postMessage({ type: "result", data: response.data });
        } catch (err) {
          currentPort.postMessage({ type: "error", error: err.message });
        }
        break;

      case "render":
        try {
          const result = await d2.render(JSON.stringify(data));
          const response = JSON.parse(result);
          if (response.error) throw new Error(response.error.message);
          const decoded = new TextDecoder().decode(
            Uint8Array.from(atob(response.data), (c) => c.charCodeAt(0))
          );
          currentPort.postMessage({ type: "result", data: decoded });
        } catch (err) {
          currentPort.postMessage({ type: "error", error: err.message });
        }
        break;
    }
  };

  if (isNode) {
    port.on("message", handleMessage);
  } else {
    port.onmessage = (e) => handleMessage(e.data);
  }
}

async function initWasmBrowser(wasmBinary) {
  const go = new Go();
  const result = await WebAssembly.instantiate(wasmBinary, go.importObject);
  go.run(result.instance);
  return self.d2;
}

setupMessageHandler(false, self, initWasmBrowser);

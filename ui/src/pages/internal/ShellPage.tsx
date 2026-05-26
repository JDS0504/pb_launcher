import { useEffect, useRef } from "react";
import { Terminal as XTerm } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebLinksAddon } from "xterm-addon-web-links";
import { buildShellWsUrl } from "../../services/shell";
import "xterm/css/xterm.css";

export const ShellPage = () => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTerm | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // ── Instanciar xterm ───────────────────────────────────────────────────
    const term = new XTerm({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: '"Cascadia Code", "JetBrains Mono", "Fira Code", monospace',
      theme: {
        background: "#0d0f14",
        foreground: "#cdd6f4",
        cursor: "#f5c2e7",
        cursorAccent: "#0d0f14",
        selectionBackground: "#585b7080",
        black: "#45475a",
        red: "#f38ba8",
        green: "#a6e3a1",
        yellow: "#f9e2af",
        blue: "#89b4fa",
        magenta: "#f5c2e7",
        cyan: "#94e2d5",
        white: "#bac2de",
        brightBlack: "#585b70",
        brightRed: "#f38ba8",
        brightGreen: "#a6e3a1",
        brightYellow: "#f9e2af",
        brightBlue: "#89b4fa",
        brightMagenta: "#f5c2e7",
        brightCyan: "#94e2d5",
        brightWhite: "#a6adc8",
      },
      scrollback: 5000,
      convertEol: true,
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(container);
    
    // Enfocar automáticamente el terminal para poder escribir de inmediato
    term.focus();
    fitAddon.fit();

    termRef.current = term;
    fitRef.current = fitAddon;

    // ── Conectar WebSocket ────────────────────────────────────────────────
    const wsUrl = buildShellWsUrl();
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.binaryType = "arraybuffer";

    ws.onopen = () => {
      term.writeln("\x1b[1;32m[CONECTADO] Shell interactiva lista.\x1b[0m");
    };

    ws.onerror = () => {
      term.writeln(
        "\x1b[1;31m[ERROR] No se pudo conectar con el servidor. Verifica que estás autenticado.\x1b[0m",
      );
    };

    ws.onclose = (e) => {
      term.writeln(
        `\x1b[33m\r\n[DESCONECTADO] Sesión cerrada (código: ${e.code}).\x1b[0m`,
      );
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(event.data));
      } else {
        term.write(event.data as string);
      }
    };

    // ── Terminal → WebSocket (input del usuario) ──────────────────────────
    const dataDisposable = term.onData((data: string) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // ── Copiar al portapapeles al seleccionar texto ──────────────────────
    const selectionDisposable = term.onSelectionChange(() => {
      if (term.hasSelection()) {
        const text = term.getSelection();
        navigator.clipboard.writeText(text).catch(() => {});
      }
    });

    // ── Soporte de Pegado (Paste) mediante Ctrl+V / Cmd+V ─────────────────
    term.attachCustomKeyEventHandler((arg) => {
      if (arg.type === "keydown" && (arg.ctrlKey || arg.metaKey) && arg.key === "v") {
        navigator.clipboard.readText().then((text) => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(text);
          }
        }).catch(() => {});
        return false;
      }
      return true;
    });

    // ── Evento pegar del contenedor (por si usan click derecho -> pegar) ─
    const handlePasteEvent = (e: ClipboardEvent) => {
      e.preventDefault();
      const text = e.clipboardData?.getData("text");
      if (text && ws.readyState === WebSocket.OPEN) {
        ws.send(text);
      }
    };
    container.addEventListener("paste", handlePasteEvent);

    // ── Resize observer ───────────────────────────────────────────────────
    const resizeObserver = new ResizeObserver(() => {
      try {
        fitAddon.fit();
      } catch {
        // ignorar
      }
    });
    resizeObserver.observe(container);

    // ── Cleanup ───────────────────────────────────────────────────────────
    return () => {
      dataDisposable.dispose();
      selectionDisposable.dispose();
      resizeObserver.disconnect();
      container.removeEventListener("paste", handlePasteEvent);
      ws.close();
      term.dispose();
    };
  }, []);

  return (
    <div className="space-y-6 flex flex-col h-full">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Interactive Shell</h2>
          <p className="text-sm text-base-content/70">
            Secure admin-only root terminal session.
          </p>
        </div>
        <button
          onClick={() => {
            termRef.current?.clear();
            termRef.current?.focus();
          }}
          className="btn btn-sm btn-ghost gap-2 font-mono border border-base-300"
          title="Clear screen"
        >
          Limpiar
        </button>
      </div>

      <div className="card bg-base-100 border border-base-300 shadow-sm flex-1 min-h-0 flex flex-col overflow-hidden">
        <div
          style={{
            display: "flex",
            alignItems: "center",
            padding: "8px 14px",
            background: "#181b23",
            borderBottom: "1px solid #313244",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: "#a6e3a1",
                display: "inline-block",
                boxShadow: "0 0 6px #a6e3a1aa",
              }}
            />
            <span
              style={{
                fontFamily: "monospace",
                fontSize: 11,
                color: "#7f849c",
                letterSpacing: "0.08em",
                textTransform: "uppercase",
              }}
            >
              Shell · Admin Only · Timeout 30m · Autocopy on Select · Ctrl+V to paste
            </span>
          </div>
        </div>

        <div
          ref={containerRef}
          className="flex-1 min-h-0 p-2 bg-[#0d0f14]"
          style={{ overflow: "hidden" }}
        />
      </div>
    </div>
  );
};

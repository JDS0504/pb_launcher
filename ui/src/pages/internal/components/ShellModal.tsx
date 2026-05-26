import { useEffect, useRef, type FC } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { buildShellWsUrl } from "../../../services/shell";
import "@xterm/xterm/css/xterm.css";

type Props = {
  onClose?: () => void;
};

export const ShellModal: FC<Props> = ({ onClose }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTerm | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

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
    term.open(containerRef.current);
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
    const disposable = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // ── Resize observer ───────────────────────────────────────────────────
    const resizeObserver = new ResizeObserver(() => {
      try {
        fitAddon.fit();
      } catch {
        // ignorar errores de resize durante desmontaje
      }
    });
    resizeObserver.observe(containerRef.current);

    // ── Cleanup ───────────────────────────────────────────────────────────
    return () => {
      disposable.dispose();
      resizeObserver.disconnect();
      ws.close();
      term.dispose();
    };
  }, []);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        height: "75vh",
        minHeight: 480,
        background: "#0d0f14",
        borderRadius: "0 0 8px 8px",
        overflow: "hidden",
      }}
    >
      {/* Barra de estado superior */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "6px 14px",
          background: "#181b23",
          borderBottom: "1px solid #313244",
          flexShrink: 0,
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
            Shell Interactiva · Admin Only · Timeout 30 min
          </span>
        </div>
        <div style={{ display: "flex", gap: 6 }}>
          <button
            onClick={() => {
              termRef.current?.clear();
            }}
            style={{
              background: "transparent",
              border: "1px solid #313244",
              color: "#7f849c",
              borderRadius: 4,
              padding: "2px 10px",
              fontSize: 11,
              cursor: "pointer",
              fontFamily: "monospace",
            }}
            title="Limpiar terminal"
          >
            Limpiar
          </button>
          {onClose && (
            <button
              onClick={onClose}
              style={{
                background: "transparent",
                border: "1px solid #f38ba8",
                color: "#f38ba8",
                borderRadius: 4,
                padding: "2px 10px",
                fontSize: 11,
                cursor: "pointer",
                fontFamily: "monospace",
              }}
              title="Cerrar shell"
            >
              Cerrar
            </button>
          )}
        </div>
      </div>

      {/* Contenedor de xterm */}
      <div
        ref={containerRef}
        style={{
          flex: 1,
          minHeight: 0,
          padding: "8px 4px 4px 4px",
          background: "#0d0f14",
          overflow: "hidden",
        }}
      />
    </div>
  );
};

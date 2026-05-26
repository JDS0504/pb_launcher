import { useState, useRef, useEffect, type FC } from "react";
import { useModal } from "../../../components/modal/hook";
import { cliService } from "../../../services/cli";
import { Terminal, Send, Trash2, HelpCircle } from "lucide-react";

type Props = {
  serviceId: string;
  serviceName: string;
};

type LogLine = {
  type: "input" | "output" | "error" | "info";
  text: string;
  time: string;
};

export const CLIConsoleModal: FC<Props> = ({ serviceId, serviceName }) => {
  const { closeModal } = useModal();
  const [command, setCommand] = useState("");
  const [history, setHistory] = useState<LogLine[]>([
    {
      type: "info",
      text: `Conectado a la CLI de la instancia '${serviceName}' (PocketBase).`,
      time: new Date().toLocaleTimeString(),
    },
    {
      type: "info",
      text: "Escribe un comando de PocketBase (ej: 'superuser upsert email pass', 'migrate history', 'version') y presiona Enter.",
      time: new Date().toLocaleTimeString(),
    },
  ]);
  const [loading, setLoading] = useState(false);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [history]);

  const handleSendCommand = async (cmdText: string) => {
    const trimmed = cmdText.trim();
    if (!trimmed) return;

    const parts = trimmed.split(/\s+/);
    const newLogLine: LogLine = {
      type: "input",
      text: `./pocketbase ${trimmed}`,
      time: new Date().toLocaleTimeString(),
    };

    setHistory((prev) => [...prev, newLogLine]);
    setCommand("");
    setLoading(true);

    try {
      const response = await cliService.executeCliCommand(serviceId, parts);
      setHistory((prev) => [
        ...prev,
        {
          type: response.success ? "output" : "error",
          text: response.output || "(Sin salida de texto)",
          time: new Date().toLocaleTimeString(),
        },
      ]);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : "Error desconocido";
      setHistory((prev) => [
        ...prev,
        {
          type: "error",
          text: `Error de ejecución: ${errMsg}`,
          time: new Date().toLocaleTimeString(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (loading) return;
    handleSendCommand(command);
  };

  const handleClear = () => {
    setHistory([
      {
        type: "info",
        text: "Historial limpiado.",
        time: new Date().toLocaleTimeString(),
      },
    ]);
  };

  return (
    <div className="flex flex-col h-[70vh] min-h-[450px] w-full max-w-2xl bg-neutral text-neutral-content rounded-lg overflow-hidden border border-base-300">
      {/* Cabecera del terminal */}
      <div className="p-3 bg-neutral-focus border-b border-neutral-content/10 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-primary animate-pulse" />
          <span className="text-xs font-mono font-bold tracking-wider uppercase text-neutral-content/95">
            PocketBase CLI: {serviceName}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={handleClear}
            className="btn btn-xs btn-ghost gap-1 font-mono text-[9px]"
            title="Limpiar Consola"
          >
            <Trash2 className="w-3 h-3 text-error" />
            Limpiar
          </button>
          <div className="dropdown dropdown-end">
            <label tabIndex={0} className="btn btn-xs btn-ghost btn-circle">
              <HelpCircle className="w-3.5 h-3.5 opacity-60 hover:opacity-100" />
            </label>
            <ul
              tabIndex={0}
              className="dropdown-content menu p-3 shadow-lg bg-base-300 text-base-content rounded-box w-72 text-xs border border-base-200 z-50 space-y-1 font-mono"
            >
              <li className="font-bold border-b border-base-200 pb-1 mb-1">Comandos Útiles:</li>
              <li>• version</li>
              <li>• migrate history</li>
              <li>• migrate collections</li>
              <li>• superuser upsert email pass</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Visor de terminal */}
      <div className="flex-1 overflow-y-auto p-4 space-y-2.5 font-mono text-xs bg-black/40 min-h-0 select-text leading-relaxed">
        {history.map((h, i) => {
          if (h.type === "info") {
            return (
              <div key={i} className="text-info/85 border-l-2 border-info/40 pl-2 py-0.5">
                <span className="text-[10px] opacity-40 mr-2">[{h.time}]</span>
                {h.text}
              </div>
            );
          }
          if (h.type === "input") {
            return (
              <div key={i} className="text-primary font-bold">
                <span className="text-[10px] opacity-40 mr-2 font-normal text-neutral-content">[{h.time}]</span>
                <span className="text-neutral-content/50 mr-1.5">$</span>
                {h.text}
              </div>
            );
          }
          if (h.type === "error") {
            return (
              <div key={i} className="text-error bg-error/10 border-l-2 border-error/55 pl-2 py-1 whitespace-pre-wrap rounded">
                <span className="text-[10px] opacity-40 mr-2">[{h.time}]</span>
                {h.text}
              </div>
            );
          }
          return (
            <div key={i} className="text-success whitespace-pre-wrap pl-4 py-0.5">
              {h.text}
            </div>
          );
        })}
        {loading && (
          <div className="flex items-center gap-2 text-primary pl-4 py-1 italic animate-pulse">
            <span className="loading loading-spinner loading-xs" />
            Ejecutando comando en el VPS...
          </div>
        )}
        <div ref={terminalEndRef} />
      </div>

      {/* Input de comandos */}
      <form
        onSubmit={handleSubmit}
        className="p-3 bg-neutral-focus border-t border-neutral-content/10 flex gap-2 items-center shrink-0"
      >
        <span className="text-sm font-bold font-mono text-primary">$</span>
        <input
          type="text"
          className="input input-sm input-bordered flex-1 bg-black/30 border-neutral-content/20 text-neutral-content focus:outline-none focus:border-primary font-mono text-xs h-9"
          placeholder="Escribe un comando... (ej: version)"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          disabled={loading}
          autoFocus
        />
        <button
          type="submit"
          className="btn btn-sm btn-primary h-9 min-h-9 px-3 gap-1"
          disabled={loading || !command.trim()}
        >
          <Send className="w-3.5 h-3.5" />
          Ejecutar
        </button>
        <button
          type="button"
          className="btn btn-sm btn-ghost h-9 min-h-9 text-xs"
          onClick={closeModal}
        >
          Cerrar
        </button>
      </form>
    </div>
  );
};

import { type FC } from "react";
import {
  Globe,
  Shield,
  CheckCircle,
  AlertTriangle,
  Copy,
  ArrowRight,
} from "lucide-react";
import { useModal } from "../../../components/modal/hook";
import toast from "react-hot-toast";

type StepProps = {
  number: number;
  title: string;
  children: React.ReactNode;
};

const Step: FC<StepProps> = ({ number, title, children }) => (
  <div className="flex gap-3">
    <div className="flex-shrink-0 w-7 h-7 rounded-full bg-primary text-primary-content flex items-center justify-center text-xs font-bold">
      {number}
    </div>
    <div className="flex-1 pb-5">
      <p className="font-semibold text-sm text-base-content mb-1.5">{title}</p>
      <div className="text-sm text-base-content/70 space-y-2">{children}</div>
    </div>
  </div>
);

type CodeRowProps = {
  label: string;
  value: string;
  copyable?: boolean;
};

const CodeRow: FC<CodeRowProps> = ({ label, value, copyable }) => {
  const handleCopy = () => {
    navigator.clipboard.writeText(value).then(() => toast.success("Copiado"));
  };

  return (
    <div className="flex items-center justify-between gap-2 bg-base-200 rounded px-3 py-1.5 font-mono text-xs">
      <span className="text-base-content/50 shrink-0">{label}</span>
      <span className="text-base-content truncate">{value}</span>
      {copyable && (
        <button
          onClick={handleCopy}
          className="btn btn-ghost btn-xs p-0.5 shrink-0"
          title="Copiar"
        >
          <Copy className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  );
};

type BadgeProps = {
  color: "warning" | "error" | "success";
  children: React.ReactNode;
};

const Badge: FC<BadgeProps> = ({ color, children }) => {
  const cls = {
    warning: "bg-warning/15 text-warning border-warning/30",
    error: "bg-error/15 text-error border-error/30",
    success: "bg-success/15 text-success border-success/30",
  }[color];
  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded border text-xs font-medium ${cls}`}
    >
      {children}
    </span>
  );
};

type Props = {
  serverIp: string;
};

export const DnsSetupGuide: FC<Props> = ({ serverIp }) => {
  return (
    <div className="w-full max-w-lg space-y-1 pb-2">
      {/* Intro */}
      <div className="flex items-start gap-2 bg-info/10 border border-info/25 rounded-lg p-3 mb-4">
        <Globe className="w-4 h-4 text-info shrink-0 mt-0.5" />
        <p className="text-xs text-base-content/80 leading-relaxed">
          Para apuntar tu dominio a este servidor necesitas configurar los
          registros DNS en tu proveedor (Cloudflare, GoDaddy, etc.) y{" "}
          <strong>esperar a que el certificado SSL quede aprobado</strong> antes
          de activar el proxy de Cloudflare.
        </p>
      </div>

      {/* Steps */}
      <div className="divide-y divide-base-300">
        {/* Paso 1 */}
        <Step number={1} title="Crea el registro A en tu proveedor DNS">
          <p>Apunta el dominio raíz a la IP de este servidor:</p>
          <div className="space-y-1.5 mt-2">
            <CodeRow label="Tipo" value="A" />
            <CodeRow label="Nombre" value="@  (o tu subdominio)" />
            <CodeRow label="Valor (IP)" value={serverIp} copyable />
            <CodeRow label="TTL" value="Auto / 300s" />
          </div>
          <p className="mt-2 text-xs">
            Si quieres usar <code className="text-xs bg-base-200 px-1 rounded">www</code>, agrega también un registro{" "}
            <strong>CNAME</strong>:
          </p>
          <div className="space-y-1.5 mt-1.5">
            <CodeRow label="Tipo" value="CNAME" />
            <CodeRow label="Nombre" value="www" />
            <CodeRow label="Valor" value="tu-dominio.com" copyable />
          </div>
        </Step>

        {/* Paso 2 */}
        <Step number={2} title="Usa modo DNS-only en Cloudflare (¡obligatorio!)">
          <div className="flex items-start gap-2 bg-warning/10 border border-warning/25 rounded-lg p-2.5">
            <AlertTriangle className="w-4 h-4 text-warning shrink-0 mt-0.5" />
            <p className="text-xs leading-relaxed">
              Mientras el certificado SSL se está generando, el proxy de
              Cloudflare <strong>debe estar desactivado</strong>. Si el ícono de
              la nube está naranja (Proxied), Let's Encrypt no puede verificar
              el dominio y el certificado fallará.
            </p>
          </div>
          <div className="flex items-center gap-2 mt-2.5">
            <Badge color="error">🟠 Proxied — NO usar aún</Badge>
            <ArrowRight className="w-3 h-3 text-base-content/40 shrink-0" />
            <Badge color="success">⬜ DNS only — usar ahora</Badge>
          </div>
          <p className="mt-2 text-xs">
            En Cloudflare: <strong>DNS → Records → click en el ícono de
            nube</strong> para que quede gris (DNS only).
          </p>
        </Step>

        {/* Paso 3 */}
        <Step number={3} title='Registra el dominio aquí y haz clic en "Validate DNS"'>
          <p>
            Una vez propagados los DNS (~5 min), añade el dominio en esta
            sección. El sistema solicitará automáticamente el certificado SSL
            via Let's Encrypt. Cuando el estado cambie a{" "}
            <strong className="text-success">approved</strong>, el certificado
            está listo.
          </p>
        </Step>

        {/* Paso 4 */}
        <Step number={4} title="Activa el proxy de Cloudflare (DDoS, caché, etc.)">
          <div className="flex items-start gap-2 bg-success/10 border border-success/25 rounded-lg p-2.5">
            <CheckCircle className="w-4 h-4 text-success shrink-0 mt-0.5" />
            <p className="text-xs leading-relaxed">
              Con el certificado <strong>aprobado</strong>, ya puedes volver a
              activar el proxy naranja en Cloudflare. A partir de ahí,
              Cloudflare protege tu dominio contra DDoS, cachea recursos y
              gestiona la seguridad TLS por ti.
            </p>
          </div>
          <div className="flex items-center gap-2 mt-2.5">
            <Badge color="success">⬜ DNS only — antes</Badge>
            <ArrowRight className="w-3 h-3 text-base-content/40 shrink-0" />
            <Badge color="warning">🟠 Proxied — activar ahora</Badge>
          </div>
          <div className="flex items-start gap-2 bg-base-200 rounded-lg p-2.5 mt-2">
            <Shield className="w-4 h-4 text-primary shrink-0 mt-0.5" />
            <p className="text-xs leading-relaxed">
              Configura el modo SSL/TLS de Cloudflare en{" "}
              <strong>Full (strict)</strong> para que Cloudflare verifique el
              certificado real del servidor.
            </p>
          </div>
        </Step>
      </div>
    </div>
  );
};

type ButtonProps = {
  serverIp?: string;
};

export const DnsSetupGuideButton: FC<ButtonProps> = ({
  serverIp = "YOUR_SERVER_IP",
}) => {
  const { openModal } = useModal();

  const handleOpen = () => {
    openModal(<DnsSetupGuide serverIp={serverIp} />, {
      title: "¿Cómo implementar un dominio personalizado?",
      width: 520,
    });
  };

  return (
    <button
      onClick={handleOpen}
      className="btn btn-sm btn-ghost gap-1.5 text-base-content/70 hover:text-base-content"
    >
      <Globe className="w-4 h-4" />
      ¿Cómo implementar?
    </button>
  );
};

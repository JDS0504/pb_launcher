import { useState, type FC } from "react";
import { useModal } from "../../../components/modal/hook";
import { serviceService } from "../../../services/services";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../../utils/errors";
import { Eye, EyeOff, Copy, Check } from "lucide-react";

type Props = {
  service_id: string;
};

export const ChangePasswordModal: FC<Props> = ({ service_id }) => {
  const { closeModal } = useModal();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [showPwd, setShowPwd] = useState(false);
  const [loading, setLoading] = useState(false);
  const [savedEmail, setSavedEmail] = useState<string | null>(null);
  const [savedPwd, setSavedPwd] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const error =
    confirm && password !== confirm ? "Las contraseñas no coinciden" : null;
  const isValid = password.length >= 8 && !error;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isValid) return;
    setLoading(true);
    try {
      const result = await serviceService.upsertSuperuser({
        service_id,
        password,
      });
      setSavedEmail(result.email);
      setSavedPwd(result.password);
      toast.success("Contraseña actualizada correctamente");
    } catch (err) {
      toast.error(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = async () => {
    if (!savedEmail || !savedPwd) return;
    await navigator.clipboard.writeText(
      `Email: ${savedEmail}\nPassword: ${savedPwd}`,
    );
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (savedEmail && savedPwd) {
    return (
      <div className="space-y-4 p-2 w-full max-w-sm">
        <div className="alert alert-success text-sm">
          Contraseña actualizada. Guarda estas credenciales:
        </div>
        <div className="bg-base-200 rounded-lg p-4 space-y-2 font-mono text-sm">
          <div>
            <span className="text-base-content/50 text-xs">EMAIL</span>
            <p className="font-medium">{savedEmail}</p>
          </div>
          <div>
            <span className="text-base-content/50 text-xs">PASSWORD</span>
            <p className="font-medium">{savedPwd}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <button className="btn btn-sm btn-ghost flex-1 gap-2" onClick={handleCopy}>
            {copied ? <Check className="w-4 h-4 text-success" /> : <Copy className="w-4 h-4" />}
            {copied ? "Copiado" : "Copiar credenciales"}
          </button>
          <button className="btn btn-sm btn-primary flex-1" onClick={closeModal}>
            Cerrar
          </button>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 p-2 w-full max-w-sm">
      <div className="text-sm text-base-content/70">
        Actualiza la contraseña del superusuario de PocketBase para esta instancia.
        La contraseña debe tener al menos 8 caracteres.
      </div>

      <div className="form-control w-full">
        <label className="label">
          <span className="label-text text-sm font-medium">Nueva contraseña</span>
        </label>
        <div className="relative">
          <input
            id="change-pwd-new"
            type={showPwd ? "text" : "password"}
            className="input input-bordered w-full pr-10"
            value={password}
            onChange={e => setPassword(e.target.value)}
            autoComplete="new-password"
            minLength={8}
            required
          />
          <button
            type="button"
            className="absolute right-3 top-1/2 -translate-y-1/2 text-base-content/50 hover:text-base-content"
            onClick={() => setShowPwd(v => !v)}
          >
            {showPwd ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </button>
        </div>
        {password && password.length < 8 && (
          <label className="label">
            <span className="label-text-alt text-error">Mínimo 8 caracteres</span>
          </label>
        )}
      </div>

      <div className="form-control w-full">
        <label className="label">
          <span className="label-text text-sm font-medium">Confirmar contraseña</span>
        </label>
        <input
          id="change-pwd-confirm"
          type={showPwd ? "text" : "password"}
          className={`input input-bordered w-full ${error ? "input-error" : ""}`}
          value={confirm}
          onChange={e => setConfirm(e.target.value)}
          autoComplete="new-password"
          required
        />
        {error && (
          <label className="label">
            <span className="label-text-alt text-error">{error}</span>
          </label>
        )}
      </div>

      <div className="flex gap-2 pt-2">
        <button
          type="button"
          className="btn btn-sm btn-ghost flex-1"
          onClick={closeModal}
          disabled={loading}
        >
          Cancelar
        </button>
        <button
          id="change-pwd-submit"
          type="submit"
          className="btn btn-sm btn-primary flex-1"
          disabled={!isValid || loading}
        >
          {loading && <span className="loading loading-spinner loading-xs" />}
          Guardar
        </button>
      </div>
    </form>
  );
};

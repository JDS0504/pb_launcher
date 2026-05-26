import { useCopyToClipboard } from "@uidotdev/usehooks";
import { Check, Copy } from "lucide-react";
import type { FC } from "react";
import { useState } from "react";

type Props = {
  service_id: string;
  username: string;
  password: string;
  onResetCredentials?: () => void;
};

export const DefaultCredentialsCard: FC<Props> = ({
  username: username_init,
  password: password_init,
}) => {
  const [{ password, username }] = useState<{
    username: string;
    password: string;
  }>({
    username: username_init,
    password: password_init,
  });
  const [, copyToClipboard] = useCopyToClipboard();
  const [copiedField, setCopiedField] = useState<
    "username" | "password" | null
  >(null);

  const handleCopy = (value: string, field: "username" | "password") => {
    copyToClipboard(value);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 1200);
  };

  return (
    <div className="card w-[350px] max-w-sm bg-base-100 shadow-xl border border-base-300">
      <div className="card-body space-y-4">
        <h2 className="card-title">Default Credentials</h2>

        <p className="text-sm text-warning">
          These credentials were generated automatically. You must change them
          after accessing the platform.
        </p>
        {username && password && (
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-4">
              <div>
                <span className="font-semibold text-xs">Username:</span>
                <div className="truncate text-sm">{username}</div>
              </div>
              <button
                className="btn btn-ghost btn-xs btn-circle"
                onClick={() => handleCopy(username, "username")}
              >
                {copiedField === "username" ? (
                  <Check size={14} className="text-success" />
                ) : (
                  <Copy size={14} />
                )}
              </button>
            </div>

            <div className="flex items-center justify-between gap-4">
              <div>
                <span className="font-semibold text-xs">Password:</span>
                <div className="truncate text-sm">{password}</div>
              </div>
              <button
                className="btn btn-ghost btn-xs btn-circle"
                onClick={() => handleCopy(password, "password")}
              >
                {copiedField === "password" ? (
                  <Check size={14} className="text-success" />
                ) : (
                  <Copy size={14} />
                )}
              </button>
            </div>
          </div>
        )}

        <div className="border-t border-base-300 pt-3">
          <p className="text-xs text-base-content/60">
            Si deseas cambiar las credenciales de esta instancia, ve a la sección <strong>Service &gt; General</strong> dentro de los detalles de la instancia.
          </p>
        </div>
      </div>
    </div>
  );
};

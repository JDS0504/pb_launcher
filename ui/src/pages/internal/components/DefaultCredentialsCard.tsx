import { useMutation } from "@tanstack/react-query";
import { useCopyToClipboard } from "@uidotdev/usehooks";
import classNames from "classnames";
import { Check, Copy } from "lucide-react";
import type { FC } from "react";
import { useState } from "react";
import { serviceService } from "../../../services/services";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../../utils/errors";
import { useModal } from "../../../components/modal/hook";
import { ConfirmAdminPasswordForm } from "../forms/ConfirmAdminPasswordForm";

type Props = {
  service_id: string;
  username: string;
  password: string;
  onResetCredentials?: () => void;
};

export const DefaultCredentialsCard: FC<Props> = ({
  service_id,
  username: username_init,
  password: password_init,
  onResetCredentials,
}) => {
  const [{ password, username }, setCredentials] = useState<{
    username: string;
    password: string;
  }>({
    username: username_init,
    password: password_init,
  });
  const { openModal } = useModal();
  const [, copyToClipboard] = useCopyToClipboard();
  const [copiedField, setCopiedField] = useState<
    "username" | "password" | null
  >(null);

  const handleCopy = (value: string, field: "username" | "password") => {
    copyToClipboard(value);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 1200);
  };
  const upsertSuperuserMutation = useMutation({
    mutationFn: serviceService.upsertSuperuser,
    onSuccess: ({ email, password }) => {
      setCredentials({ password, username: email });
      onResetCredentials?.();
    },
    onError: error => toast.error(getErrorMessage(error)),
  });

  const onUpsertSuperuserHandle = () => {
    openModal(
      <ConfirmAdminPasswordForm
        description="Confirma la contraseña de tu cuenta principal de PBLauncher para clonarla en esta instancia."
        label="Contraseña de Administrador"
        submitLabel="Confirmar y Clonar"
        onSubmit={async (adminPassword) => {
          await upsertSuperuserMutation.mutateAsync({
            service_id,
            password: adminPassword,
          });
        }}
      />,
      { title: "Clonar Contraseña Principal" }
    );
  };

  return (
    <div className="card w-[350px] max-w-sm bg-base-100 shadow-xl">
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
                <span className="font-semibold">Username:</span>
                <div className="truncate">{username}</div>
              </div>
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => handleCopy(username, "username")}
              >
                {copiedField === "username" ? (
                  <Check size={18} />
                ) : (
                  <Copy size={18} />
                )}
              </button>
            </div>

            <div className="flex items-center justify-between gap-4">
              <div>
                <span className="font-semibold">Password:</span>
                <div className="truncate">{password}</div>
              </div>
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => handleCopy(password, "password")}
              >
                {copiedField === "password" ? (
                  <Check size={18} />
                ) : (
                  <Copy size={18} />
                )}
              </button>
            </div>
          </div>
        )}

        <div
          className={classNames("card-actions", {
            "justify-end": username && password,
          })}
        >
          <button
            className={classNames("btn btn-outline btn-sm", {
              "btn-error": username && password,
              "btn-success w-full": !(username && password),
            })}
            onClick={onUpsertSuperuserHandle}
            disabled={upsertSuperuserMutation.isPending}
          >
            Upsert Superuser
          </button>
        </div>
      </div>
    </div>
  );
};

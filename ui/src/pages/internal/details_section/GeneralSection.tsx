import { useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import classNames from "classnames";
import { ServiceForm } from "../forms/ServiceForm";
import type { ServiceDto } from "../../../services/services";
import { useModal } from "../../../components/modal/hook";
import { ChangePasswordModal } from "../components/ChangePasswordModal";
import { KeyRound } from "lucide-react";

type Props = {
  service_id: string;
  service?: ServiceDto;
};

export const GeneralSection: FC<Props> = ({ service_id, service }) => {
  const queryClient = useQueryClient();
  const { openModal } = useModal();

  if (service == null) {
    return <div className="p-4">Loading...</div>;
  }

  const status =
    service.status.charAt(0).toUpperCase() + service.status.slice(1);

  const openChangePassword = () => {
    openModal(
      <ChangePasswordModal service_id={service_id} />,
      { title: "Cambiar contraseña del superuser" },
    );
  };

  return (
    <div className="relative pt-4 space-y-4">
      {/* Badge de estado */}
      <p
        className={classNames("badge badge-sm absolute -top-2 right-2", {
          "badge-success": service.status === "running",
          "badge-info": service.status === "sleeping",
          "badge-warning":
            service.status === "pending" || service.status === "idle",
          "badge-error": service.status === "failure",
          "badge-neutral": !["running", "pending", "idle", "failure", "sleeping"].includes(
            service.status,
          ),
        })}
      >
        {status}
      </p>

      {/* Badge de versión */}
      {service.release_version && (
        <div className="flex items-center gap-2">
          <span className="badge badge-ghost badge-sm font-mono">
            v{service.release_version}
          </span>
          <span className="text-xs text-base-content/50">{service.repository}</span>
        </div>
      )}

      <ServiceForm
        record={service}
        onSaveRecord={() => {
          queryClient.invalidateQueries({ queryKey: ["services", service_id] });
          queryClient.invalidateQueries({ queryKey: ["services"] });
        }}
      />

      {/* Acciones adicionales */}
      <div className="pt-2 border-t border-base-300">
        <button
          id="btn-change-password"
          className="btn btn-sm btn-outline gap-2"
          onClick={openChangePassword}
          title="Cambiar la contraseña del superusuario de esta instancia PocketBase"
        >
          <KeyRound className="w-4 h-4" />
          Cambiar contraseña del superuser
        </button>
      </div>
    </div>
  );
};

import { useMutation, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { useModal } from "../components/modal/hook";
import { useConfirmModal } from "./useConfirmModal";
import { getErrorMessage } from "../utils/errors";
import { serviceService, type ServiceDto } from "../services/services";
import { ChangeVersionForm } from "../pages/internal/forms/ChangeVersionForm";
import { CloneServiceForm } from "../pages/internal/forms/CloneServiceForm";

/**
 * Hook SSOT con todas las acciones sobre un servicio.
 * Úsalo en ServicesPage (para las cards) y en GeneralSection (para el detalle).
 */
export const useServiceActions = (onMutationSuccess?: () => void) => {
  const queryClient = useQueryClient();
  const { openModal } = useModal();
  const confirm = useConfirmModal();

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["services"] });
    onMutationSuccess?.();
  };

  // ── Comando genérico (start / stop / restart / upgrade) ──────────────────
  const commandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => setTimeout(invalidate),
    onError: error => toast.error(getErrorMessage(error)),
  });

  // ── Delete ────────────────────────────────────────────────────────────────
  const deleteMutation = useMutation({
    mutationFn: serviceService.deleteServiceInstance,
    onSuccess: () => setTimeout(invalidate),
    onError: error => toast.error(getErrorMessage(error)),
  });


  // ── Handlers ──────────────────────────────────────────────────────────────
  const handleStart = (service_id: string) => {
    commandMutation.mutate({ service_id, action: "start" });
  };

  const handleStop = async (service_id: string) => {
    const ok = await confirm(
      "Detener servicio",
      "¿Seguro que quieres detener este servicio?",
    );
    if (ok) commandMutation.mutate({ service_id, action: "stop" });
  };

  const handleRestart = async (service_id: string) => {
    const ok = await confirm(
      "Reiniciar servicio",
      "¿Seguro que quieres reiniciar este servicio?",
    );
    if (ok) commandMutation.mutate({ service_id, action: "restart" });
  };

  const handleDelete = async (service_id: string) => {
    const ok = await confirm(
      "Eliminar servicio",
      "¿Estás seguro? Esta acción es irreversible y eliminará permanentemente la carpeta de datos y base de datos en el servidor.",
    );
    if (ok) deleteMutation.mutate(service_id);
  };


  const handleChangeVersion = (service: ServiceDto) => {
    openModal(
      <ChangeVersionForm service={service} onSuccess={() => setTimeout(invalidate)} />,
      { title: "Cambiar versión", width: 440 },
    );
  };

  const handleClone = (service: ServiceDto) => {
    openModal(
      <CloneServiceForm service={service} onClone={() => setTimeout(invalidate)} />,
      { title: "Clonar servicio", width: 420 },
    );
  };

  return {
    handleStart,
    handleStop,
    handleRestart,
    handleDelete,
    handleChangeVersion,
    handleClone,
    isCommandPending: commandMutation.isPending,
    isDeletePending: deleteMutation.isPending,
  };
};

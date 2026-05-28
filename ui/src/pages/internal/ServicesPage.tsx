import toast from "react-hot-toast";
import { useMemo, useState } from "react";
import { Plus, Upload } from "lucide-react";
import { useModal } from "../../components/modal/hook";
import { ServiceForm } from "./forms/ServiceForm";
import { UpgradeServiceForm } from "./forms/UpgradeServiceForm";
import { RestoreBackupForm } from "./forms/RestoreBackupForm";
import { CloneServiceForm } from "./forms/CloneServiceForm";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useLocalStorage } from "@uidotdev/usehooks";
import { serviceService, type ServiceDto } from "../../services/services";
import { ServiceCard } from "./components/ServiceCard";
import { useConfirmModal } from "../../hooks/useConfirmModal";
import { getErrorMessage } from "../../utils/errors";
import { useNavigate } from "react-router-dom";
import { useProxyConfigs } from "../../hooks/useProxyConfigs";
import { backupService } from "../../services/backup";

const STATUS_FILTER_KEY = "pb-dashboard-status-filter";
type TStatus = "all" | "running" | "stopped";
export const ServicesPage = () => {
  const navigate = useNavigate();
  const { openModal } = useModal();
  const confirm = useConfirmModal();

  const proxyInfo = useProxyConfigs();
  const servicesQuery = useQuery({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
    refetchInterval: 3000,
  });

  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useLocalStorage<{ value: TStatus }>(
    STATUS_FILTER_KEY,
    { value: "all" },
  );

  const filtered = useMemo(() => {
    return (servicesQuery.data ?? [])
      .filter(
        s =>
          String(s.id).includes(query.toLowerCase()) ||
          s.name.toLowerCase().includes(query.toLowerCase()),
      )
      .filter(s => {
        switch (statusFilter.value) {
          case "all":
            return true;
          case "running":
            return s.status === "running";
          case "stopped":
            return s.status === "stopped";
        }
      });
  }, [servicesQuery.data, query, statusFilter]);

  const deleteMutation = useMutation({
    mutationFn: serviceService.deleteServiceInstance,
    onSuccess: () => setTimeout(() => servicesQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const backupMutation = useMutation({
    mutationFn: backupService.downloadBackup,
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleDeleteService = async (id: string) => {
    const ok = await confirm(
      "Eliminar servicio",
      "¿Estás seguro de que deseas eliminar este servicio? Esta acción es irreversible y eliminará permanentemente su carpeta de datos y base de datos en el servidor VPS.",
    );
    if (ok) {
      deleteMutation.mutate(id);
    }
  };

  const serviceCommandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => setTimeout(() => servicesQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleStartService = async (id: string) => {
    serviceCommandMutation.mutate({ service_id: id, action: "start" });
  };

  const handleStopService = async (id: string) => {
    const ok = await confirm(
      "Stop service",
      "Are you sure you want to stop this service?",
    );
    if (ok) {
      serviceCommandMutation.mutate({ service_id: id, action: "stop" });
    }
  };

  const handleRestartService = async (id: string) => {
    const ok = await confirm(
      "Restart service",
      "Are you sure you want to restart this service?",
    );
    if (ok) {
      serviceCommandMutation.mutate({ service_id: id, action: "restart" });
    }
  };

  const handleUpgradeService = (service: ServiceDto) => {
    openModal(
      <UpgradeServiceForm
        service={service}
        onUpgrade={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Upgrade Service", width: 420 },
    );
  };

  const handleBackupService = async (service: ServiceDto) => {
    backupMutation.mutate(service.id);
  };

  const handleCloneService = (service: ServiceDto) => {
    openModal(
      <CloneServiceForm
        service={service}
        onClone={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Clone Service", width: 420 },
    );
  };

  const openRestoreBackupModal = () => {
    openModal(
      <RestoreBackupForm
        onRestore={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Import Backup", width: 420 },
    );
  };

  const openCreateServiceModal = () => {
    openModal(
      <ServiceForm
        onSaveRecord={() => setTimeout(() => servicesQuery.refetch())}
        width={360}
      />,
      {
        title: "Create Service",
      },
    );
  };

  const openDetailsService = (service: ServiceDto) =>
    navigate(`/services/${service.id}`);

  return (
    <div className="space-y-6">
      {/* Cabecera (Sólo título sin iconos en móvil, oculto en PC) */}
      <h2 className="text-xl font-bold md:hidden block">Services</h2>

      <div className="flex flex-col md:flex-row md:items-center gap-3.5 w-full">
        {/* Bloque 1: Búsqueda y Filtro de Estado */}
        <div className="flex flex-row items-center gap-2 flex-grow w-full md:w-auto">
          <input
            type="text"
            placeholder="Search service..."
            className="input input-sm input-bordered w-full md:max-w-xs"
            value={query}
            onChange={e => setQuery(e.target.value)}
          />
          <select
            className="select select-sm select-bordered w-32 sm:w-48 shrink-0"
            value={statusFilter.value}
            onChange={e =>
              setStatusFilter({ value: e.target.value as TStatus })
            }
          >
            <option value="all">All</option>
            <option value="running">Running</option>
            <option value="stopped">Stopped</option>
          </select>
        </div>

        {/* Bloque 2: Botones de Acción */}
        <div className="flex flex-row gap-2 w-full md:w-auto shrink-0 md:justify-end">
          <button
            className="btn btn-sm btn-secondary gap-2 flex-1 md:flex-none"
            onClick={openRestoreBackupModal}
          >
            <Upload className="w-4 h-4" />
            Import backup
          </button>
          <button
            className="btn btn-sm btn-primary gap-2 flex-1 md:flex-none"
            onClick={openCreateServiceModal}
          >
            <Plus className="w-4 h-4" />
            New instance
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filtered.map(service => (
          <ServiceCard
            proxyInfo={proxyInfo}
            key={service.id}
            service={service}
            refreshData={() => setTimeout(() => servicesQuery.refetch())}
            onDetails={() => openDetailsService(service)}
            onDelete={() => handleDeleteService(service.id)}
            onStart={() => handleStartService(service.id)}
            onStop={() => handleStopService(service.id)}
            onRestart={() => handleRestartService(service.id)}
            onUpgrade={() => handleUpgradeService(service)}
            onBackup={() => handleBackupService(service)}
            onClone={() => handleCloneService(service)}
          />
        ))}
      </div>
    </div>
  );
};

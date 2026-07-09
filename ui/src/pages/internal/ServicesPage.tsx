import toast from "react-hot-toast";
import { useMemo, useState } from "react";
import { Plus, Upload } from "lucide-react";
import { useModal } from "../../components/modal/hook";
import { ServiceForm } from "./forms/ServiceForm";
import { RestoreBackupForm } from "./forms/RestoreBackupForm";
import { useQuery } from "@tanstack/react-query";
import { useLocalStorage } from "@uidotdev/usehooks";
import { serviceService } from "../../services/services";
import { ServiceCard } from "./components/ServiceCard";
import { useNavigate } from "react-router-dom";
import { useProxyConfigs } from "../../hooks/useProxyConfigs";
import { useServiceActions } from "../../hooks/useServiceActions";

const STATUS_FILTER_KEY = "pb-dashboard-status-filter";
type TStatus = "all" | "running" | "stopped";

export const ServicesPage = () => {
  const navigate = useNavigate();
  const { openModal } = useModal();

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

  const { handleStart, handleStop, handleClone } = useServiceActions(() =>
    setTimeout(() => servicesQuery.refetch()),
  );

  const openImportSnapshotModal = () => {
    openModal(
      <RestoreBackupForm
        onRestore={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Importar Snapshot", width: 420 },
    );
  };

  const openCreateServiceModal = () => {
    openModal(
      <ServiceForm
        onSaveRecord={() => {
          setTimeout(() => servicesQuery.refetch());
          toast.success("Instancia creada correctamente");
        }}
        width={360}
      />,
      { title: "Create Service" },
    );
  };

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold md:hidden block">Services</h2>

      <div className="flex flex-col md:flex-row md:items-center gap-3.5 w-full">
        {/* Búsqueda y filtro */}
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

        {/* Botones globales */}
        <div className="flex flex-row gap-2 w-full md:w-auto shrink-0 md:justify-end">
          <button
            className="btn btn-sm btn-secondary gap-2 flex-1 md:flex-none"
            onClick={openImportSnapshotModal}
          >
            <Upload className="w-4 h-4" />
            Importar snapshot
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

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
        {filtered.map(service => (
          <ServiceCard
            proxyInfo={proxyInfo}
            key={service.id}
            service={service}
            onDetails={() => navigate(`/services/${service.name}`)}
            onStart={() => handleStart(service.id)}
            onStop={() => handleStop(service.id)}
            onClone={() => handleClone(service)}
          />
        ))}
      </div>
    </div>
  );
};

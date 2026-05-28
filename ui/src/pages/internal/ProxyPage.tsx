import { useMutation, useQuery } from "@tanstack/react-query";
import { proxyEntryService, type ProxyEntryDto } from "../../services/proxy";
import { Plus, RefreshCcw } from "lucide-react";
import classNames from "classnames";
import { useMemo, useState } from "react";
import { useProxyConfigs } from "../../hooks/useProxyConfigs";
import { ProxyCard } from "./components/ProxyCard";
import { useModal } from "../../components/modal/hook";
import { ProxyEntryForm } from "./forms/ProxyEntryForm";
import { useNavigate } from "react-router-dom";
import toast from "react-hot-toast";
import { getErrorMessage } from "../../utils/errors";
import { useConfirmModal } from "../../hooks/useConfirmModal";

export const ProxyPage = () => {
  const navigate = useNavigate();
  const { openModal } = useModal();
  const confirm = useConfirmModal();

  const proxyInfo = useProxyConfigs();
  const proxyQuery = useQuery({
    queryKey: ["proxy_entires"],
    queryFn: proxyEntryService.fetchAll,
    refetchInterval: 5000,
  });
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    return (proxyQuery.data ?? []).filter(
      s =>
        String(s.id).includes(query.toLowerCase()) ||
        s.name.toLowerCase().includes(query.toLowerCase()),
    );
  }, [proxyQuery.data, query]);

  const openCreateModal = () => {
    openModal(
      <ProxyEntryForm
        onSaveRecord={() => setTimeout(() => proxyQuery.refetch())}
        width={360}
      />,
      {
        title: "Create Proxy Entry",
      },
    );
  };

  const deleteMutation = useMutation({
    mutationFn: proxyEntryService.delete,
    onSuccess: () => setTimeout(() => proxyQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleDeleteProxyEntry = async (id: string) => {
    const ok = await confirm(
      "Delete Proxy Endpoint",
      "Are you sure you want to delete this proxy entry?",
    );
    if (ok) {
      deleteMutation.mutate(id);
    }
  };

  const openDetailsProxyEntry = (entry: ProxyEntryDto) =>
    navigate(`/proxy/${entry.id}`);

  return (
    <div className="space-y-6">
      {/* Cabecera (Sólo título sin iconos en móvil, oculto en PC) */}
      <h2 className="text-xl font-bold md:hidden block">Proxy</h2>

      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex gap-2 w-full sm:max-w-md">
          <input
            type="text"
            placeholder="Search entries..."
            className="input input-sm input-bordered w-full"
            value={query}
            onChange={e => setQuery(e.target.value)}
          />
          <button
            onClick={() => proxyQuery.refetch()}
            className="btn btn-sm btn-ghost"
          >
            <RefreshCcw
              className={classNames("w-4 h-4", {
                "animate-spin": proxyQuery.isFetching,
              })}
            />
          </button>
        </div>

        <div className="flex flex-col md:flex-row gap-4 select-none">
          <button
            className="btn btn-sm btn-primary gap-2 w-full sm:w-auto"
            onClick={openCreateModal}
          >
            <Plus className="w-4 h-4" />
            New Entry
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filtered.map(entry => (
          <ProxyCard
            proxyInfo={proxyInfo}
            key={entry.id}
            entry={entry}
            onDetails={() => openDetailsProxyEntry(entry)}
            onDelete={() => handleDeleteProxyEntry(entry.id)}
          />
        ))}
      </div>
    </div>
  );
};

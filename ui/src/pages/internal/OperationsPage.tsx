import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { Link } from "react-router-dom";
import { useState, useMemo } from "react";
import { Search } from "lucide-react";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import { serviceService } from "../../services/services";

const PAGE_SIZE = 25;

export const OperationsPage = () => {
  const [selectedServiceId, setSelectedServiceId] = useState<string>("all");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "success" | "error">("all");
  const [page, setPage] = useState(0);

  const operationsQuery = useQuery({
    queryKey: ["operation-logs", "global"],
    queryFn: () => serviceService.fetchAllOperationLogs(),
    refetchInterval: 3000,
  });

  const servicesQuery = useQuery({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
  });

  const filteredOperations = useMemo(() => {
    const ops = operationsQuery.data ?? [];
    return ops.filter(op => {
      const matchService = selectedServiceId === "all" || op.service === selectedServiceId;
      const matchSearch =
        search === "" ||
        op.operation.toLowerCase().includes(search.toLowerCase()) ||
        (op.message ?? "").toLowerCase().includes(search.toLowerCase()) ||
        (op.expand?.service?.name ?? "").toLowerCase().includes(search.toLowerCase());
      const matchStatus = statusFilter === "all" || op.status === statusFilter;
      return matchService && matchSearch && matchStatus;
    });
  }, [operationsQuery.data, selectedServiceId, search, statusFilter]);

  const totalPages = Math.ceil(filteredOperations.length / PAGE_SIZE);
  const paginated = filteredOperations.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  if (operationsQuery.isLoading || servicesQuery.isLoading) {
    return <div className="p-4">Loading...</div>;
  }

  if (operationsQuery.isError) {
    return (
      <ErrorFallback
        error={operationsQuery.error}
        onRetry={() => setTimeout(operationsQuery.refetch)}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Cabecera (Sólo título sin iconos en móvil, oculto en PC) */}
      <h2 className="text-xl font-bold md:hidden block">History</h2>

      <div className="flex flex-col sm:flex-row gap-2 w-full items-stretch sm:items-center justify-start">
        <div className="relative flex-grow sm:flex-none">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40" />
            <input
              id="history-global-search"
              type="search"
              className="input input-bordered input-sm w-full sm:w-48 pl-9"
              placeholder="Buscar por op, msg, serv..."
              value={search}
              onChange={e => { setSearch(e.target.value); setPage(0); }}
            />
          </div>
          <select
            className="select select-sm select-bordered w-full sm:w-40 shrink-0"
            value={selectedServiceId}
            onChange={e => { setSelectedServiceId(e.target.value); setPage(0); }}
          >
            <option value="all">All Services</option>
            {servicesQuery.data?.map(service => (
              <option key={service.id} value={service.id}>
                {service.name}
              </option>
            ))}
          </select>
          <div className="flex gap-1 shrink-0 justify-center sm:justify-start">
            {(["all", "success", "error"] as const).map(s => (
              <button
                key={s}
                className={classNames("btn btn-xs capitalize", {
                  "btn-neutral": statusFilter === s,
                  "btn-ghost": statusFilter !== s,
                })}
                onClick={() => { setStatusFilter(s); setPage(0); }}
              >
                {s === "all" ? "Todos" : s}
              </button>
            ))}
          </div>
        </div>

      {/* Contador */}
      {(search || statusFilter !== "all" || selectedServiceId !== "all") && (
        <p className="text-xs text-base-content/50">
          {filteredOperations.length} resultado{filteredOperations.length !== 1 ? "s" : ""} de {operationsQuery.data?.length ?? 0}
        </p>
      )}

      <div className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body">
          {filteredOperations.length === 0 ? (
            <div className="text-sm text-base-content/70 py-4 text-center">No operations found.</div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="table table-sm">
                  <thead>
                    <tr>
                      <th>Time</th>
                      <th>Service</th>
                      <th>Operation</th>
                      <th>Status</th>
                      <th>Message</th>
                    </tr>
                  </thead>
                  <tbody>
                    {paginated.map(operation => (
                      <tr key={operation.id}>
                        <td className="whitespace-nowrap">
                          {new Date(operation.created).toLocaleString()}
                        </td>
                        <td className="whitespace-nowrap">
                          {operation.expand?.service ? (
                            <Link
                              to={`/services/${operation.expand.service.id}?section=history`}
                              className="link link-primary"
                            >
                              {operation.expand.service.name}
                            </Link>
                          ) : (
                            "-"
                          )}
                        </td>
                        <td className="capitalize">{operation.operation}</td>
                        <td>
                          <span
                             className={classNames("badge badge-sm", {
                              "badge-success": operation.status === "success",
                              "badge-error": operation.status === "error",
                            })}
                          >
                            {operation.status}
                          </span>
                        </td>
                        <td className="max-w-md whitespace-pre-wrap">
                          {operation.message || "-"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Paginación */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between gap-2 pt-4 border-t border-base-200 mt-2">
                  <span className="text-xs text-base-content/50">
                    Página {page + 1} de {totalPages}
                  </span>
                  <div className="flex gap-1">
                    <button
                      className="btn btn-xs btn-ghost"
                      disabled={page === 0}
                      onClick={() => setPage(p => p - 1)}
                    >
                      ← Anterior
                    </button>
                    <button
                      className="btn btn-xs btn-ghost"
                      disabled={page >= totalPages - 1}
                      onClick={() => setPage(p => p + 1)}
                    >
                      Siguiente →
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

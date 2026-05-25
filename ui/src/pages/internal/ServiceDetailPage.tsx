import { Navigate, useParams, useSearchParams } from "react-router-dom";
import { useState } from "react";
import { MenuIcon, XIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { serviceService } from "../../services/services";
import { GeneralSection } from "./details_section/GeneralSection";
import { DomainsSection } from "./details_section/DomainsSection";
import { ServiceLogsSection } from "./details_section/ServiceLogsSection";
import { OperationHistorySection } from "./details_section/OperationHistorySection";
import { FileManagerSection } from "./details_section/FileManagerSection";
import { SnapshotsSection } from "./details_section/SnapshotsSection";

export const ServiceDetailPage = () => {
  const { service_id } = useParams<{ service_id: string }>();

  const [searchParams, setSearchParams] = useSearchParams();
  const activeSection = searchParams.get("section") || "general";
  const [menuOpen, setMenuOpen] = useState(false);

  const serviceQuery = useQuery({
    queryKey: ["services", service_id],
    queryFn: () => serviceService.fetchServiceByID(service_id || ""),
    enabled: service_id != null && service_id !== "",
  });

  const handleSectionChange = (section: string) => {
    setSearchParams({ section });
    setMenuOpen(false);
  };

  const menuItemClass = (section: string) =>
    `btn btn-ghost shrink-0 justify-start whitespace-nowrap text-left md:w-full ${activeSection === section ? "bg-primary text-primary-content" : ""}`;

  if (service_id == null || service_id === "") return <Navigate to={"/"} />;

  const service = serviceQuery.data;

  return (
    <div className="flex h-full min-w-0 flex-col bg-base-100 text-base-content md:flex-row">
      <div className="flex items-center justify-between border-b border-base-300 p-2 sm:p-4 md:hidden">
        <div className="flex flex-col pl-2">
          <span className="text-[10px] uppercase tracking-wider text-base-content/50 font-semibold">Instancia</span>
          <span className="text-sm font-bold text-primary truncate max-w-xs">{service?.name || "Cargando..."}</span>
        </div>
        <button
          className="btn btn-ghost"
          onClick={() => setMenuOpen(!menuOpen)}
        >
          {menuOpen ? (
            <XIcon className="w-5 h-5" />
          ) : (
            <MenuIcon className="w-5 h-5" />
          )}
        </button>
      </div>

      <aside
        className={`${menuOpen ? "block" : "hidden"} w-full border-b border-base-300 bg-base-200 p-2 sm:p-4 md:block md:w-64 md:shrink-0 md:border-b-0 md:border-r`}
      >
        <ul className="menu menu-horizontal w-full flex-nowrap gap-2 overflow-x-auto p-0 md:menu-vertical md:overflow-visible">
            <li className="shrink-0 select-none pointer-events-none md:mb-4">
              <div className="flex items-center gap-2 px-3 py-1.5 bg-primary/10 border border-primary/20 rounded-lg text-left max-w-[180px] md:max-w-none md:flex-col md:items-start md:px-4 md:py-3 md:bg-base-300/40 md:border-base-300 hover:bg-primary/10 md:hover:bg-base-300/40">
                <span className="text-[10px] uppercase tracking-wider text-base-content/50 font-semibold md:mb-1">Instancia</span>
                {serviceQuery.isLoading ? (
                  <div className="h-4 w-20 animate-pulse rounded bg-base-300" />
                ) : serviceQuery.isError ? (
                  <span className="text-error text-xs font-semibold">Error</span>
                ) : (
                  <span className="text-sm font-bold text-primary truncate max-w-[100px] md:max-w-none w-full" title={service?.name}>
                    {service?.name}
                  </span>
                )}
              </div>
            </li>
            <li>
              <button
                className={menuItemClass("general")}
                onClick={() => handleSectionChange("general")}
              >
                Service
              </button>
            </li>
            <li>
              <button
                className={menuItemClass("domains")}
                onClick={() => handleSectionChange("domains")}
              >
                Domains
              </button>
            </li>
            <li>
              <button
                className={menuItemClass("logs")}
                onClick={() => handleSectionChange("logs")}
              >
                Logs
              </button>
            </li>
            <li>
              <button
                className={menuItemClass("snapshots")}
                onClick={() => handleSectionChange("snapshots")}
              >
                Snapshots
              </button>
            </li>
            <li>
              <button
                className={menuItemClass("history")}
                onClick={() => handleSectionChange("history")}
              >
                History
              </button>
            </li>
            <li>
              <button
                className={menuItemClass("files")}
                onClick={() => handleSectionChange("files")}
              >
                Files
              </button>
            </li>
            {/* <li>
              <button
                className={menuItemClass("settings")}
                onClick={() => handleSectionChange("settings")}
              >
                Settings
              </button>
            </li> */}
        </ul>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto p-3 sm:p-4 md:p-6">
        {activeSection === "general" && (
          <div className="mb-8 ">
            <h3 className="text-lg font-semibold mb-6">General</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4 md:py-6">
              <GeneralSection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "domains" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-4">Domains</h3>
            <div className="md:px-4 rounded-box">
              <DomainsSection
                service_id={service_id}
                proxy_id=""
                url_route_suffix="/_/"
              />
            </div>
          </div>
        )}

        {activeSection === "logs" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">Logs</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4">
              <ServiceLogsSection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "history" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">History</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4">
              <OperationHistorySection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "snapshots" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">Snapshots</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4">
              <SnapshotsSection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "files" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">Files</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4">
              <FileManagerSection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "settings" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">Settings</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4 md:py-8">
              Settings panel
            </div>
          </div>
        )}
      </main>
    </div>
  );
};

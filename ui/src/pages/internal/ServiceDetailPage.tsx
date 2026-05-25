import { Navigate, useParams, useSearchParams } from "react-router-dom";
import { useState, useMemo } from "react";
import { MenuIcon, XIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { serviceService } from "../../services/services";
import { useProxyConfigs } from "../../hooks/useProxyConfigs";
import { formatUrl } from "../../utils/url";
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

  const service = serviceQuery.data;
  const proxyInfo = useProxyConfigs();

  const serviceUrls = useMemo((): string[] => {
    if (!service) return [];
    const domains: string[] = [];
    domains.push(...(service.domains ?? []).map(d => d.domain));
    if (proxyInfo.base_domain) {
      domains.push(`${service.id}.${proxyInfo.base_domain}`);
    }
    return domains.map(domain => {
      const customDom = service.domains?.find(d => d.domain === domain);
      const useHttps = customDom ? customDom.use_https === "yes" : proxyInfo.use_https;

      const urlStr = formatUrl(
        useHttps ? "https" : "http",
        domain,
        useHttps ? proxyInfo.https_port : proxyInfo.http_port,
      );
      if (service._pb_install)
        return `${urlStr}/_/#/pbinstal/${service._pb_install}`;
      return `${urlStr}/_/`;
    });
  }, [proxyInfo, service]);

  const handleSectionChange = (section: string) => {
    setSearchParams({ section });
    setMenuOpen(false);
  };

  const menuItemClass = (section: string) =>
    `btn btn-ghost shrink-0 justify-start whitespace-nowrap text-left md:w-full ${activeSection === section ? "bg-primary text-primary-content" : ""}`;

  if (service_id == null || service_id === "") return <Navigate to={"/"} />;

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
            <li className="shrink-0 select-none md:mb-4">
              <div className="dropdown dropdown-bottom md:dropdown-right flex flex-col items-start px-3 py-1.5 bg-primary/10 border border-primary/20 rounded-lg text-left max-w-[180px] md:max-w-none md:flex-col md:items-start md:px-4 md:py-3 md:bg-base-300/40 md:border-base-300 hover:bg-primary/20 cursor-pointer w-full">
                <label tabIndex={0} className="cursor-pointer w-full flex flex-col items-start">
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
                </label>
                {service && serviceUrls.length > 0 && (
                  <ul tabIndex={0} className="dropdown-content menu p-2 shadow-lg bg-base-200 border border-base-300 rounded-box w-56 z-[50] mt-1 select-text">
                    <li className="menu-title text-[9px] uppercase tracking-wider font-bold text-base-content/50 px-2 py-1">Enlaces de acceso</li>
                    {serviceUrls.map((url) => {
                      let cleanLabel = url.replace("http://", "").replace("https://", "");
                      if (cleanLabel.endsWith("/_/")) cleanLabel = cleanLabel.slice(0, -3);
                      if (cleanLabel.includes("/_/#/")) {
                        cleanLabel = cleanLabel.split("/_/#/")[0];
                      }
                      return (
                        <li key={url}>
                          <a
                            href={url}
                            target="_blank"
                            rel="noreferrer"
                            className="text-xs text-primary hover:text-primary-focus truncate w-full flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-primary/10"
                          >
                            <span className="truncate w-full block" title={url}>{cleanLabel}</span>
                          </a>
                        </li>
                      );
                    })}
                  </ul>
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

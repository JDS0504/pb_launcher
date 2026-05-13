import { Navigate, useParams, useSearchParams } from "react-router-dom";
import { useState } from "react";
import { MenuIcon, XIcon } from "lucide-react";
import { GeneralSection } from "./details_section/GeneralSection";
import { DomainsSection } from "./details_section/DomainsSection";
import { ServiceLogsSection } from "./details_section/ServiceLogsSection";
import { OperationHistorySection } from "./details_section/OperationHistorySection";
import { PBHooksSection } from "./details_section/PBHooksSection";

export const ServiceDetailPage = () => {
  const { service_id } = useParams<{ service_id: string }>();

  const [searchParams, setSearchParams] = useSearchParams();
  const activeSection = searchParams.get("section") || "general";
  const [menuOpen, setMenuOpen] = useState(false);

  const handleSectionChange = (section: string) => {
    setSearchParams({ section });
    setMenuOpen(false);
  };

  const menuItemClass = (section: string) =>
    `btn btn-ghost justify-start w-full text-left ${activeSection === section ? "bg-primary text-primary-content" : ""}`;

  if (service_id == null || service_id === "") return <Navigate to={"/"} />;
  return (
    <div className="flex h-full flex-col md:flex-row bg-base-100 text-base-content">
      <div className="md:hidden p-4 flex justify-end items-center border-b border-base-300">
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

      {(menuOpen || window.innerWidth >= 768) && (
        <aside className="w-full md:w-64 bg-base-200 p-4 border-b md:border-b-0 md:border-r border-base-300 md:block">
          <ul className="menu flex md:flex-col flex-wrap gap-2">
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
                className={menuItemClass("history")}
                onClick={() => handleSectionChange("history")}
              >
                History
              </button>
            </li>
            <li>
              <button
                className={menuItemClass("hooks")}
                onClick={() => handleSectionChange("hooks")}
              >
                PB Hooks
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
      )}

      <main className="flex-1 p-4 md:p-6 overflow-auto">
        {activeSection === "general" && (
          <div className="mb-8 ">
            <h3 className="text-lg font-semibold mb-6">General</h3>
            <div className="md:px-4 py-6 bg-base-200 rounded-box">
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
            <div className="px-4 py-4 bg-base-200 rounded-box">
              <ServiceLogsSection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "history" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">History</h3>
            <div className="px-4 py-4 bg-base-200 rounded-box">
              <OperationHistorySection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "hooks" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">PB Hooks</h3>
            <div className="px-4 py-4 bg-base-200 rounded-box">
              <PBHooksSection service_id={service_id} />
            </div>
          </div>
        )}

        {activeSection === "settings" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-6">Settings</h3>
            <div className="px-4 py-8 bg-base-200 rounded-box">
              Settings panel
            </div>
          </div>
        )}
      </main>
    </div>
  );
};

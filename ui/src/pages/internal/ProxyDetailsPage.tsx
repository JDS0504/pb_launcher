import { Navigate, useParams, useSearchParams } from "react-router-dom";
import { useState } from "react";
import { MenuIcon, XIcon } from "lucide-react";
import { ProxyGeneralSection } from "./proxy_section/ProxyGeneralSection";
import { DomainsSection } from "./details_section/DomainsSection";

export const ProxyDetailsPage = () => {
  const { proxy_id } = useParams<{ proxy_id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const activeSection = searchParams.get("section") || "general";
  const [menuOpen, setMenuOpen] = useState(false);

  const handleSectionChange = (section: string) => {
    setSearchParams({ section });
    setMenuOpen(false);
  };

  const menuItemClass = (section: string) =>
    `btn btn-ghost shrink-0 justify-start whitespace-nowrap text-left w-full ${activeSection === section ? "bg-primary text-primary-content" : ""}`;

  if (proxy_id == null || proxy_id === "") return <Navigate to={"/"} />;
  return (
    <div className="flex h-full min-w-0 flex-col bg-base-100 text-base-content md:flex-row">
      <div className="flex items-center justify-end border-b border-base-300 p-2 sm:p-4 md:hidden">
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
        <ul className="menu menu-vertical w-full gap-2 p-0 md:overflow-visible">
            <li>
              <button
                className={menuItemClass("general")}
                onClick={() => handleSectionChange("general")}
              >
                Proxy Entry
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
        </ul>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto p-3 sm:p-4 md:p-6">
        {activeSection === "general" && (
          <div className="mb-8 ">
            <h3 className="text-lg font-semibold mb-6">General</h3>
            <div className="rounded-box bg-base-200 p-3 sm:p-4 md:py-6">
              <ProxyGeneralSection proxy_id={proxy_id} />
            </div>
          </div>
        )}

        {activeSection === "domains" && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold mb-4">Domains</h3>
            <div className="md:px-4 rounded-box">
              <DomainsSection
                proxy_id={proxy_id}
                service_id=""
                url_route_suffix={""}
              />
            </div>
          </div>
        )}
      </main>
    </div>
  );
};

import { Routes, Route, Link, useLocation } from "react-router";
import classNames from "classnames";
import { Instructions } from "./Instructions";
import { ConfigReference } from "./ConfigReference";

export const DocsLayout: React.FC = () => {
  const location = useLocation();

  const isActive = (path: string): boolean => location.pathname === path;

  return (
    <div className="min-h-screen bg-gray-100 md:flex">
      <div className="bg-gray-900 text-white p-4 md:fixed md:h-screen md:w-64 md:p-6">
        <Link
          to="/"
          className="block text-2xl font-bold hover:text-gray-200 md:mb-8"
        >
          PBLauncher
        </Link>
        <nav className="mt-4 flex gap-2 overflow-x-auto pb-1 md:mt-0 md:flex-col md:gap-3 md:overflow-visible md:pb-0">
          <Link
            to="/"
            className={classNames("shrink-0 rounded px-3 py-2 hover:bg-gray-800", {
              "bg-gray-800 font-bold": isActive("/"),
            })}
          >
            Home
          </Link>
          <Link
            to="/docs"
            className={classNames("shrink-0 rounded px-3 py-2 hover:bg-gray-800", {
              "bg-gray-800 font-bold": isActive("/docs"),
            })}
          >
            Production
          </Link>
          <Link
            to="/docs/config"
            className={classNames(
              "shrink-0 rounded px-3 py-2 hover:bg-gray-800",
              {
                "bg-gray-800 font-bold": isActive("/docs/config"),
              },
            )}
          >
            Config Reference
          </Link>
        </nav>
      </div>
      <main className="min-w-0 flex-1 p-4 md:ml-64 md:p-8">
        <Routes>
          <Route path="/" element={<Instructions />} />
          <Route path="config" element={<ConfigReference />} />
        </Routes>
      </main>
    </div>
  );
};

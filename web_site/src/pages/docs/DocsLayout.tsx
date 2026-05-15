import { Routes, Route, Link, useLocation } from "react-router";
import classNames from "classnames";
import { Instructions } from "./Instructions";
import { ConfigReference } from "./ConfigReference";

export const DocsLayout: React.FC = () => {
  const location = useLocation();

  const isActive = (path: string): boolean => location.pathname === path;

  return (
    <div className="flex">
      <div className="w-64 h-screen bg-gray-900 text-white p-6 flex flex-col gap-4 fixed">
        <Link to="/" className="text-2xl font-bold mb-8 hover:text-gray-200">
          PBLauncher
        </Link>
        <nav className="flex flex-col gap-3">
          <Link
            to="/"
            className={classNames("rounded px-3 py-2 hover:bg-gray-800", {
              "bg-gray-800 font-bold": isActive("/"),
            })}
          >
            Home
          </Link>
          <Link
            to="/docs"
            className={classNames("rounded px-3 py-2 hover:bg-gray-800", {
              "bg-gray-800 font-bold": isActive("/docs"),
            })}
          >
            Production
          </Link>
          <Link
            to="/docs/config"
            className={classNames("rounded px-3 py-2 hover:bg-gray-800", {
              "bg-gray-800 font-bold": isActive("/docs/config"),
            })}
          >
            Config Reference
          </Link>
        </nav>
      </div>
      <div className="flex-1 ml-64 p-8">
        <Routes>
          <Route path="/" element={<Instructions />} />
          <Route path="config" element={<ConfigReference />} />
        </Routes>
      </div>
    </div>
  );
};

import { Routes, Route } from "react-router";
import { LandingPage } from "./pages/LandingPage";
import { DocsLayout } from "./pages/docs/DocsLayout";

export const App = () => {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/docs/*" element={<DocsLayout />} />
    </Routes>
  );
};

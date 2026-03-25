import { useState } from "react";
import { BrowserRouter, Routes, Route, NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Box,
  Shield,
  Menu,
  X,
} from "lucide-react";
import Dashboard from "./components/Dashboard";
import SandboxList from "./components/SandboxList";
import SandboxDetail from "./components/SandboxDetail";
import ReplayView from "./components/ReplayView";
import PolicyEditor from "./components/PolicyEditor";

const navItems = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/sandboxes", icon: Box, label: "Sandboxes" },
  { to: "/policies", icon: Shield, label: "Policies" },
] as const;

function Sidebar({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <>
      {/* Mobile overlay */}
      {open && (
        <div
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={onClose}
        />
      )}
      <aside
        className={`fixed top-0 left-0 z-40 h-full w-60 bg-gray-800 border-r border-gray-700 flex flex-col transition-transform lg:translate-x-0 ${open ? "translate-x-0" : "-translate-x-full"}`}
      >
        <div className="flex items-center gap-2 px-4 h-14 border-b border-gray-700">
          <Box className="w-6 h-6 text-blue-400" />
          <span className="text-lg font-semibold text-gray-100">
            AgentSandbox
          </span>
          <button
            className="ml-auto lg:hidden text-gray-400 hover:text-gray-100"
            onClick={onClose}
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <nav className="flex-1 py-4 space-y-1 px-2">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === "/"}
              onClick={onClose}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-blue-600/20 text-blue-400"
                    : "text-gray-400 hover:bg-gray-700 hover:text-gray-100"
                }`
              }
            >
              <Icon className="w-5 h-5" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="px-4 py-3 border-t border-gray-700 text-xs text-gray-500">
          AgentSandbox v0.1.0
        </div>
      </aside>
    </>
  );
}

export default function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-900 text-gray-100">
        <Sidebar
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
        />

        {/* Main content */}
        <div className="lg:pl-60">
          {/* Top bar (mobile) */}
          <header className="sticky top-0 z-20 flex items-center h-14 px-4 bg-gray-900/80 backdrop-blur border-b border-gray-800 lg:hidden">
            <button
              onClick={() => setSidebarOpen(true)}
              className="text-gray-400 hover:text-gray-100"
            >
              <Menu className="w-6 h-6" />
            </button>
            <span className="ml-3 text-lg font-semibold">AgentSandbox</span>
          </header>

          <main className="p-4 sm:p-6 lg:p-8">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/sandboxes" element={<SandboxList />} />
              <Route path="/sandboxes/:id" element={<SandboxDetail />} />
              <Route path="/sandboxes/:id/replay" element={<ReplayView />} />
              <Route path="/policies" element={<PolicyEditor />} />
            </Routes>
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
}

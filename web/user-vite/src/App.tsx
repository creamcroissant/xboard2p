import { lazy, Suspense } from "react";
import { Navigate, Outlet, Route, Routes } from "react-router-dom";
import { useAuth } from "@/providers/AuthProvider";
import { AppLayout } from "@/components/layout";
import { Loading } from "@/components/ui";
import { ROUTES, ADMIN_ROUTES } from "@/lib/constants";

// Lazy load pages
const Login = lazy(() => import("@/pages/auth/Login"));
const Register = lazy(() => import("@/pages/auth/Register"));
const Install = lazy(() => import("@/pages/install"));
const Dashboard = lazy(() => import("@/pages/dashboard"));
const Servers = lazy(() => import("@/pages/servers"));
const Plans = lazy(() => import("@/pages/plans"));
const TrafficStats = lazy(() => import("@/pages/traffic"));
const Knowledge = lazy(() => import("@/pages/knowledge"));
const Settings = lazy(() => import("@/pages/settings"));
const NotFound = lazy(() => import("@/pages/NotFound"));

// Admin pages
const AdminAgents = lazy(() => import("@/pages/admin/agents"));
const AdminUsers = lazy(() => import("@/pages/admin/users"));
const AdminPlans = lazy(() => import("@/pages/admin/plans"));
const AdminNotices = lazy(() => import("@/pages/admin/notices"));
const AdminKnowledge = lazy(() => import("@/pages/admin/knowledge"));
const AdminSystem = lazy(() => import("@/pages/admin/system"));
const AdminForwarding = lazy(() => import("@/pages/admin/forwarding"));
const AdminAccessLogs = lazy(() => import("@/pages/admin/access-logs"));

// Route guard for authenticated routes
function RequireAuth() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <Loading fullScreen />;
  }

  if (!isAuthenticated) {
    return <Navigate to={ROUTES.LOGIN} replace />;
  }

  return (
    <AppLayout>
      <Suspense fallback={<Loading />}>
        <Outlet />
      </Suspense>
    </AppLayout>
  );
}

// Route guard for public routes (redirect if already logged in)
function PublicRoute() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <Loading fullScreen />;
  }

  if (isAuthenticated) {
    return <Navigate to={ROUTES.DASHBOARD} replace />;
  }

  return (
    <Suspense fallback={<Loading fullScreen />}>
      <Outlet />
    </Suspense>
  );
}

// Route guard for admin-only routes
function RequireAdminAuth() {
  const { isAuthenticated, isAdmin, isLoading } = useAuth();

  if (isLoading) {
    return <Loading fullScreen />;
  }

  if (!isAuthenticated) {
    return <Navigate to={ROUTES.LOGIN} replace />;
  }

  if (!isAdmin) {
    return <Navigate to={ROUTES.DASHBOARD} replace />;
  }

  return (
    <AppLayout>
      <Suspense fallback={<Loading />}>
        <Outlet />
      </Suspense>
    </AppLayout>
  );
}

export default function App() {
  return (
    <Routes>
      {/* Install route - no auth required */}
      <Route
        path={ROUTES.INSTALL}
        element={
          <Suspense fallback={<Loading fullScreen />}>
            <Install />
          </Suspense>
        }
      />

      {/* Public routes */}
      <Route element={<PublicRoute />}>
        <Route path={ROUTES.LOGIN} element={<Login />} />
        <Route path={ROUTES.REGISTER} element={<Register />} />
      </Route>

      {/* Protected routes */}
      <Route element={<RequireAuth />}>
        <Route index element={<Navigate to={ROUTES.DASHBOARD} replace />} />
        <Route path={ROUTES.DASHBOARD} element={<Dashboard />} />
        <Route path={ROUTES.SERVERS} element={<Servers />} />
        <Route path={ROUTES.PLANS} element={<Plans />} />
        <Route path={ROUTES.TRAFFIC} element={<TrafficStats />} />
        <Route path={ROUTES.KNOWLEDGE} element={<Knowledge />} />
        <Route path={ROUTES.SETTINGS} element={<Settings />} />
      </Route>

      {/* Admin routes */}
      <Route element={<RequireAdminAuth />}>
        <Route path={ADMIN_ROUTES.AGENTS} element={<AdminAgents />} />
        <Route path={ADMIN_ROUTES.USERS} element={<AdminUsers />} />
        <Route path={ADMIN_ROUTES.PLANS} element={<AdminPlans />} />
        <Route path={ADMIN_ROUTES.NOTICES} element={<AdminNotices />} />
        <Route path={ADMIN_ROUTES.KNOWLEDGE} element={<AdminKnowledge />} />
        <Route path={ADMIN_ROUTES.SYSTEM} element={<AdminSystem />} />
        <Route path={ADMIN_ROUTES.FORWARDING} element={<AdminForwarding />} />
        <Route path={ADMIN_ROUTES.ACCESS_LOGS} element={<AdminAccessLogs />} />
      </Route>

      {/* 404 */}
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}

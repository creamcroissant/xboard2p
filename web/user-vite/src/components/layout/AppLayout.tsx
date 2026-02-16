import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Sidebar from "./Sidebar";
import Header from "./Header";
import NoticePopup from "@/components/NoticePopup";
import { fetchUnreadNotice, markNoticeRead } from "@/api/notice";
import { QUERY_KEYS } from "@/lib/constants";
import { useAuth } from "@/providers/AuthProvider";

interface AppLayoutProps {
  children: ReactNode;
}

export default function AppLayout({ children }: AppLayoutProps) {
  const { isAuthenticated } = useAuth();
  const queryClient = useQueryClient();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    if (typeof window === "undefined") {
      return false;
    }
    return window.localStorage.getItem("xboard.sidebar.collapsed") === "true";
  });

  const noticeQuery = useQuery({
    queryKey: QUERY_KEYS.USER_NOTICE,
    queryFn: fetchUnreadNotice,
    enabled: isAuthenticated,
    refetchOnWindowFocus: false,
  });

  const markReadMutation = useMutation({
    mutationFn: (id: number) => markNoticeRead(id),
    onSuccess: () => {
      queryClient.setQueryData(QUERY_KEYS.USER_NOTICE, null);
    },
  });

  const notice = noticeQuery.data ?? null;
  const showNotice = Boolean(notice);

  const handleNoticeClose = () => {
    if (!notice) {
      return;
    }
    markReadMutation.mutate(notice.id);
  };

  const containerStyle = useMemo(
    () =>
      ({
        "--sidebar-width": sidebarCollapsed
          ? "var(--sidebar-width-collapsed)"
          : "var(--sidebar-width-expanded)",
      }) as React.CSSProperties,
    [sidebarCollapsed]
  );

  useEffect(() => {
    window.localStorage.setItem(
      "xboard.sidebar.collapsed",
      String(sidebarCollapsed)
    );
  }, [sidebarCollapsed]);

  return (
    <div className="h-screen bg-background overflow-hidden" style={containerStyle}>
      <NoticePopup notice={notice} open={showNotice} onClose={handleNoticeClose} />
      <div className="flex h-full">
        <Sidebar
          isOpen={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
          isCollapsed={sidebarCollapsed}
          onToggleCollapsed={() =>
            setSidebarCollapsed((previous) => !previous)
          }
        />

        <div className="flex-1 flex flex-col min-h-0 min-w-0 transition-all duration-300">
          <Header onMenuClick={() => setSidebarOpen(true)} />

          <main className="flex-1 overflow-y-auto p-4 md:p-6 lg:p-8">
            <div className="w-full mx-auto px-2 md:px-4">{children}</div>
          </main>
        </div>
      </div>
    </div>
  );
}

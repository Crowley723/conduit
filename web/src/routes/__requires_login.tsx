import { createFileRoute, Outlet, redirect } from '@tanstack/react-router';
import { FileText, LayoutGrid, ScrollText, Users, Key, Activity, FilePlus } from 'lucide-react';
import { GetLoginStatus, authRedirect } from '@/services/Auth.ts';
import { useAuth } from '@/hooks/useAuth.ts';
import { AppSidebar } from '@/components/app-sidebar.tsx';
import type { SidebarMode } from '@/components/app-sidebar.tsx';
import {
  SidebarInset,
  SidebarProvider,
} from '@/components/ui/sidebar.tsx';

export const Route = createFileRoute('/__requires_login')({
  beforeLoad: async ({ location }) => {
    const response = await GetLoginStatus();
    if (!response?.authenticated) {
      authRedirect.save(location.pathname);
      throw redirect({ to: '/login' });
    }
  },
  component: AppLayout,
});

function AppLayout() {
  const { user } = useAuth();

  const modes: SidebarMode[] = [
    {
      label: 'Admin',
      isAdmin: true,
      navItems: [
        { title: 'Review Queue', url: '/admin/review-queue', icon: LayoutGrid, isActive: true, badge: 7 },
        { title: 'All Certificates', url: '/admin/certificates', icon: FileText },
        { title: 'Audit Log', url: '/admin/audit-log', icon: ScrollText },
        { title: 'Users & Service Accounts', url: '/admin/users', icon: Users },
      ],
    },
    {
      label: 'User',
      navItems: [
        { title: 'My Certificates', url: '/user/my-certificates', icon: FileText },
        { title: 'Request Certificate', url: '/user/request-certificate', icon: FilePlus },
        { title: 'Activity', url: '/user/activity', icon: Activity },
        { title: 'Service Accounts', url: '/user/service-accounts', icon: Key },
      ],
    },
  ];

  return (
    <SidebarProvider>
      <AppSidebar modes={modes} user={user ?? undefined} />
      <SidebarInset>
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  );
}

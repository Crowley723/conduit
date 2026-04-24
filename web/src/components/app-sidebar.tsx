'use client';

import * as React from 'react';
import { ChevronsUpDown, Check } from 'lucide-react';
import { NavMain } from '@/components/ui/nav-main.tsx';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu.tsx';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from '@/components/ui/sidebar.tsx';
import type { LucideIcon } from 'lucide-react';
import { Bug, DatabaseZap } from 'lucide-react';
import { Link, useNavigate } from '@tanstack/react-router';
import type { User } from '@/services/Auth.ts';
import { getInitials } from '@/services/Utils.ts';
import { useAuth } from '@/hooks/useAuth.ts';

interface NavGroup {
  groupLabel?: string;
  items: NavItem[];
}

interface NavItem {
  title: string;
  url: string;
  icon?: LucideIcon;
  isActive?: boolean;
  badge?: number;
  items?: NavItem[];
}

export interface SidebarMode {
  label: string;
  isAdmin?: boolean;
  navItems: NavItem[] | NavGroup[];
}

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  modes: SidebarMode[];
  user?: User;
  storageKey?: string;
}

function ModeSwitcher({
  modes,
  activeMode,
  onSelect,
}: {
  modes: SidebarMode[];
  activeMode: SidebarMode;
  onSelect: (mode: SidebarMode) => void;
}) {
  const { isMobile } = useSidebar();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <div className="flex h-8 w-8 items-center justify-center rounded-md bg-blue-500 text-sm font-semibold text-white">
                C
              </div>
              <div className="flex min-w-0 flex-1 flex-col text-left">
                <span className="truncate text-sm font-semibold">Conduit</span>
                <span className="truncate text-xs text-neutral-500">
                  {activeMode.label}
                </span>
              </div>
              <ChevronsUpDown className="ml-auto shrink-0 opacity-50" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-48 rounded-lg"
            align="start"
            side={isMobile ? 'bottom' : 'right'}
            sideOffset={4}
          >
            {modes.map((mode) => (
              <DropdownMenuItem
                key={mode.label}
                onClick={() => onSelect(mode)}
                className="gap-2 p-2"
              >
                <div className="flex h-6 w-6 items-center justify-center rounded-md bg-blue-500 text-[10px] font-semibold text-white">
                  C
                </div>
                <span className="flex-1">{mode.label}</span>
                {mode.label === activeMode.label && (
                  <Check className="h-4 w-4 opacity-60" />
                )}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function UserFooter({ user }: { user: User }) {
  const { isMobile } = useSidebar();
  const navigate = useNavigate();
  const { logout } = useAuth();

  const handleLogout = () => {
    logout(undefined, {
      onSuccess: () => navigate({ to: '/login' }),
    });
  };

  return (
    <>
      <SidebarSeparator />
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton size="lg" tooltip="Account">
                  <div className="flex h-7 w-7 items-center justify-center rounded-full bg-blue-900 text-xs font-semibold text-blue-400 shrink-0">
                    {getInitials(user.display_name)}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium text-neutral-300">
                      {user.display_name}
                    </p>
                    <p className="truncate text-[11px] text-neutral-500">
                      {user.email}
                    </p>
                  </div>
                  <ChevronsUpDown className="ml-auto shrink-0 opacity-50" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                className="w-(--radix-dropdown-menu-trigger-width) min-w-48 rounded-lg"
                side={isMobile ? 'bottom' : 'right'}
                align="end"
                sideOffset={4}
              >
                <DropdownMenuItem asChild>
                  <Link to="/user/profile">Profile</Link>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  onClick={handleLogout}
                >
                  Logout
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </>
  );
}

export function AppSidebar({
  modes,
  user,
  storageKey = 'conduitSidebarMode',
  ...props
}: AppSidebarProps) {
  const [activeMode, setActiveMode] = React.useState<SidebarMode>(() => {
    const saved = localStorage.getItem(storageKey);
    return modes.find((m) => m.label === saved) ?? modes[0];
  });

  const handleSelect = (mode: SidebarMode) => {
    setActiveMode(mode);
    localStorage.setItem(storageKey, mode.label);
  };

  const navItems = activeMode.navItems;
  const isGrouped = navItems.length > 0 && 'groupLabel' in navItems[0];

  const [routerDevtoolsOpen, setRouterDevtoolsOpen] = React.useState(
    () => localStorage.getItem('tanstackRouterDevtoolsOpen') === 'true'
  );
  const [queryDevtoolsOpen, setQueryDevtoolsOpen] = React.useState(false);

  React.useEffect(() => {
    const observer = new MutationObserver(() => {
      setRouterDevtoolsOpen(
        localStorage.getItem('tanstackRouterDevtoolsOpen') === 'true'
      );
      setQueryDevtoolsOpen(
        !!document.querySelector('[aria-label="Close tanstack query devtools"]')
      );
    });
    observer.observe(document.body, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, []);

  const toggleRouterDevtools = () => {
    const btn = document.querySelector<HTMLButtonElement>(
      '[aria-label="Open TanStack Router Devtools"]'
    );
    btn?.click();
  };

  const toggleQueryDevtools = () => {
    const btn = document.querySelector<HTMLButtonElement>(
      '[aria-label="Open Tanstack query devtools"]'
    );
    btn?.click();
  };

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        {modes.length > 1 ? (
          <ModeSwitcher
            modes={modes}
            activeMode={activeMode}
            onSelect={handleSelect}
          />
        ) : (
          <div className="flex items-center gap-2.5 px-2 py-1.5">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-blue-500 text-xs font-semibold text-white">
              C
            </div>
            <span className="flex-1 text-sm font-semibold">
              {activeMode.label}
            </span>
            {activeMode.isAdmin && (
              <span className="rounded border border-neutral-700 bg-neutral-800 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-widest text-neutral-500">
                Admin
              </span>
            )}
          </div>
        )}
      </SidebarHeader>

      <SidebarSeparator />

      <SidebarContent>
        {isGrouped ? (
          (navItems as NavGroup[]).map((group, index) => (
            <NavMain
              key={index}
              items={group.items}
              groupLabel={group.groupLabel}
            />
          ))
        ) : (
          <NavMain items={navItems as NavItem[]} />
        )}
        <SidebarGroup className="mt-auto">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                onClick={toggleRouterDevtools}
                isActive={routerDevtoolsOpen}
                tooltip="Router Devtools"
              >
                <Bug />
                <span>Router Devtools</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                onClick={toggleQueryDevtools}
                isActive={queryDevtoolsOpen}
                tooltip="Query Devtools"
              >
                <DatabaseZap />
                <span>Query Devtools</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>

      {user && <UserFooter user={user} />}
    </Sidebar>
  );
}

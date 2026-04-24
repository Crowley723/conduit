import {
  NavigationMenu,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
} from '@/components/ui/navigation-menu.tsx';
import { Link } from '@tanstack/react-router';
import { LoginDialog } from '@/components/LoginDialog.tsx';
import { Button } from '@/components/ui/button.tsx';
import UserDropdown from '@/components/UserDropdown.tsx';
import { useAuth } from '@/hooks/useAuth.ts';

export function Header() {
  const { login, isLoggingIn, logout, isAuthenticated, user } = useAuth();

  return (
    <header className="w-full border-b">
      <div className="container mx-auto px-6 py-4 flex items-center justify-between">
        <div className="w-[76px]"></div>
        <NavigationMenu className="justify-self-center">
          <NavigationMenuList>
            <NavigationMenuItem>
              <NavigationMenuLink asChild>
                <Link to={'/'} search={{} as any}>
                  Home
                </Link>
              </NavigationMenuLink>
            </NavigationMenuItem>
            <NavigationMenuItem>
              <NavigationMenuLink asChild>
                <Link to={'/about'}>About</Link>
              </NavigationMenuLink>
            </NavigationMenuItem>
          </NavigationMenuList>
        </NavigationMenu>

        <div className="justify-self-end">
          {isAuthenticated && user ? (
            <UserDropdown user={user} onLogout={logout} />
          ) : (
            <LoginDialog login={login} isLoggingIn={isLoggingIn}>
              <Button>Login</Button>
            </LoginDialog>
          )}
        </div>
      </div>
    </header>
  );
}

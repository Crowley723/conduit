import {
  DropdownMenuTrigger,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu.tsx';
import { Button } from '@/components/ui/button.tsx';
import { Avatar, AvatarFallback } from '@/components/ui/avatar.tsx';
import { ChevronDownIcon } from 'lucide-react';
import type { User } from '@/services/auth.tsx';
import { Link } from '@tanstack/react-router';

const UserMenu = ({
  user,
  onLogout,
}: {
  user?: User | null;
  onLogout?: () => void;
}) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="h-8 px-2 py-0 hover:bg-accent hover:text-accent-foreground"
        >
          <Avatar className="h-6 w-6">
            <AvatarFallback className="text-xs">
              {user?.display_name
                .split(' ')
                .map((n) => n[0])
                .join('')}
            </AvatarFallback>
          </Avatar>
          <ChevronDownIcon className="h-3 w-3 ml-1" />
          <span className="sr-only">User menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel>
          <div className="flex flex-col space-y-1">
            <p className="text-sm font-medium leading-none">{user?.username}</p>
            <p className="text-xs leading-none text-muted-foreground">
              {user?.email}
            </p>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link to={'/user/profile'}>Profile</Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link to={'/settings'}>Settings</Link>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onLogout}>Log out</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export default UserMenu;

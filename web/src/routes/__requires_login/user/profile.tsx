import { createFileRoute } from '@tanstack/react-router';
import { requireAuth } from '@/utils/Auth.ts';
import { useAuth } from '@/hooks/useAuth';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';

export const Route = createFileRoute('/__requires_login/user/profile')({
  component: RouteComponent,
  beforeLoad: async ({ location }) => {
    await requireAuth(location, 'You must login to view your profile.', true);
  },
});

function RouteComponent() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="container mx-auto p-6 max-w-4xl">
        <div className="text-center py-12">Loading...</div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="container mx-auto p-6 max-w-4xl">
        <div className="text-center py-12 text-destructive">
          Not authenticated
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 max-w-4xl">
      <div className="mb-6">
        <h1 className="text-3xl font-bold mb-2">Profile</h1>
        <p className="text-muted-foreground">
          View your account information and group memberships
        </p>
      </div>

      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Account Information</CardTitle>
            <CardDescription>
              Your identity and authentication details
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableBody>
                <TableRow>
                  <TableHead className="w-1/3">Display Name</TableHead>
                  <TableCell className="font-medium">{user.display_name}</TableCell>
                </TableRow>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableCell className="font-mono text-sm">{user.username}</TableCell>
                </TableRow>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableCell>{user.email}</TableCell>
                </TableRow>
                <TableRow>
                  <TableHead>Subject (sub)</TableHead>
                  <TableCell className="font-mono text-xs">{user.sub}</TableCell>
                </TableRow>
                <TableRow>
                  <TableHead>Issuer (iss)</TableHead>
                  <TableCell className="font-mono text-xs">{user.iss}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Group Memberships</CardTitle>
            <CardDescription>
              Groups you belong to for authorization and access control
            </CardDescription>
          </CardHeader>
          <CardContent>
            {user.groups && user.groups.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {user.groups.map((group) => (
                  <Badge key={group} variant="secondary">
                    {group}
                  </Badge>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground text-sm">
                You are not a member of any groups
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Login } from '@/services/Auth';

export const Route = createFileRoute('/login')({
  component: RouteComponent,
});

function RouteComponent() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const handleUsernamePasswordSubmit = (e: { preventDefault: () => void }) => {
    e.preventDefault();
    console.log('Username/Password login:', { username, password });
  };

  const handleAutheliaLogin = () => {
    Login();
  };

  return (
    <div className="flex items-start justify-center min-h-screen pt-[20vh]">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Sign In to Continue</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleUsernamePasswordSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                placeholder="Enter your username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit" className="w-full">
              Sign In
            </Button>
          </form>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <Separator />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-card px-2 text-muted-foreground">
                Or continue with
              </span>
            </div>
          </div>

          <Button
            variant="outline"
            className="w-full"
            onClick={handleAutheliaLogin}
          >
            Login with Authelia
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

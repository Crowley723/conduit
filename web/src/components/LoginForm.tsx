import { Button } from './ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card.tsx';
import { cn } from '@/services/Utils';
import React from 'react';

export function LoginForm({
  className,
  onLogin,
  isLoading,
  message,
  error,
  ...props
}: React.ComponentProps<'div'> & {
  onLogin?: () => void;
  isLoading?: boolean;
  message?: string;
  error?: boolean;
}) {
  return (
    <div className={cn('flex flex-col gap-6', className)} {...props}>
      <Card>
        <CardHeader>
          <CardTitle>Login to your account</CardTitle>
        </CardHeader>
        <CardContent>
          {message && (
            <p
              className={
                error
                  ? 'text-red-600 mb-4 text-center'
                  : 'text-gray-600 mb-4 text-center'
              }
            >
              {message}
            </p>
          )}
          <div className="flex flex-col gap-3">
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={onLogin}
              disabled={isLoading}
            >
              {isLoading ? 'Starting login...' : 'Login with Authelia'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

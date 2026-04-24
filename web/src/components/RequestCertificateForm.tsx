import { Button } from './ui/button';
import { Card, CardContent } from '@/components/ui/card.tsx';
import { Input } from '@/components/ui/input.tsx';
import { Label } from '@/components/ui/label.tsx';
import { Textarea } from '@/components/ui/textarea.tsx';
import { cn } from '@/services/Utils';
import React from 'react';

interface RequestCertificateFormProps {
  className?: string;
  onSubmit: (data: { message: string; validity_days: number }) => void;
  isLoading?: boolean;
  errorMessage?: string;
  successMessage?: string;
}

export function RequestCertificateForm({
  className,
  onSubmit,
  isLoading,
  errorMessage,
  successMessage,
}: RequestCertificateFormProps) {
  const [message, setMessage] = React.useState('');
  const [validityDays, setValidityDays] = React.useState(90);
  const [errors, setErrors] = React.useState<{
    message?: string;
    validity_days?: string;
  }>({});

  const validateForm = (): boolean => {
    const newErrors: { message?: string; validity_days?: string } = {};

    if (!message.trim()) {
      newErrors.message = 'Message is required';
    }

    if (!validityDays || validityDays < 1) {
      newErrors.validity_days = 'Validity days must be at least 1';
    } else if (validityDays > 365) {
      newErrors.validity_days = 'Validity days must be at most 365';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validateForm()) {
      onSubmit({
        message: message.trim(),
        validity_days: validityDays,
      });
    }
  };

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <Card>
        <CardContent>
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {errorMessage && (
              <div className="text-destructive text-sm p-3 rounded-md bg-destructive/10 border border-destructive/20">
                {errorMessage}
              </div>
            )}
            {successMessage && (
              <div className="text-green-600 dark:text-green-400 text-sm p-3 rounded-md bg-green-500/10 border border-green-500/20">
                {successMessage}
              </div>
            )}

            <div className="flex flex-col gap-2">
              <Label htmlFor="message">
                Reason
                <span className="text-destructive ml-1">*</span>
              </Label>
              <Textarea
                id="message"
                placeholder="Describe why you need this certificate..."
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                disabled={isLoading}
                aria-invalid={!!errors.message}
                rows={4}
              />
              {errors.message && (
                <span className="text-destructive text-xs">
                  {errors.message}
                </span>
              )}
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="validity_days">
                Validity (days)
                <span className="text-destructive ml-1">*</span>
              </Label>
              <Input
                id="validity_days"
                type="number"
                min={1}
                max={365}
                value={validityDays}
                onChange={(e) =>
                  setValidityDays(parseInt(e.target.value, 10) || 0)
                }
                disabled={isLoading}
                aria-invalid={!!errors.validity_days}
              />
              {errors.validity_days && (
                <span className="text-destructive text-xs">
                  {errors.validity_days}
                </span>
              )}
            </div>

            <div className="flex flex-col gap-2 pt-2">
              <Button type="submit" disabled={isLoading} className="w-full">
                {isLoading ? 'Submitting request...' : 'Submit Request'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

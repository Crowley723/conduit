import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { requireAuth } from '@/utils/Auth.ts';
import { useState } from 'react';
import { useMyCertificateRequests } from '@/api/Certificates';
import type { CertificateRequest, CertificateRequestStatus } from '@/types/Certificates';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { DownloadCertificateDialog } from '@/components/DownloadCertificateDialog';

export const Route = createFileRoute('/__requires_login/user/my-certificates')({
  component: RouteComponent,
  beforeLoad: async ({ location }) => {
    await requireAuth(location, 'You must login to view your certificates.', true);
  },
});

type DisplayStatus = 'active' | 'issued' | 'awaiting_review' | 'expiring_soon' | 'expired' | 'pending' | 'rejected' | 'failed' | 'completed' | 'approved';

function getDisplayStatus(cert: CertificateRequest): DisplayStatus {
  if (cert.status === 'issued') {
    if (cert.expires_at) {
      const now = new Date();
      const expires = new Date(cert.expires_at);
      if (expires <= now) return 'expired';
      const daysRemaining = (expires.getTime() - now.getTime()) / (1000 * 60 * 60 * 24);
      if (daysRemaining <= 30) return 'expiring_soon';
    }
    return 'active';
  }
  return cert.status as DisplayStatus;
}

function StatusBadge({ status }: { status: DisplayStatus }) {
  const config: Record<DisplayStatus, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' | 'warning' | 'success' | 'info' }> = {
    active:         { label: 'Active',          variant: 'success' },
    issued:         { label: 'Issued',          variant: 'success' },
    expiring_soon:  { label: 'Expiring Soon',   variant: 'warning' },
    expired:        { label: 'Expired',         variant: 'destructive' },
    awaiting_review:{ label: 'Awaiting Review', variant: 'warning' },
    approved:       { label: 'Approved',        variant: 'info' },
    pending:        { label: 'Pending',         variant: 'secondary' },
    rejected:       { label: 'Rejected',        variant: 'destructive' },
    failed:         { label: 'Failed',          variant: 'destructive' },
    completed:      { label: 'Completed',       variant: 'secondary' },
  };
  const { label, variant } = config[status];
  return <Badge variant={variant}>{label}</Badge>;
}

function ExpiryBar({ cert }: { cert: CertificateRequest }) {
  if (!cert.issued_at || !cert.expires_at || cert.validity_days === 0) return null;

  const issued = new Date(cert.issued_at).getTime();
  const expires = new Date(cert.expires_at).getTime();
  const now = Date.now();
  const total = expires - issued;
  const elapsed = Math.min(Math.max(now - issued, 0), total);
  const pct = (elapsed / total) * 100;

  const daysRemaining = (expires - now) / (1000 * 60 * 60 * 24);
  const isExpired = now >= expires;

  let fillColor = 'bg-green-500';
  if (isExpired) fillColor = 'bg-destructive';
  else if (daysRemaining <= 30) fillColor = 'bg-yellow-500';

  return (
    <div className="flex items-center gap-2 min-w-[140px]">
      <div className="relative h-2 flex-1 rounded-full bg-muted overflow-hidden">
        <div
          className={`absolute left-0 top-0 h-full rounded-full transition-all ${fillColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs text-muted-foreground whitespace-nowrap">
        {isExpired
          ? 'Expired'
          : daysRemaining < 1
            ? '<1d'
            : `${Math.floor(daysRemaining)}d`}
      </span>
    </div>
  );
}

function StatCard({ label, value, accent }: { label: string; value: number; accent?: string }) {
  return (
    <Card className="flex-1 rounded-md py-3">
      <CardContent className="px-4">
        <p className="text-xs font-medium text-muted-foreground mb-1">{label}</p>
        <span className={`text-2xl font-bold ${accent ?? ''}`}>{value}</span>
      </CardContent>
    </Card>
  );
}

function RouteComponent() {
  const navigate = useNavigate();
  const { data: requests, isLoading, isError, error } = useMyCertificateRequests();
  const [downloadDialogOpen, setDownloadDialogOpen] = useState(false);
  const [selectedCertificateId, setSelectedCertificateId] = useState<number | null>(null);

  if (isLoading) {
    return (
      <div className="p-6">
        <div className="text-center py-12">Loading...</div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="p-6">
        <div className="text-center py-12 text-destructive">
          Error loading certificates: {error?.message}
        </div>
      </div>
    );
  }

  const certs = Array.isArray(requests) ? requests : [];

  const active = certs.filter((c) => getDisplayStatus(c) === 'active').length;
  const pending = certs.filter((c) =>
    ['awaiting_review', 'approved', 'pending'].includes(c.status)
  ).length;
  const expiringSoon = certs.filter((c) => getDisplayStatus(c) === 'expiring_soon').length;
  const expired = certs.filter((c) => getDisplayStatus(c) === 'expired').length;

  return (
    <div className="p-6">
      <div className="mb-3 flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold mb-1">My Certificates</h1>
          <p className="text-muted-foreground">
            Client certificates issued to you and your service accounts.
          </p>
        </div>
        <Button onClick={() => navigate({ to: '/user/request-certificate' })}>
          Request Certificate
        </Button>
      </div>
      <Separator className="mb-6" />

      <div className="flex gap-4 mb-6">
        <StatCard label="Active" value={active} accent="text-green-600 dark:text-green-400" />
        <StatCard label="Pending" value={pending} />
        <StatCard label="Expiring Soon" value={expiringSoon} accent="text-yellow-600 dark:text-yellow-400" />
        <StatCard label="Expired" value={expired} accent="text-destructive" />
      </div>

      <Card className="rounded-md py-0">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Serial Number</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Lifetime</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {certs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center py-12 text-muted-foreground">
                    No certificates found.{' '}
                    <button
                      className="underline underline-offset-2"
                      onClick={() => navigate({ to: '/user/request-certificate' })}
                    >
                      Request one.
                    </button>
                  </TableCell>
                </TableRow>
              ) : (
                certs.map((cert) => {
                  const displayStatus = getDisplayStatus(cert);
                  return (
                    <TableRow key={cert.id}>
                      <TableCell>
                        <span className="font-mono text-sm">
                          {cert.serial_number ?? <span className="text-muted-foreground">—</span>}
                        </span>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={displayStatus} />
                      </TableCell>
                      <TableCell>
                        {cert.issued_at && cert.expires_at ? (
                          <ExpiryBar cert={cert} />
                        ) : (
                          <span className="text-muted-foreground text-xs">—</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          {displayStatus === 'active' || displayStatus === 'expiring_soon' ? (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                setSelectedCertificateId(cert.id);
                                setDownloadDialogOpen(true);
                              }}
                            >
                              Download
                            </Button>
                          ) : null}
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {selectedCertificateId !== null && (
        <DownloadCertificateDialog
          certificateId={selectedCertificateId}
          open={downloadDialogOpen}
          onOpenChange={setDownloadDialogOpen}
        />
      )}
    </div>
  );
}

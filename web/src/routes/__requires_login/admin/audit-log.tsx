import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/__requires_login/admin/audit-log')({
  component: () => <div className="p-4">hello from /admin/audit-log</div>,
});

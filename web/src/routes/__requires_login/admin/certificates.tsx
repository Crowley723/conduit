import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/__requires_login/admin/certificates')({
  component: () => <div className="p-4">hello from /admin/certificates</div>,
});

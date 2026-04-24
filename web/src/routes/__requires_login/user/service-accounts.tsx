import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/__requires_login/user/service-accounts')({
  component: () => <div className="p-4">hello from /user/service-accounts</div>,
});

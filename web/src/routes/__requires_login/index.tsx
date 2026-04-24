import { createFileRoute, redirect } from '@tanstack/react-router';
import { authRedirect } from '@/services/Auth';

export const Route = createFileRoute('/__requires_login/')({
  beforeLoad: () => {
    const saved = authRedirect.get();
    if (saved && saved !== '/') {
      authRedirect.clear();
      throw redirect({ to: saved });
    }
  },
  component: () => <div className="p-4">Dashboard</div>,
});

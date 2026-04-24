import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/__requires_login/user/activity')({
  component: () => <div className="p-4">hello from /user/activity</div>,
});

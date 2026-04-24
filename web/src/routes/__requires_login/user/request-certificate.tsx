import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { requireAuth } from '@/utils/Auth.ts';
import { useCreateCertificateRequest } from '@/api/Certificates';
import { RequestCertificateForm } from '@/components/RequestCertificateForm';
import { Separator } from '@/components/ui/separator';

export const Route = createFileRoute('/__requires_login/user/request-certificate')({
  component: RouteComponent,
  beforeLoad: async ({ location }) => {
    await requireAuth(location, 'You must login to request a certificate.', true);
  },
});

function RouteComponent() {
  const navigate = useNavigate();
  const { mutate, isPending, error, isSuccess } = useCreateCertificateRequest();

  const handleSubmit = (data: { message: string; validity_days: number }) => {
    mutate(data, {
      onSuccess: () => {
        navigate({ to: '/user/my-certificates' });
      },
    });
  };

  return (
    <div className="p-6">
      <div className="mb-4">
        <h1 className="text-3xl font-bold mb-2">Request Certificate</h1>
        <p className="text-muted-foreground">
          Submit a request for a new mTLS client certificate. An admin will review before issuance.
        </p>
      </div>
      <Separator className="mb-6" />

      <div className="max-w-lg">
        <RequestCertificateForm
          onSubmit={handleSubmit}
          isLoading={isPending}
          errorMessage={error?.message}
          successMessage={isSuccess ? 'Request submitted successfully!' : undefined}
        />
      </div>
    </div>
  );
}

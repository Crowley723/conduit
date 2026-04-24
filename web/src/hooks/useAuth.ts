import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Login,
  Logout,
  GetLoginStatus,
  type AuthResponse,
} from '@/services/Auth';

export const useAuth = () => {
  const queryClient = useQueryClient();

  const { data: authResponse, isLoading } = useQuery<AuthResponse | null>({
    queryKey: ['auth', 'user'],
    queryFn: GetLoginStatus,
    retry: false,
  });

  const user = authResponse?.user || null;
  const isAuthenticated = authResponse?.authenticated || false;
  const config = authResponse?.config;

  const loginMutation = useMutation<void, Error, string | undefined>({
    mutationFn: (redirectTo?: string) => {
      Login(redirectTo);
      return Promise.resolve();
    },
  });

  const logoutMutation = useMutation<void, Error>({
    mutationFn: Logout,
    onSuccess: () => {
      queryClient.setQueryData(['auth', 'user'], null);
    },
  });

  const isInGroup = (groupName: string): boolean => {
    return user?.groups?.includes(groupName) ?? false;
  };

  const isMTLSAdmin = (): boolean => {
    if (!config?.mtls?.enabled) return false;
    return isInGroup('conduit:mtls:admin');
  };

  const isMTLSUser = (): boolean => {
    if (!config?.mtls?.enabled) return false;
    return isInGroup('conduit:mtls:user');
  };

  return {
    user,
    isLoading,
    isAuthenticated,
    config,
    isInGroup,
    isMTLSAdmin,
    isMTLSUser,
    login: loginMutation.mutate,
    logout: logoutMutation.mutate,
    isLoggingIn: loginMutation.isPending,
    isLoggingOut: logoutMutation.isPending,
  } as const;
};

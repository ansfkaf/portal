// src/auth/hooks/use-is-admin.ts
import { useAuthUser } from './use-auth-user';

// 使用此 hook 来获取当前用户是否为管理员
export function useIsAdmin(): boolean {
  const { user } = useAuthUser();
  return user.isAdmin === 1;
}
// src/auth/guard/role-based-route.tsx
import { Navigate } from 'react-router';
import { useAuthUser } from '../hooks/use-auth-user';
import { paths } from 'src/routes/paths';
import { RoleBasedGuard } from './role-based-guard';

type RoleBasedRouteProps = {
  children: React.ReactNode;
  requiredRole: 'admin' | 'user' | 'all';
};

export function RoleBasedRoute({ children, requiredRole }: RoleBasedRouteProps) {
  const { user } = useAuthUser();
  const userRole = user.isAdmin === 1 ? 'admin' : 'user';

  // 如果路由对所有人开放，直接渲染
  if (requiredRole === 'all') {
    return <>{children}</>;
  }

  // 如果需要管理员权限但用户不是管理员，显示权限拒绝页面
  if (requiredRole === 'admin' && userRole !== 'admin') {
    return (
      <RoleBasedGuard
        hasContent
        currentRole={userRole}
        acceptRoles={['admin']}
      >
        {children}
      </RoleBasedGuard>
    );
  }

  // 其他情况下允许访问
  return <>{children}</>;
}
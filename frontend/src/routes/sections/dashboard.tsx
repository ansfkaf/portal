// src/routes/sections/dashboard.tsx
import type { RouteObject } from 'react-router';
import { Outlet } from 'react-router';
import { lazy, Suspense } from 'react';
import { CONFIG } from 'src/global-config';
import { DashboardLayout } from 'src/layouts/dashboard';
import { LoadingScreen } from 'src/components/loading-screen';
import { AuthGuard } from 'src/auth/guard';
import { RoleBasedRoute } from 'src/auth/guard/role-based-route';
import { usePathname } from '../hooks';

// ----------------------------------------------------------------------
const HomePage = lazy(() => import('src/pages/dashboard/home'));
const AccountPage = lazy(() => import('src/pages/dashboard/account'));
const InstancePage = lazy(() => import('src/pages/dashboard/instance'));
const ImportPage = lazy(() => import('src/pages/dashboard/import'));
const PoolPage = lazy(() => import('src/pages/dashboard/pool'));
const SettingPage = lazy(() => import('src/pages/dashboard/setting'));
const MonitorPage = lazy(() => import('src/pages/dashboard/monitor'));
const UserPage = lazy(() => import('src/pages/dashboard/user')); // 用户管理页面
const DatabasePage = lazy(() => import('src/pages/dashboard/database'));

// 队列管理相关页面
const QueueOutlet = lazy(() => import('src/pages/dashboard/queue')); // 新增队列管理外层组件
const MakeupHistoryPage = lazy(() => import('src/pages/dashboard/queue/makeup-history')); // 补机历史页面
const AccountPoolPage = lazy(() => import('src/pages/dashboard/queue/accountpool')); // 账号池队列页面
const MakeupQueuePage = lazy(() => import('src/pages/dashboard/queue/makeup-queue')); // 新增：补机队列页面

function SuspenseOutlet() {
  const pathname = usePathname();
  return (
    <Suspense key={pathname} fallback={<LoadingScreen />}>
      <Outlet />
    </Suspense>
  );
}

const dashboardLayout = () => (
  <DashboardLayout>
    <SuspenseOutlet />
  </DashboardLayout>
);

// 定义路由配置的类型
interface RouteConfig {
  path?: string;
  element: React.ReactNode;
  requiredRole: 'admin' | 'user' | 'all';
  index?: boolean;
  children?: RouteConfig[]; // 添加children属性支持嵌套路由
}

// 定义路由配置，包含权限信息
export const routeConfig: RouteConfig[] = [
  { path: '', element: <HomePage />, requiredRole: 'all', index: true },
  { path: 'account', element: <AccountPage />, requiredRole: 'admin' },
  { path: 'instance', element: <InstancePage />, requiredRole: 'admin' },
  { path: 'import', element: <ImportPage />, requiredRole: 'admin' },
  { path: 'pool', element: <PoolPage />, requiredRole: 'all' },
  { path: 'setting', element: <SettingPage />, requiredRole: 'all' },
  { path: 'monitor', element: <MonitorPage />, requiredRole: 'all' },
  { path: 'user', element: <UserPage />, requiredRole: 'admin' }, // 用户管理路由
  { path: 'database', element: <DatabasePage />, requiredRole: 'admin' }, // 新增：数据库管理路由
  
  // 使用嵌套路由结构处理队列管理
  { 
    path: 'queue', 
    element: <QueueOutlet />, // 使用Outlet组件
    requiredRole: 'admin',
    children: [
      { path: 'accountpool', element: <AccountPoolPage />, requiredRole: 'admin' },
      { path: 'makeup-history', element: <MakeupHistoryPage />, requiredRole: 'admin' },
      { path: 'makeup-queue', element: <MakeupQueuePage />, requiredRole: 'admin' } // 新增：补机队列路由
    ]
  }
];

// 递归处理路由配置，为每个路由添加权限检查
const processRoutes = (routes: RouteConfig[]): RouteObject[] => {
  return routes.map(route => {
    const processed: RouteObject = {
      path: route.path,
      index: route.index,
      element: <RoleBasedRoute requiredRole={route.requiredRole}>{route.element}</RoleBasedRoute>
    };
    
    // 处理子路由
    if (route.children && route.children.length > 0) {
      processed.children = processRoutes(route.children);
    }
    
    return processed;
  });
};

export const dashboardRoutes: RouteObject[] = [
  {
    path: '',  // 移除 dashboard 前缀
    element: CONFIG.auth.skip ? dashboardLayout() : <AuthGuard>{dashboardLayout()}</AuthGuard>,
    children: processRoutes(routeConfig),
  },
];